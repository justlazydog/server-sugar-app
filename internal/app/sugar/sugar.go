package sugar

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"server-sugar-app/config"
	"server-sugar-app/internal/dao"
	"server-sugar-app/internal/model"
	"server-sugar-app/internal/pkg/util"
)

const (
	FilePath  = "sugar/"
	Precision = 1000000
)

// 计算糖果奖励
func calcReward(files []string) (err error) {
	// mapS 各文件包含的用户账户信息
	mapS := make(map[string]map[string]float64)
	// sumS 存取商户用户、店铺累计销毁金额
	sumS := make([]float64, 2)

	sie := config.SIE
	for i, filename := range files {
		key := strings.Split(filename, "_")[0]
		m, sum, err := util.ParseCompressAccountFile(FilePath + filename)
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

	// reward2M 糖果邀请奖励商城积分计算依据
	reward2M := cloneMap(creditM)
	for key := range shopM {
		reward2M[key] += shopM[key]
	}

	// 获取销毁SIE数量累计值
	shopSIE, err := getUsedShopSIE()
	if err != nil {
		return errors.Wrap(err, "get used shop sie amount")
	}

	// 计算应发糖果金额
	curSugar := shopSIE * 0.01

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

	lastSugar, err := dao.Sugar.GetLastRecord()
	if err != nil {
		return errors.Wrap(err, "get last sugar record")
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

	// 生成奖励数据文件
	rewardFiles, err := writeRewardFile(r1, r2)
	if err != nil {
		return errors.Wrap(err, "write reward file")
	}

	// 通知IM下载文件
	if err := noticeIMDownloadRewardFile(rewardFiles); err != nil {
		return errors.Wrap(err, "notice IM server download reward file")
	}

	// todo
	// 当前糖果计算各用户信息更新进数据库
	// go func() {
	// 	ua := make([]UserSugar, 0)
	// 	tx, err := db.GetMysql().Begin()
	// 	defer tx.Commit()
	// 	if err != nil {
	// 		return
	// 	}
	//
	// 	index := 1
	// 	for _, key := range Users {
	// 		u := UserSugar{
	// 			UID:              key,
	// 			UserSugarAmount:  r1[key] + r2[key],
	// 			UserPossessForce: r1f[key],
	// 			UserInviteForce:  r2f[key],
	// 			UserAmount:       propertyM[key],
	// 			// UserFrozen:       frozenM[key],
	// 			UserCredit: creditM[key],
	// 		}
	//
	// 		// 过滤皆为0的数据
	// 		tmp := UserSugar{UID: key}
	// 		if !(u == tmp) {
	// 			ua = append(ua, u)
	// 		}
	//
	// 		// 批量插入
	// 		if index == len(Users) || len(ua) > 499 {
	// 			tmp := UserSugar{}
	// 			if err = tmp.CreateWithTx(tx, ua); err != nil {
	// 				tx.Rollback()
	// 				return
	// 			}
	// 			ua = make([]UserSugar, 0)
	// 		}
	// 		index++
	// 	}
	// }()
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

// todo
// 获取最新SIE销毁量
func getUsedShopSIE() (usedSIE float64, err error) {
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
		rsp, err := util.PostIMServer("https://account.isecret.im/open/wallet/user/GetBillSummary", string(body))
		if err != nil {
			return accountMap, sumBalance, err
		}

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
			return accountMap, sumBalance, err
		}

		if rspMap, ok := res.Data["SIE"].(map[string]interface{}); ok {
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
	var (
		callBackUrl string
		postUrl     string
	)
	for i, filename := range filenames {
		if config.Server.Env == "pro" {
			callBackUrl = "https://open.isecret.im" + "/manager/exchange/reward/download/" + filename
			postUrl = "https://account.isecret.im" + "/open/SieGame/UpdateBalaneFromFile"
		} else {
			callBackUrl = "https://testm.isecret.im" + "/manager/exchange/reward/download/" + filename
			postUrl = "https://accounttest.isecret.im" + "/open/SieGame/UpdateBalaneFromFile"
		}

		postBody := fmt.Sprintf(`{"callback":"%s","type":%d}`, callBackUrl, i+1)
		_, err = util.PostIMServer(postUrl, postBody)
		if err != nil {
			return
		}
		time.Sleep(time.Millisecond * 500)
	}
	return nil
}
