package sugar

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"server-sugar-app/internal/app/group"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
	"server-sugar-app/internal/dao"
	"server-sugar-app/internal/db"
	"server-sugar-app/internal/model"
	"server-sugar-app/internal/pkg/util"
)

const (
	Precision = 1000000
	rootUser  = "a3b640a1-1de0-4fe8-9b57-b3985256efb2"
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

// 计算糖果奖励前的准备工作
func prepare() error {
	if err := persistLockSIE(); err != nil {
		return fmt.Errorf("persistLockSIE failed: %v", err)
	}

	sie := config.SIE
	accInMap, _, err := getAccountsBalanceInOrOut(sie.SIEAddAccounts, 1)
	if err != nil {
		return errors.Wrap(err, "get account balance in")
	}
	writeFile(accInMap, 1)

	accOutMap, _, err := getAccountsBalanceInOrOut(sie.SIESubAccounts, 2)
	if err != nil {
		return errors.Wrap(err, "get account balance out")
	}
	writeFile(accOutMap, 2)

	return nil
}

// 计算糖果奖励
func CalcReward(now time.Time) (err error) {
	dirname := now.Format("2006-01-02") + "/"
	curFilePath = "sugar/" + dirname

	group.Cond.L.Lock()
	for !group.RelateUpdated {
		group.Cond.Wait()
	}
	group.Cond.L.Unlock()

	if group.StopCalc {
		return errors.New("received relation update stop signal")
	}

	// 获取用户昨日增长率
	yesterdayGrowthRate := getUserYesterdayGrowthRate(now)

	// 用户sie持币量
	yesterdaySIEBals, todaySIEBals, err := getUserSIEBalance(now)
	if err != nil {
		return fmt.Errorf("getUserSIEBalance failed: %v", err)
	}
	// 用户冻结数量(持久化冻结数据)
	yesterdayLockSIEBals, todayLockSIEBals, err := getUserLockSIEBalance(now)
	if err != nil {
		return fmt.Errorf("getUserLockSIEBalance failed: %v", err)
	}

	for k, v := range yesterdayLockSIEBals {
		yesterdaySIEBals[k] += v
	}
	for k, v := range todayLockSIEBals {
		todaySIEBals[k] += v
	}

	// 用户销毁算力
	userDestroyHashRate, err := destroyHashRates()
	if err != nil {
		return fmt.Errorf("get destroyHashRates failed: %v", err)
	}

	lastSugar, err := dao.Sugar.GetLastRecord()
	if err != nil {
		return errors.Wrap(err, "get last sugar record")
	}

	// 获取销毁SIE数量累计值
	shopSIE, err := getUsedShopSIE()
	if err != nil {
		return errors.Wrap(err, "get used shop sie amount")
	}

	// read account in and out
	sumBalanceIn, sumBalanceOut, err := getAccountInAndOut(now)
	if err != nil {
		return errors.Wrap(err, "getAccountInAndOut")
	}

	curCurrency := lastSugar.RealCurrency - (sumBalanceOut - lastSugar.AccountOut) - (shopSIE - lastSugar.ShopSIE) - (sumBalanceIn - lastSugar.AccountIn)

	// 总发行量: 流通量的千分之一
	totalIssuerAmount := curCurrency / 1000
	log.Infof("总发行量: %f", totalIssuerAmount)

	curRealCurrency := curCurrency + totalIssuerAmount

	// make up user details
	rewardDetails := make(map[string]*RewardDetail)
	for u, bal := range todaySIEBals {
		rewardDetails[u] = &RewardDetail{
			YesterdayBal:        yesterdaySIEBals[u],
			TodayBal:            bal,
			DestroyHashRate:     userDestroyHashRate[u],
			YesterdayGrowthRate: yesterdayGrowthRate[u],
		}
	}

	err = rewardOne(rewardDetails, totalIssuerAmount/2)
	if err != nil {
		return fmt.Errorf("reward one failed: %v", err)
	}

	err = rewardTwo(rewardDetails, totalIssuerAmount/2)
	if err != nil {
		return fmt.Errorf("reward two failed: %v", err)
	}

	writeParent(rootUser, rewardDetails)

	// 记录每个人的上线

	newSugar := model.Sugar{
		Sugar:        totalIssuerAmount,
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

	for user, detail := range rewardDetails {
		if detail.BalanceReward > 0.000000 {
			u := model.UserReward{
				UID:     user,
				RewardA: math.Floor(detail.BalanceReward*Precision) / Precision,
			}
			ur = append(ur, u)
		}

		// 批量插入
		if len(ur) > 499 {
			err = dao.UserReward.CreateWithTx(tx, ur)
			if err != nil {
				tx.Rollback()
				return errors.Wrap(err, "user reward insert")
			}
			ur = make([]model.UserReward, 0)
		}
	}
	if len(ur) > 0 {
		err = dao.UserReward.CreateWithTx(tx, ur)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "user reward insert")
		}
	}
	tx.Commit()
	log.Infof("over save user_reward, cost time: %v", time.Since(t))

	// 生成奖励数据文件
	rewardFiles, err := writeRewardFile(rewardDetails)
	if err != nil {
		return errors.Wrap(err, "write reward file")
	}

	// fmt.Println(rewardFiles)
	// 通知IM下载文件
	go func() {
		if err := noticeIMDownloadRewardFile(rewardFiles); err != nil {
			log.Error(errors.Wrap(err, "notice IM server download reward file").Error())
		}
	}()

	log.Info("start save reward detail")
	t = time.Now()

	if err := SaveRewardDetail(rewardDetails); err != nil {
		return fmt.Errorf("SaveRewardDetail failed: %v", err)
	}
	log.Infof("over save reward_detail, cost time: %v", time.Since(t))

	return
}

func SaveRewardDetail(details map[string]*RewardDetail) error {
	rd := make([]model.RewardDetail, 0, 300)
	for user, detail := range details {
		if detail.Droppable() {
			continue
		}
		r := model.RewardDetail{
			UserID:              user,
			YesterdayBal:        detail.YesterdayBal,
			TodayBal:            detail.TodayBal,
			DestroyHashRate:     detail.DestroyHashRate,
			YesterdayGrowthRate: detail.YesterdayGrowthRate,
			GrowthRate:          detail.GrowthRate,
			BalanceHashRate:     detail.BalanceHashRate,
			InviteHashRate:      detail.InviteHashRate,
			BalanceReward:       detail.BalanceReward,
			InviteReward:        detail.InviteReward,
			ParentUID:           detail.ParentUID,
			TeamHashRate:        detail.TeamHashRate,
		}
		rd = append(rd, r)

		if len(rd) > 299 {
			err := dao.RewardDetail.Create(rd)
			if err != nil {
				log.Errorf("reward detail insert failed: %v", err)
			}
			time.Sleep(50 * time.Millisecond) // mysql need rest
			rd = make([]model.RewardDetail, 0, 300)
		}
	}

	if len(rd) > 0 {
		err := dao.RewardDetail.Create(rd)
		if err != nil {
			return errors.Wrap(err, "reward detail insert")
		}
	}
	return nil
}

func getUserYesterdayGrowthRate(now time.Time) map[string]float64 {
	yesterdayDirPath := "sugar/" + now.Add(-24*time.Hour).Format("2006-01-02")

	growthRateFilePath, err := getFilePath(yesterdayDirPath, "growth_rate_")
	if err != nil {
		return nil
	}

	userGrowthRates, err := util.ParseGrowthRateFile(growthRateFilePath)
	if err != nil {
		log.Errorf("parse growth rate file failed: %v", err)
		return nil
	}
	return userGrowthRates
}

// 持久化冻结sie数据
func persistLockSIE() error {
	lockedSIEs, err := dao.GetLockedSIE()
	if err != nil {
		return fmt.Errorf("GetLockedSIE failed: %v", err)
	}
	today := make(map[string]float64)
	for i := range lockedSIEs {
		today[lockedSIEs[i].UID] = lockedSIEs[i].Volume
	}

	// 持久化今日数据
	_, err = writeLockSIEFile(today)
	if err != nil {
		return fmt.Errorf("writeLockSIEFile failed: %v", err)
	}
	return nil
}

// 获取用户昨天和今天的sie持币量
func getUserSIEBalance(now time.Time) (yesterday map[string]float64, today map[string]float64, err error) {
	todayDirPath := "sugar/" + now.Format("2006-01-02")
	yesterdayDirPath := "sugar/" + now.Add(-24*time.Hour).Format("2006-01-02")

	filePrefix := "property_"

	var todayFilePath, yesterdayFilePath string
	todayFilePath, err = getFilePath(todayDirPath, filePrefix)
	if err != nil {
		return
	}
	yesterdayFilePath, err = getFilePath(yesterdayDirPath, filePrefix)
	if err != nil {
		return
	}
	today, _, err = util.ParseCompressAccountFile(todayFilePath)
	if err != nil {
		return nil, nil, err
	}
	yesterday, _, err = util.ParseCompressAccountFile(yesterdayFilePath)
	return
}

// 获取用户昨天和今天的sie冻结量
func getUserLockSIEBalance(now time.Time) (yesterday map[string]float64, today map[string]float64, err error) {
	filePrefix := "lockSIE_"

	// yesterday
	yesterdayDirPath := "sugar/" + now.Add(-24*time.Hour).Format("2006-01-02")
	var yesterdayFilePath string
	hasYesterday := false // 第一天上线没有昨日数据
	yesterdayFilePath, err = getFilePath(yesterdayDirPath, filePrefix)
	if err == nil {
		hasYesterday = true
	} else {
		log.Warnf(err.Error())
		err = nil
	}
	if hasYesterday {
		yesterday, _, err = util.ParseLockSIEFile(yesterdayFilePath)
	}

	// today
	todayDirPath := "sugar/" + now.Format("2006-01-02")
	var todayFilePath string
	todayFilePath, err = getFilePath(todayDirPath, filePrefix)
	if err != nil {
		err = fmt.Errorf("get today lockSIE file path failed: %v", err)
		return
	}
	today, _, err = util.ParseLockSIEFile(todayFilePath)
	return
}

func getAccountInAndOut(now time.Time) (in, out float64, err error) {
	var accInPath, accOutPath string

	todayDirPath := "sugar/" + now.Format("2006-01-02")
	accInPath, err = getFilePath(todayDirPath, "account_in")
	if err != nil {
		err = fmt.Errorf("get account in file path failed: %v", err)
		return
	}
	in, err = util.ParseAccountInOutFile(accInPath)
	if err != nil {
		err = fmt.Errorf("ParseAccountInOutFile failed: %v", err)
		return
	}

	accOutPath, err = getFilePath(todayDirPath, "account_out")
	if err != nil {
		err = fmt.Errorf("get account out file path failed: %v", err)
		return
	}
	out, err = util.ParseAccountInOutFile(accOutPath)
	if err != nil {
		err = fmt.Errorf("ParseAccountInOutFile failed: %v", err)
		return
	}
	return
}

func writeParent(uid string, details map[string]*RewardDetail) {
	children := group.GetDownLineUsers(uid)
	for _, child := range children {
		detail, ok := details[child]
		if ok {
			detail.ParentUID = uid
		}
		writeParent(child, details)
	}
	return
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
	shNow := time.Now().In(util.ShLoc)
	sDate := shNow.Add(-24 * time.Hour).Format("2006-01-02")
	eDate := shNow.Format("2006-01-02")
	// read yesterday check point
	in, out, err := getAccountInAndOut(time.Now().Add(-24 * time.Hour))
	if err != nil {
		return accountMap, sumBalance, fmt.Errorf("getAccountInAndOut failed: %v", err)
	}

	accountMap = make(map[string]float64)
	for _, account := range accounts {
		m := map[string]string{
			"uid":        account,
			"coin":       "SIE",
			"start_date": sDate,
			"end_date":   eDate,
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
					balance += in
				}
			} else if flag == 2 {
				if _, ok := rspMap["balance_out"].(string); ok {
					balance, err = strconv.ParseFloat(rspMap["balance_out"].(string), 64)
					if err != nil {
						return accountMap, sumBalance, err
					}
					balance += out
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

/*
销毁算力
- 销毁后，用户获得销毁算力为销毁的sie数量的10倍，同时商家获得销毁的sie数量的2倍算力
- 个人销毁算力有效期为销毁后3年，商家销毁有效期为销毁后3年
*/
func destroyHashRates() (map[string]float64, error) {
	now := time.Now()
	year2023, _ := time.Parse("2006-01-02 15:04:05", "2023-12-01 00:00:00")
	validityPeriod := -30 * 365 * 24 * time.Hour // 30 year
	if now.After(year2023) {
		validityPeriod = -3 * 365 * 24 * time.Hour // 3 year
	}
	userDestroyedAmount, err := dao.User.QueryDestroyedAmountGroupByUID(now.Add(validityPeriod))
	if err != nil {
		return nil, fmt.Errorf("QueryDestroyedAmountGroupByUID failed: %v", err)
	}
	merchantDestroyedAmount, err := dao.Boss.QueryDestroyedAmountGroupByBossID(now.Add(validityPeriod))
	if err != nil {
		return nil, fmt.Errorf("QueryDestroyedAmountGroupByBossID failed: %v", err)
	}

	// notice that userDestroyedAmount and merchantDestroyedAmount may has overlapped uid.
	destroyHashRates := make(map[string]float64, len(userDestroyedAmount))
	for _, u := range userDestroyedAmount {
		destroyHashRates[u.UID] += math.Pow(u.Credit, 1.1)
	}
	for _, merchant := range merchantDestroyedAmount {
		destroyHashRates[merchant.UID] += math.Pow(merchant.Credit, 1.1)
	}

	return destroyHashRates, nil
}

func getFilePath(dirPath, filePrefix string) (string, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return "", err
	}
	fInfos, err := dir.Readdir(0)
	if err != nil {
		return "", err
	}

	for i := range fInfos {
		fName := fInfos[i].Name()
		if strings.HasPrefix(fName, filePrefix) {
			return dirPath + "/" + fName, nil
		}
	}
	return "", fmt.Errorf("could not found file with [%s] prefix", filePrefix)
}

func ParseRewardDetail(path string) (map[string]*RewardDetail, error) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		err = errors.Wrap(err, "open zip file reader")
		return nil, err
	}
	defer rc.Close()

	if len(rc.Reader.File) == 0 || rc.Reader.File[0] == nil {
		err = errors.New("empty zip file")
		return nil, err
	}
	f, err := rc.Reader.File[0].Open()
	if err != nil {
		err = errors.Wrap(err, "open zip file")
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	details := make(map[string]*RewardDetail)
	for {
		line, _, err := r.ReadLine()
		if err == nil {
			bs := bytes.Split(line, []byte(","))
			if len(bs) == 14 {
				uid := string(bs[0])
				yesterdayBal, _ := strconv.ParseFloat(string(bs[1]), 64)
				todayBal, _ := strconv.ParseFloat(string(bs[2]), 64)
				destroyHashRate, _ := strconv.ParseFloat(string(bs[3]), 64)
				yesterdayGrowthRate, _ := strconv.ParseFloat(string(bs[4]), 64)
				growthRate, _ := strconv.ParseFloat(string(bs[5]), 64)
				balanceHashRate, _ := strconv.ParseFloat(string(bs[7]), 64)
				inviteHashRate, _ := strconv.ParseFloat(string(bs[9]), 64)
				balanceReward, _ := strconv.ParseFloat(string(bs[10]), 64)
				inviteReward, _ := strconv.ParseFloat(string(bs[11]), 64)
				teamHashRate, _ := strconv.ParseFloat(string(bs[12]), 64)
				parentUID := string(bs[13])

				rewardDetail := RewardDetail{
					YesterdayBal:        yesterdayBal,
					TodayBal:            todayBal,
					DestroyHashRate:     destroyHashRate,
					YesterdayGrowthRate: yesterdayGrowthRate,
					GrowthRate:          growthRate,
					BalanceHashRate:     balanceHashRate,
					InviteHashRate:      inviteHashRate,
					BalanceReward:       balanceReward,
					InviteReward:        inviteReward,
					ParentUID:           parentUID,
					TeamHashRate:        teamHashRate,
				}

				details[uid] = &rewardDetail
			} else {
				return details, err
			}
		} else if err == io.EOF {
			break
		} else {
			err = errors.Wrap(err, "read line")
			return details, err
		}
	}
	return details, nil
}
