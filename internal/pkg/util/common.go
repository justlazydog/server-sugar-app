package util

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// 解析压缩账户信息文件
func ParseCompressAccountFile(filename string) (m map[string]float64, sumBalance float64, err error) {
	rc, err := zip.OpenReader(filename)
	if err != nil {
		err = errors.Wrap(err, "open zip file reader")
		return
	}
	defer rc.Close()

	if len(rc.Reader.File) == 0 || rc.Reader.File[0] == nil {
		err = errors.New("empty zip file")
		return
	}
	f, err := rc.Reader.File[0].Open()
	if err != nil {
		err = errors.Wrap(err, "open zip file")
		return
	}
	defer f.Close()

	r := bufio.NewReader(f)
	m = make(map[string]float64)
	for {
		line, _, err := r.ReadLine()
		if err == nil {
			bs := bytes.Split(line, []byte(","))
			if len(bs) == 2 && len(bs[0]) > 0 && len(bs[1]) > 0 {
				uid := string(bytes.TrimSpace(bs[0]))
				balance, err := strconv.ParseFloat(string(bytes.TrimSpace(bs[1])), 64)
				if err != nil {
					err = errors.Wrap(err, "parse string to float")
					return m, sumBalance, err
				}
				m[uid] = balance
				sumBalance += balance
			} else {
				return m, sumBalance, err
			}
		} else if err == io.EOF {
			break
		} else {
			err = errors.Wrap(err, "read line")
			return m, sumBalance, err
		}
	}
	return
}

func ParseGrowthRateFile(path string) (map[string]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	r := bufio.NewReader(f)
	m := make(map[string]float64)
	for {
		line, _, err := r.ReadLine()
		if err == nil {
			bs := bytes.Split(line, []byte(","))
			if len(bs) == 2 && len(bs[0]) > 0 && len(bs[1]) > 0 {
				uid := string(bytes.TrimSpace(bs[0]))
				growthRate, err := strconv.ParseFloat(string(bytes.TrimSpace(bs[1])), 64)
				if err != nil {
					err = errors.Wrap(err, "parse string to float")
					return m, err
				}
				m[uid] = growthRate
			}
		} else if err == io.EOF {
			break
		} else {
			return m, err
		}
	}
	return m, nil
}

// IM服务
func PostIMServer(url, body string) (data map[string]interface{}, err error) {
	log.Infof("post [%s]", url)
	var (
		req *http.Request
		rsp *http.Response
	)
	for i := 1; i <= 3; i++ {
		var nowTime = time.Now().Unix()
		sign := md5.Sum([]byte(fmt.Sprintf("%s%d%s", body, nowTime, "asdfeaegrgrew&asdfeaegrgrew%asdfeaegrgrew")))
		req, err = http.NewRequest("POST", url, strings.NewReader(body))
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("new request url %s", url))
			return
		}

		req.Header.Set("time", fmt.Sprintf("%d", nowTime))
		req.Header.Set("sign", hex.EncodeToString(sign[:]))
		req.Header.Set("content-type", "application/json")

		client := http.Client{Timeout: time.Minute}
		rsp, err = client.Do(req)
		if err != nil {
			log.Warnf("asd IM err: %v, cnt: %d", err, i)
			time.Sleep(time.Second * time.Duration(i))
			continue
		}

		if rsp.StatusCode != 200 {
			bs, _ := ioutil.ReadAll(rsp.Body)
			rsp.Body.Close()
			log.Warnf("asd IM rsp not ok, cnt: %d, msg: %s", i, string(bs))
			time.Sleep(time.Second * time.Duration(i))
			continue
		}

		break
	}

	if err != nil {
		err = errors.Wrap(err, "ask IM")
		return
	}

	if rsp.StatusCode != 200 {
		rspBody, _ := ioutil.ReadAll(rsp.Body)
		rsp.Body.Close()
		err = errors.New(string(rspBody))
		err = errors.Wrap(err, "IM rsp not ok")
		return
	}

	defer rsp.Body.Close()

	rspBody, _ := ioutil.ReadAll(rsp.Body)

	type Result struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}

	var res Result
	err = json.Unmarshal(rspBody, &res)
	if err != nil {
		err = errors.Wrap(err, "json unmarshal")
		return
	}
	if res.Code != 200 {
		err = errors.Wrap(fmt.Errorf("post [%s] response code is [%d]", url, res.Code), res.Msg)
	}
	data = res.Data
	return
}

// 生成随机字符串
func RandString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	var src = rand.NewSource(time.Now().UnixNano())

	const (
		letterIdxBits = 6
		letterIdxMask = 1<<letterIdxBits - 1
		letterIdxMax  = 63 / letterIdxBits
	)

	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}

// ToString()
func ToString(v interface{}) string {
	var str string
	switch v.(type) {
	case int:
		str = strconv.Itoa(v.(int))
	case int64:
		str = strconv.FormatInt(v.(int64), 10)
	case string:
		str, _ = v.(string)
	case float64:
		str = fmt.Sprintf("%f", v.(float64))
	default:
		str = ""
	}
	return str
}

func GenSignCode(form url.Values, key string) (signCode string) {
	s := make([]string, 0)
	for k := range form {
		if k == "s" {
			continue
		}
		s = append(s, k)
	}
	sort.Strings(s)

	str := ""
	for _, v := range s {
		str += ToString(form[v][0])
	}

	str += key
	//log.Infof("str gen sign: %s", str)
	//fmt.Println(str)
	signCode = fmt.Sprintf("%x", md5.Sum([]byte(str)))
	return
}
