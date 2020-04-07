package sugar

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"server-sugar-app/config"
	"server-sugar-app/internal/dao"
	"server-sugar-app/internal/db"
	"server-sugar-app/internal/model"
	"server-sugar-app/internal/pkg/util"
)

const (
	Precision = 1000000
)

var curFilePath string

func StartSugar() {
	log.Info("start sugar")
	// 每次发放糖果将文件置于新文件夹中
	dirname := time.Now().Format("2006-01-02") + "/"
	curFilePath = "sugar/" + dirname
	err := os.MkdirAll(curFilePath, 0755)
	if err != nil {
		log.Errorf("err: %+v", errors.WithMessage(err, "create sugar dir"))
		return
	}

	sieCfg := config.SIE
	token := md5.Sum([]byte(util.RandString(16) + fmt.Sprintf("%d", time.Now().Unix())))
	expectToken = fmt.Sprintf("%x", token)
	for _, v := range sieCfg.Sugars {
		// callBackUrl := config.Server.DomainName + "/sugar/upload/" + ExpectToken + "/" + v.Origin
		callBackUrl := fmt.Sprintf("%s/sugar/upload/%s/%s", config.Server.DomainName, expectToken, v.Origin)
		t, _ := time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02")+" 16:00:00")
		postBody := fmt.Sprintf(`{"callback":"%s","timestamp":%d}`, callBackUrl, t.Unix())
		_, err = util.PostIMServer(v.Request, postBody)
		if err != nil {
			log.Errorf("err: %+v", errors.WithMessage(err, "post im server"))
			return
		}
	}
}

// 计算糖果奖励
func calcReward(files []string) (err error) {
	// mapS 各文件包含的用户账户信息
	mapS := make(map[string]map[string]float64)
	// sumS 存取商户用户、店铺累计销毁金额
	sumS := make([]float64, 2)

	sie := config.SIE
	for i, filename := range files {
		key := strings.Split(filename, "_")[0]
		m, sum, err := util.ParseCompressAccountFile(curFilePath + filename)
		if err != nil {
			return errors.Wrap(err, "parse compress file")
		}
		if key == sie.Sugars[1].Origin {
			sumS[i-1] = sum
		}
		if key == sie.Sugars[2].Origin {
			sumS[i-1] = sum
		}
		mapS[key] = m
	}

	// 糖果奖励所需计算数据
	// 用户持币文件
	propertyM := mapS[sie.Sugars[0].Origin]
	// 用户商城积分文件
	creditM := mapS[sie.Sugars[1].Origin]
	// 用户商城店铺积分文件
	shopM := mapS[sie.Sugars[2].Origin]

	// creditM-上次持币奖励*10/2
	rewardA, err := dao.UserReward.Get()
	if err != nil {
		return errors.Wrap(err, "get user possess reward")
	}

	// reward2M 糖果邀请奖励商城积分计算依据
	reward2M := cloneMap(creditM)
	for key := range shopM {
		reward2M[key] += shopM[key]
	}

	for k, v := range rewardA {
		creditM[k] = creditM[k] - v*10/2
		reward2M[k] = reward2M[k] - v*10/2
		if reward2M[k] < 0 {
			reward2M[k] = 0
		}
	}

	// 获取销毁SIE数量累计值
	shopSIE, err := getUsedShopSIE()
	if err != nil {
		return errors.Wrap(err, "get used shop sie amount")
	}

	sumRewardA, err := dao.UserReward.GetRewardA()
	if err != nil {
		return errors.Wrap(err, "get sum possess reward")
	}

	// 计算应发糖果金额
	curSugar := (shopSIE - sumRewardA/2) * 0.0048

	// 持币奖励结果保存的map
	r1 := make(map[string]float64)
	// 持币奖励用户算力
	r1f := make(map[string]float64)
	// 邀请奖励结果保存的map
	r2 := make(map[string]float64)
	// 邀请奖励结用户算力
	r2f := make(map[string]float64)

	var g errgroup.Group
	g.Go(func() error {
		return rewardOne([]map[string]float64{propertyM, creditM, shopM}, r1, r1f, curSugar/2)
	})

	g.Go(func() error {
		return rewardTwo(propertyM, reward2M, r2, r2f, curSugar/2)
	})
	err = g.Wait()
	if err != nil {
		return errors.Wrap(err, "goroutine wait")
	}

	accInMap, sumBalanceIn, err := getAccountsBalanceInOrOut(sie.SIEAddAccounts, 1)
	if err != nil {
		return errors.Wrap(err, "get account balance in or out")
	}
	go writeFile(accInMap, 1)

	accOutMap, sumBalanceOut, err := getAccountsBalanceInOrOut(sie.SIESubAccounts, 2)
	if err != nil {
		return errors.Wrap(err, "get account balance in or out")
	}
	go writeFile(accOutMap, 2)

	lastSugar, err := dao.Sugar.GetLastRecord()
	if err != nil {
		return errors.Wrap(err, "get last sugar record")
	}

	curCurrency := lastSugar.RealCurrency - (sumBalanceOut - lastSugar.AccountOut) - (shopSIE - lastSugar.ShopSIE) - (sumBalanceIn - lastSugar.AccountIn)
	curRealCurrency := curCurrency + curSugar

	newSugar := model.Sugar{
		Sugar:        curSugar,
		Currency:     curCurrency,
		RealCurrency: curRealCurrency,
		ShopSIE:      shopSIE,
		ShopUsedSIE:  shopSIE - lastSugar.ShopSIE,
		AccountIn:    sumBalanceIn,
		AccountOut:   sumBalanceOut,
	}
	err = dao.Sugar.Create(newSugar)
	if err != nil {
		return errors.Wrap(err, "add sugar record")
	}

	// save(持币奖励)
	log.Info("start save user_reward")
	t := time.Now()
	ur := make([]model.UserReward, 0)
	tx, err := db.MysqlCli.Begin()
	if err != nil {
		return errors.Wrap(err, "tx begin")
	}

	index := 0
	for user, amount := range r1 {
		if amount > 0.000000 {
			u := model.UserReward{
				UID:     user,
				RewardA: math.Floor(amount*Precision) / Precision,
			}
			ur = append(ur, u)
		}

		// 批量插入
		if index == len(r1)-1 || len(ur) > 499 {
			err = dao.UserReward.CreateWithTx(tx, ur)
			if err != nil {
				tx.Rollback()
				return errors.Wrap(err, "user reward insert")
			}
			ur = make([]model.UserReward, 0)
		}
		index++
	}
	tx.Commit()
	log.Infof("over save user_reward, cost time: %v", time.Since(t))

	// 生成奖励数据文件
	rewardFiles, err := writeRewardFile(r1, r2)
	if err != nil {
		return errors.Wrap(err, "write reward file")
	}

	// fmt.Println(rewardFiles)
	// 通知IM下载文件
	if err := noticeIMDownloadRewardFile(rewardFiles); err != nil {
		return errors.Wrap(err, "notice IM server download reward file")
	}
	return
}

// 复制map，避免混淆引用
func cloneMap(m map[string]float64) map[string]float64 {
	tmp := make(map[string]float64)
	for key, value := range m {
		tmp[key] = value
	}
	return tmp
}

// 获取最新SIE销毁量
func getUsedShopSIE() (usedSIE float64, err error) {
	off, on, err := dao.User.GetUsedAmount()
	if err != nil {
		return
	}
	usedSIE = off + on
	return
}

// 获取特定账户差值信息
func getAccountsBalanceInOrOut(accounts []string, flag int) (
	accountMap map[string]float64, sumBalance float64, err error) {
	accountMap = make(map[string]float64)
	for _, account := range accounts {
		m := map[string]string{
			"uid":  account,
			"coin": "SIE",
		}

		body, _ := json.Marshal(m)
		data, err := util.PostIMServer("https://account.isecret.im/open/wallet/user/GetBillSummary", string(body))
		if err != nil {
			return accountMap, sumBalance, err
		}

		if rspMap, ok := data["SIE"].(map[string]interface{}); ok {
			var balance float64
			if flag == 1 {
				if _, ok := rspMap["balance_in"].(string); ok {
					balance, err = strconv.ParseFloat(rspMap["balance_in"].(string), 64)
					if err != nil {
						return accountMap, sumBalance, err
					}
				}
			} else if flag == 2 {
				if _, ok := rspMap["balance_out"].(string); ok {
					balance, err = strconv.ParseFloat(rspMap["balance_out"].(string), 64)
					if err != nil {
						return accountMap, sumBalance, err
					}
				}
			}
			sumBalance += balance
			accountMap[account] = balance
		}
	}
	return
}

// 通知IM来下载奖励文件
func noticeIMDownloadRewardFile(filenames []string) (err error) {
	log.Info("start notice IM to download reward file")
	var callBackUrl, postUrl string
	for i, filename := range filenames {
		if config.Server.Env == "pro" {
			callBackUrl = config.Server.DomainName + "/sugar/download/" + filename
			postUrl = "https://account.isecret.im" + "/open/SieGame/UpdateBalaneFromFile"
		} else {
			callBackUrl = config.Server.DomainName + "/sugar/download/" + filename
			postUrl = "https://accounttest.isecret.im" + "/open/SieGame/UpdateBalaneFromFile"
		}

		postBody := fmt.Sprintf(`{"callback":"%s","type":%d}`, callBackUrl, i+1)
		_, err = util.PostIMServer(postUrl, postBody)
		if err != nil {
			return
		}
		time.Sleep(time.Millisecond * 500)
	}
	return
}
