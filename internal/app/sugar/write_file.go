package sugar

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"server-sugar-app/internal/app/client"
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
func writeForceFile(details map[string]*RewardDetail, flag int) (filename string) {
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
		for key, d := range details {
			str := fmt.Sprintf("id: %s, possess_force: %15.6f\n", key, math.Floor(d.BalanceHashRate*Precision)/Precision)
			_, err = f.WriteString(str)
			if err != nil {
				log.Warnf("err: %+v", errors.Wrap(err, "write string"))
				return
			}
		}
	} else {
		for uid, d := range details {
			m := make(map[string]bool)
			users, err := group.GetAllDownLineUsers(uid, m)
			if err != nil {
				log.Warnf("err: %+v", errors.Wrap(err, "get all downline details"))
				return
			}
			teamAmount := details[uid].TodayBal
			for _, u := range users {
				uDetail, ok := details[u]
				if ok {
					teamAmount += uDetail.TodayBal
				}
			}
			str := fmt.Sprintf("id: %s, invite_force: %15.6f, team_amount: %15.6f\n",
				uid, math.Floor(d.InviteHashRate*Precision)/Precision, math.Floor(teamAmount*Precision)/Precision)
			_, err = f.WriteString(str)
			if err != nil {
				log.Warnf("err: %+v", errors.Wrap(err, "write string"))
				return
			}
		}
		// 此时内存中已不需要群用户关系
		group.Flush()
	}
	return filename
}

func writeGrowthRateFile(users map[string]*RewardDetail) (string, error) {
	filename := fmt.Sprintf("%s%s_%s.txt", curFilePath, "growth_rate", time.Now().Format("200601021504"))
	f, err := os.Create(filename)
	if err != nil {
		log.Warnf("err: %+v", errors.Wrap(err, "create file"))
		return filename, err
	}
	defer f.Close()

	for uid, d := range users {
		str := fmt.Sprintf("%s,%d\n", uid, int64(d.GrowthRate))
		_, err = f.WriteString(str)
		if err != nil {
			log.Warnf("err: %+v", errors.Wrap(err, "write string"))
			return filename, err
		}
	}

	return filename, nil
}

func writeLockSIEFile(lockSIE map[string]float64) (string, error) {
	filename := fmt.Sprintf("%s%s_%s.txt", curFilePath, "lockSIE_", time.Now().Format("200601021504"))
	f, err := os.Create(filename)
	if err != nil {
		log.Warnf("err: %+v", errors.Wrap(err, "create file"))
		return filename, err
	}
	defer f.Close()

	for uid, v := range lockSIE {
		str := fmt.Sprintf("%s,%f\n", uid, v)
		_, err = f.WriteString(str)
		if err != nil {
			log.Warnf("err: %+v", errors.Wrap(err, "write string"))
			return filename, err
		}
	}

	return filename, nil
}

func writePledgeFile(pledges []client.DefiPledge) (string, error) {
	filename := fmt.Sprintf("%s%s_%s.txt", curFilePath, "pledge_", time.Now().Format("200601021504"))
	f, err := os.Create(filename)
	if err != nil {
		log.Warnf("err: %+v", errors.Wrap(err, "create file"))
		return filename, err
	}
	defer f.Close()

	for _, pledge := range pledges {
		str := fmt.Sprintf("%s,%s,%s\n", pledge.OpenID, pledge.Token, pledge.Amount)
		_, err = f.WriteString(str)
		if err != nil {
			log.Warnf("err: %+v", errors.Wrap(err, "write string"))
			return filename, err
		}
	}

	return filename, nil
}

// 写奖励文件
func writeRewardFile(details map[string]*RewardDetail) (zipFn []string, err error) {
	if details == nil {
		err = errors.Wrap(fmt.Errorf("details: %v", details), "lack of data")
		return
	}

	t := time.Now().Unix()
	fn1Path := fmt.Sprintf("%sreward_1_%d.txt", curFilePath, t)
	fn2Path := fmt.Sprintf("%sreward_2_%d.txt", curFilePath, t)
	detailPath := fmt.Sprintf("%sdetail_%d.txt", curFilePath, t)
	// BOB要求2个奖励文件单独发送。。。
	zipFn1 := fmt.Sprintf("reward_1_%d.zip", t)
	zipFn2 := fmt.Sprintf("reward_2_%d.zip", t)
	zipDetail := fmt.Sprintf("detail_%d.zip", t)

	if err = writeRewardFileByMap(fn1Path, fn2Path, detailPath, details); err != nil {
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
	if err = util.ZipFiles(curFilePath+zipDetail, []string{detailPath}); err != nil {
		err = errors.Wrap(err, "zip file")
		return
	}
	os.Remove(fn1Path)
	os.Remove(fn2Path)
	os.Remove(detailPath)
	zipFn = []string{zipFn1, zipFn2}
	return
}

// 根据奖励计算结果的map写文件
func writeRewardFileByMap(filename1, filename2, fileDetail string, details map[string]*RewardDetail) (err error) {
	file1, err := os.Create(filename1)
	if err != nil {
		err = errors.Wrap(err, "crate file1")
		return
	}
	defer file1.Close()

	file2, err := os.Create(filename2)
	if err != nil {
		err = errors.Wrap(err, "crate file1")
		return
	}
	defer file2.Close()

	fileD, err := os.Create(fileDetail)
	if err != nil {
		err = errors.Wrap(err, "crate fileD")
		return
	}
	defer fileD.Close()

	var buf1 = bufio.NewWriter(file1)
	var buf2 = bufio.NewWriter(file2)
	var bufD = bufio.NewWriter(fileD)
	var line string
	for uid, d := range details {
		if d.BalanceReward > 0.000000 {
			line = fmt.Sprintf("%s,%f\n", uid, math.Floor(d.BalanceReward*Precision)/Precision)
			buf1.WriteString(line)
		}
		if d.InviteReward > 0.000000 {
			line = fmt.Sprintf("%s,%f\n", uid, math.Floor(d.InviteReward*Precision)/Precision)
			buf2.WriteString(line)
		}
		line = fmt.Sprintf("%s,%f,%f,%f,%f,%f,%f,%f,%f,%f,%f,%f,%f,%s\n",
			uid, d.YesterdayBal, d.TodayBal, d.DestroyHashRate, d.YesterdayGrowthRate, d.GrowthRate,
			d.BalanceHashRateForInvite, d.BalanceHashRate, d.PureBalanceHashRate, d.InviteHashRate, d.BalanceReward,
			d.InviteReward, d.TeamHashRate, d.ParentUID)
		bufD.WriteString(line)
	}
	err1 := buf1.Flush()
	err2 := buf2.Flush()
	err3 := bufD.Flush()
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return err3
}
