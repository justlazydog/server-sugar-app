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
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
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

func PostIMServer(url, body string) (rsp *http.Response, err error) {
	var (
		req *http.Request
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

		rsp, err = http.DefaultClient.Do(req)
		if err != nil || rsp.StatusCode != 200 {
			time.Sleep(time.Second * time.Duration(i))
			continue
		} else {
			break
		}
	}
	if err == nil && rsp.StatusCode != 200 {
		// 获取状态码非200原因
		rspBody, _ := ioutil.ReadAll(rsp.Body)
		rsp.Body.Close()

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
		err = errors.Wrap(fmt.Errorf("post [%s] response code is [%d]", url, rsp.StatusCode), res.Msg)
		return
	}
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
