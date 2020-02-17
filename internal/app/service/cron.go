package service

import (
	"crypto/md5"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
	"server-sugar-app/internal/app/group"
	"server-sugar-app/internal/app/sugar"
	"server-sugar-app/internal/pkg/util"
)

func SugarTicker() {
	c := cron.New()
	sieCfg := config.SIE
	_ = c.AddFunc(sieCfg.SIESchedule, StartSugar)
	_ = c.AddFunc(sieCfg.SIESchedule, GetLatestGroupRela)
	c.Start()
}

func GetLatestGroupRela() {
	defer func() {
		group.RelateUpdated = true
		group.Cond.Broadcast()
	}()

	t := time.Now()
	group.RelateUpdated = false
	group.StopCalc = false
	log.Info("start update relation")
	err := group.UpdateGroupRelation()
	if err != nil {
		group.StopCalc = true
		log.Errorf("err: %+v", errors.WithMessage(err, "update relation"))
		return
	}
	m := make(map[string]bool)
	users, err := group.GetAllDownLineUsers(group.SystemAccount, m)
	if err != nil {
		group.StopCalc = true
		log.Errorf("err: %+v", errors.WithMessage(err, "get all down line users"))
		return
	}
	users = append(users, group.SystemAccount)
	group.Users = users
	log.Infof("relation updated, cost time: %v", time.Since(t))
}

func StartSugar() {
	// 每次发放糖果将文件置于新文件夹中
	dirname := time.Now().Format("2006-01-02") + "/"
	err := os.Mkdir(sugar.FilePath+dirname, 0755)
	if err != nil {
		log.Errorf("err: %+v", errors.WithMessage(err, "create sugar dir"))
		return
	}

	sieCfg := config.SIE
	token := md5.Sum([]byte(util.RandString(16) + fmt.Sprintf("%d", time.Now().Unix())))
	sugar.ExpectToken = string(token[:])
	for _, v := range sieCfg.Sugars {
		callBackUrl := config.Server.DomainName + "/abc/" + sugar.ExpectToken + "/" + v.Origin
		t, _ := time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02")+" 16:00:00")
		postBody := fmt.Sprintf(`{"callback":"%s","timestamp":%d}`, callBackUrl, t.Unix())
		_, err = util.PostIMServer(v.Request, postBody)
		if err != nil {
			log.Errorf("err: %+v", errors.WithMessage(err, "post im server"))
			return
		}
	}
}

// // 周期执行
// func GetCurrentUserSIEBal() {
// 	go StartSugar()
// 	sieCfg := config.SIE
// 	calcPerf.files = []string{}
// 	for _, perf := range sieCfg.Perfs {
// 		callBackUrl := config.GetHostUrl(config.ReceivePerfFile) + "/" + perf.Origin
// 		t, _ := time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02")+" 16:00:00")
// 		postBody := fmt.Sprintf(`{"callback":"%s","timestamp":%d}`, callBackUrl, t.Unix())
// 		code, msg, _ := exchange.HttpPostIM(perf.Request, postBody)
// 		if code != http.StatusOK {
// 			log.Errorf("post request to IM for account file file failed, msg: %s", msg)
// 			return
// 		}
// 	}
// 	return
// }
//
// func StartSugar() {
// 	// 每次发放糖果将文件置于新文件夹中
// 	dirname := time.Now().Format("2006-01-02") + "/"
// 	err := os.Mkdir(exchange.SugarFilePath+dirname, 0755)
// 	if err != nil {
// 		log.Errorf("create sugar file dir err: %v", err)
// 		return
// 	}
// 	exchange.SugarFileDir = exchange.SugarFilePath + dirname
//
// 	go func() {
// 		sie := config.GetSIE()
// 		calcSugar.files = []string{}
// 		token := md5.Sum([]byte(RandString(16) + fmt.Sprintf("%d", time.Now().Unix())))
// 		expectToken = fmt.Sprintf("%x", token)
// 		for _, sugar := range sie.Sugar {
// 			callBackUrl := config.GetHostUrl(config.ReceiveSugarFile) + expectToken + "/" + sugar.Origin
// 			t, _ := time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02")+" 16:00:00")
// 			postBody := fmt.Sprintf(`{"callback":"%s","timestamp":%d}`, callBackUrl, t.Unix())
// 			code, msg, _ := exchange.HttpPostIM(sugar.Request, postBody)
// 			if code != http.StatusOK {
// 				log.Errorf("post request to IM for account file file failed, msg: %s", msg)
// 				return
// 			}
// 		}
// 	}()
// }
