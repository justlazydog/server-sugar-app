package sugar

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/internal/app/group"
	"server-sugar-app/internal/pkg/util"
)

// 写账户差值文件
func writeFile(user map[string]float64, flag int) {
	var filename string

	if flag == 1 {
		filename = fmt.Sprintf("%s%s_%s.txt", curFilePath, "account_in", time.Now().Format("20060102150405"))
	} else {
		filename = fmt.Sprintf("%s%s_%s.txt", curFilePath, "account_out", time.Now().Format("20060102150405"))
	}

	f, err := os.Create(filename)
	if err != nil {
		log.Warnf("err: %+v", errors.Wrap(err, "create file"))
		return
	}
	defer f.Close()

	for key, value := range user {
		str := fmt.Sprintf("%s,%f\n", key, math.Floor(value*1000000)/1000000)
		_, err = f.WriteString(str)
		if err != nil {
			log.Warnf("err: %+v", errors.Wrap(err, "write string"))
			return
		}
	}
}

// 写算力文件
func writeForceFile(user map[string]float64, amount map[string]float64, flag int) (filename string) {
	if flag == 1 {
		filename = fmt.Sprintf("%s%s_%s.txt", curFilePath, "possess", time.Now().Format("200601021504"))
	} else {
		filename = fmt.Sprintf("%s%s_%s.txt", curFilePath, "invite", time.Now().Format("200601021504"))
	}

	f, err := os.Create(filename)
	if err != nil {
		log.Warnf("err: %+v", errors.Wrap(err, "create file"))
		return
	}
	defer f.Close()

	if flag == 1 {
		for key, value := range user {
			str := fmt.Sprintf("id: %s, possess_force: %15.6f\n", key, math.Floor(value*Precision)/Precision)
			_, err = f.WriteString(str)
			if err != nil {
				log.Warnf("err: %+v", errors.Wrap(err, "write string"))
				return
			}
		}
	} else {
		for key, value := range user {
			m := make(map[string]bool)
			users, err := group.GetAllDownLineUsers(key, m)
			if err != nil {
				log.Warnf("err: %+v", errors.Wrap(err, "get all downline users"))
				return
			}
			teamAmount := amount[key]
			for _, u := range users {
				teamAmount += amount[u]
			}
			str := fmt.Sprintf("id: %s, invite_force: %15.6f, team_amount: %15.6f\n",
				key, math.Floor(value*Precision)/Precision, math.Floor(teamAmount*Precision)/Precision)
			_, err = f.WriteString(str)
			if err != nil {
				log.Warnf("err: %+v", errors.Wrap(err, "write string"))
				return
			}
		}
	}
	return filename
}

// 写奖励文件
func writeRewardFile(r1 map[string]float64, r2 map[string]float64) (zipFn []string, err error) {
	if r1 == nil || r2 == nil {
		err = errors.Wrap(fmt.Errorf("r1: %v, r2: %v", r1, r2), "lack of data")
		return
	}

	t := time.Now().Unix()
	fn1Path := fmt.Sprintf("%sreward_1_%d.txt", curFilePath, t)
	fn2Path := fmt.Sprintf("%sreward_2_%d.txt", curFilePath, t)
	// BOB要求2个奖励文件单独发送。。。
	zipFn1 := fmt.Sprintf("reward_1_%d.zip", t)
	zipFn2 := fmt.Sprintf("reward_2_%d.zip", t)

	if err = writeRewardFileByMap(fn1Path, r1); err != nil {
		err = errors.Wrap(err, "write reward file by map")
		return
	}
	if err = writeRewardFileByMap(fn2Path, r2); err != nil {
		err = errors.Wrap(err, "write reward file by map")
		return
	}
	// 压缩失败，传txt文件
	// 如果fn带路径的话，压缩文件也会有路径
	if err = util.ZipFiles(curFilePath+zipFn1, []string{fn1Path}); err != nil {
		err = errors.Wrap(err, "zip file")
		return
	}
	if err = util.ZipFiles(curFilePath+zipFn2, []string{fn2Path}); err != nil {
		err = errors.Wrap(err, "zip file")
		return
	}
	os.Remove(fn1Path)
	os.Remove(fn2Path)
	zipFn = []string{zipFn1, zipFn2}
	return
}

// 根据奖励计算结果的map写文件
func writeRewardFileByMap(filename string, m map[string]float64) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		err = errors.Wrap(err, "crate file")
		return
	}
	defer file.Close()

	var buf = bufio.NewWriter(file)
	var line string
	for k, v := range m {
		if v > 0 {
			line = fmt.Sprintf("%s,%f\n", k, math.Floor(v*Precision)/Precision)
			_, err = buf.WriteString(line)
			if err != nil {
				continue
			}
		}
	}
	return buf.Flush()
}
