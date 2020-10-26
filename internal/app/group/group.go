package group

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const SystemAccount = "a3b640a1-1de0-4fe8-9b57-b3985256efb2"

var (
	RelateUpdated bool
	StopCalc      bool
	Users         []string
	Cond          = sync.NewCond(new(sync.Mutex))
)

var userRelaMap map[string][]string

func GetLatestGroupRela() {
	defer func() {
		RelateUpdated = true
		Cond.Broadcast()
	}()

	t := time.Now()
	RelateUpdated = false
	StopCalc = false
	log.Info("start update relation")
	err := UpdateGroupRelation()
	if err != nil {
		StopCalc = true
		log.Errorf("err: %+v", errors.WithMessage(err, "update relation"))
		return
	}
	m := make(map[string]bool)
	users, err := GetAllDownLineUsers(SystemAccount, m)
	if err != nil {
		StopCalc = true
		log.Errorf("err: %+v", errors.WithMessage(err, "get all down line users"))
		return
	}
	users = append(users, SystemAccount)
	Users = users
	log.Infof("relation updated, cost time: %v", time.Since(t))
}

// UpdateGroupRelation 更新最新的IM用户关系
func UpdateGroupRelation() (err error) {
	userRelaMap = make(map[string][]string)
	// 从IM获取最新的用户关系
	param := map[string]interface{}{
		"code": "secret",
	}
	reqBody, _ := json.Marshal(param)
	sign := md5.Sum([]byte(param["code"].(string) + "asdfeaegrgrew&asdfeaegrgrew%asdfeaegrgrew"))
	// 测试环境也用正式的关系数据测
	req, err := http.NewRequest("POST", "https://account.isecret.im/open/bind/secret/userref/safe",
		bytes.NewReader(reqBody))
	if err != nil {
		err = errors.Wrap(err, "new request")
		return
	}
	defer req.Body.Close()

	req.Header.Add("sign", fmt.Sprintf("%x", sign))
	req.Header.Add("Content-Type", "application/json")

	//proxyUrl, _ := url.Parse("http://127.0.0.1:1087")
	client := &http.Client{
		Transport: &http.Transport{
			//Proxy:           http.ProxyURL(proxyUrl),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: time.Minute,
	}

	var rsp *http.Response
	for i := 0; i < 3; i++ {
		rsp, err = client.Do(req)
		if err != nil {
			log.Warnf("ask IM err: %v, ask cnt: %d", err, i+1)
			time.Sleep(time.Millisecond * 200)
			continue
		}

		if rsp.StatusCode != 200 {
			rsp.Body.Close()
			log.Warnf("IM rsp code not ok, ask cnt: %d", err, i+1)
			time.Sleep(time.Millisecond * 200)
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
	rels := make([]map[string]string, 0)
	err = json.Unmarshal(rspBody, &rels)
	if err != nil {
		//fmt.Println(string(rspBody))
		err = errors.Wrap(err, "json unmarshal")
		return
	}

	// 将关系写进UserRelaMap
	for i := len(rels); i > 0; i-- {
		var father, son string
		for son, father = range rels[i-1] {
		}
		if father == son {
			continue
		}

		userRelaMap[father] = append(userRelaMap[father], son)
	}
	return
}

func TimeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, nil
	}
}

// GetAllDownLineUsers 获取当前用户的所有下线
func GetAllDownLineUsers(uid string, cm map[string]bool) (ids []string, err error) {
	ids = make([]string, 0)
	users := GetDownLineUsers(uid)
	if len(users) == 0 {
		return ids, nil
	}
	ids = append(ids, users...)
	for _, user := range users {
		if _, ok := cm[user]; ok {
			err = errors.Errorf("dirty user data cause circle in relation, uid: %s", user)
			return
		}
		cm[user] = true
		subs, err := GetAllDownLineUsers(user, cm)
		if err != nil {
			return ids, err
		}
		ids = append(ids, subs...)
	}
	return
}

// GetDownLineUsers 获取当前用户的直接下线
func GetDownLineUsers(uid string) []string {
	ids := make([]string, 0)
	ids = append(ids, userRelaMap[uid]...)
	return ids
}

func Flush() {
	userRelaMap = make(map[string][]string)
	Users = []string{}
	return
}
