package sugar

import (
	"fmt"
	"github.com/shopspring/decimal"
	"math"
	"os"
	"sort"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
	"server-sugar-app/internal/app/group"
	"server-sugar-app/internal/pkg/util"
)

// 持币奖励信息
type RewardDetail struct {
	YesterdayBal        float64 // 昨日持币
	TodayBal            float64 // 今日持币
	DestroyHashRate     float64 // 销毁算力
	YesterdayGrowthRate float64 // 昨日增长率
	GrowthRate          float64 // 今日增长率
	BalanceHashRate     float64 // 持币算力
	InviteHashRate      float64 // 邀请算力
	BalanceReward       float64 // 持币奖励
	InviteReward        float64 // 邀请奖励
	ParentUID           string  // 邀请人
	TeamHashRate        float64 // 区域算力
}

/*
持币糖果最低发放需要持有100SIE
持币不足100SIE的奖励发放到一个特定的账户上。
*/
func rewardOne(details map[string]*RewardDetail, sumAmount float64) error {
	log.Info("Start calc reward one...")
	t := time.Now()

	// 持有发行数量, 总算力
	hashRateTotal := 0.0

	// internal account
	sie := config.SIE
	// 特殊系统账户
	sysAccountA := sie.SIERewardAccounts[0]
	sysAccountB := sie.SIERewardAccounts[1]

	// make sure account exists
	if _, ok := details[sysAccountA]; !ok {
		details[sysAccountA] = &RewardDetail{}
	}
	if _, ok := details[sysAccountB]; !ok {
		details[sysAccountB] = &RewardDetail{}
	}

	// 计算持币算力
	for u, d := range details { // 包含所有用户（即使持币为0）
		if !isInWhiteList(u) {
			// 计算增长率
			calculateGrowthRate(d)
			// 用户当前持币算力=用户持币量*增长率+用户的销毁算力
			d.BalanceHashRate = d.TodayBal*d.GrowthRate + d.DestroyHashRate
			hashRateTotal += d.BalanceHashRate
		}
	}
	log.Infof("总持币算力: %f", hashRateTotal)

	// make sure internal reward account exists
	if _, ok := details[sie.SIERewardAccount]; !ok {
		details[sie.SIERewardAccount] = &RewardDetail{
			YesterdayGrowthRate: 1,
			GrowthRate:          1,
		}
	}

	extraHashRate := hashRateTotal * 0.05
	details[sysAccountA].BalanceHashRate += extraHashRate / 2
	details[sysAccountB].BalanceHashRate += extraHashRate / 2
	hashRateTotal += extraHashRate
	defer func() {
		details[sysAccountA].BalanceHashRate -= extraHashRate / 2
		details[sysAccountB].BalanceHashRate -= extraHashRate / 2
	}()

	// 计算持币奖励
	for uid, d := range details {
		// 持币糖果最低发放需要持有100SIE
		rewardable := true
		if d.TodayBal < 100 {
			rewardable = false
		}

		// 用户获得持币糖果的数量=用户个人持币算力/平台总持币算力*本次发放的糖果总和*50%, notice that sumAmount = issuerAmount / 2
		reward := d.BalanceHashRate / hashRateTotal * sumAmount
		if rewardable {
			if uid == sie.SIERewardAccount { // 系统帐号
				d.BalanceReward += reward
			} else {
				d.BalanceReward = reward
			}
		} else {
			details[sie.SIERewardAccount].BalanceReward += reward
		}
	}

	go func() {
		filename := writeForceFile(details, 1)
		err := util.ZipFiles(filename+".zip", []string{filename})
		if err != nil {
			return
		}
		os.Remove(filename)
	}()

	filename, err := writeGrowthRateFile(details)
	if err != nil {
		return fmt.Errorf("write growth rate file failed: %v", err)
	}
	err = util.ZipFiles(filename+".zip", []string{filename})
	if err != nil {
		log.Warnf("zip growth rate file failed: %v", err)
	} else {
		os.Remove(filename)
	}

	log.Infof("calc reward one over, cost time: %v", time.Since(t))
	return nil
}

/* 计算增长率
增长率：每个钱包的初始增长率为1，通过每日持币量变化，增长率为
	（（当日持有sie数量-前一日持有的sie数量）/前一日持有的sie数量*100%，如果前一日持币数量为0，则增长率为1）增长率的变化如下：
钱包日持币量增加n%[100%,+∞)，增长率为 (昨日增长率/(n+1)) ，向下取整
钱包日持币量增加[2%,100%)，增长率（+1）
钱包日持币量增加不足2%，增长率（-1）
钱包日持币量减少n%，增长率（-n），增长率最小为1
*/
func calculateGrowthRate(d *RewardDetail) {
	if d.YesterdayGrowthRate < 1 { // 增长率最小为1
		d.YesterdayGrowthRate = 1
	}
	if d.YesterdayBal > 0 {
		growthPercent, _ := safeDiv(d.TodayBal-d.YesterdayBal, d.YesterdayBal)
		if growthPercent >= 1 { // 钱包日持币量增加n倍[100%,+∞)，增长率(昨日增长率/(n+1))
			d.GrowthRate, _ = safeDiv(d.YesterdayGrowthRate, growthPercent+1)
			d.GrowthRate = math.Floor(d.GrowthRate)
		} else if growthPercent >= 0.02 { // 钱包日持币量增加[2%,100%)，增长率（+1）
			d.GrowthRate = d.YesterdayGrowthRate + 1
		} else if growthPercent >= 0 { // 钱包日持币量增加不足2%，增长率（-1）
			d.GrowthRate = d.YesterdayGrowthRate - 1
		} else { //钱包日持币量减少n%，增长率（-n）
			d.GrowthRate = d.YesterdayGrowthRate + math.Floor(growthPercent*100)
		}
	} else {
		d.GrowthRate = 1
	}
	if d.GrowthRate < 1 { // 增长率最小为1
		d.GrowthRate = 1
	}
}

/*
	每个用户的区域为：用户自己+直接邀请的成员的区域
	团队持币算力：用户区域中所有成员的持币算力之和
	大区域：用户直接邀请的成员中，团队持币最大的区域
	小区域：用户直接邀请的成员中，出去大区域以外的其他区域都是小区域

	根据大小区计算用户邀请算力
	持币100以上的用户可获得奖励
	用户当前邀请算力=大区团队算力的0.3次方+所有小区团队算力的0.7次方之和（每个小区的0.7次相加）
*/
func rewardTwo(details map[string]*RewardDetail, sumAmount float64) error {
	log.Info("start calc reward two...")
	t := time.Now()
	group.Cond.L.Lock()
	for !group.RelateUpdated {
		group.Cond.Wait()
	}
	group.Cond.L.Unlock()

	if group.StopCalc {
		return errors.New("received relation update stop signal")
	}

	var allMinorForce, allForce float64
	uForeM := make(map[string]float64) // 用于持币大于100的用户算力（实际发送奖励者)
	for _, user := range group.Users {
		if !isInWhiteList(user) {
			detail, ok := details[user]
			if !ok {
				detail = &RewardDetail{
					YesterdayGrowthRate: 1,
					GrowthRate:          1,
				}
				details[user] = detail
			}
			if detail.TodayBal < 100 {
				minorForce, err := calcInviteReward(user, details)
				if err != nil {
					return errors.Wrap(err, "calc invite reward")
				}
				detail.InviteHashRate = minorForce
				allMinorForce += minorForce
				allForce += minorForce
			} else {
				inviteForce, err := calcInviteReward(user, details)
				if err != nil {
					return errors.Wrap(err, "calc invite reward")
				}
				detail.InviteHashRate = inviteForce
				uForeM[user] = inviteForce
				allForce += inviteForce
			}
		}
	}

	// 特殊系统账户
	sie := config.SIE
	sysAccountA := sie.SIERewardAccounts[0]
	sysAccountB := sie.SIERewardAccounts[1]

	// make sure account exists
	if _, ok := details[sysAccountA]; !ok {
		details[sysAccountA] = &RewardDetail{}
	}
	if _, ok := details[sysAccountB]; !ok {
		details[sysAccountB] = &RewardDetail{}
	}

	details[sysAccountA].InviteHashRate = allForce*0.025 + allMinorForce
	details[sysAccountB].InviteHashRate = allForce * 0.025

	uForeM[sysAccountA] = allForce*0.025 + allMinorForce
	uForeM[sysAccountB] = allForce * 0.025

	for k, v := range uForeM {
		detail, ok := details[k]
		if !ok {
			detail := &RewardDetail{
				YesterdayGrowthRate: 1,
			}
			details[k] = detail
		}
		detail.InviteReward = v / (allForce * 1.05) * sumAmount
	}

	go func() {
		filename := writeForceFile(details, 2)
		err := util.ZipFiles(filename+".zip", []string{filename})
		if err != nil {
			return
		}
		os.Remove(filename)
	}()

	log.Infof("calc reward two over, cost time: %v", time.Since(t))
	return nil
}

// 计算邀请算力
func calcInviteReward(uid string, details map[string]*RewardDetail) (fInviteForce float64, err error) {
	users := group.GetDownLineUsers(uid)
	connRegion := make([]float64, 0)
	var teamBalHashRate float64
	detail, ok := details[uid]
	if !ok {
		detail = &RewardDetail{
			YesterdayGrowthRate: 1,
			GrowthRate:          1,
		}
		details[uid] = detail
	}
	teamBalHashRate = detail.BalanceHashRate
	for _, user := range users {
		if !isInWhiteList(user) {
			var curProperty float64
			detail, ok := details[user]
			if ok {
				curProperty = detail.BalanceHashRate
				teamBalHashRate += detail.BalanceHashRate
			}
			m := make(map[string]bool)
			subUsers, err := group.GetAllDownLineUsers(user, m)
			if err != nil {
				err = errors.Wrap(err, "get all downline users")
				return fInviteForce, err
			}
			for _, v := range subUsers {
				if !isInWhiteList(v) {
					detail, ok := details[v]
					if ok {
						curProperty += detail.BalanceHashRate
						teamBalHashRate += detail.BalanceHashRate
					}
				}
			}
			connRegion = append(connRegion, curProperty)
		}
	}
	details[uid].TeamHashRate = teamBalHashRate

	if len(connRegion) < 2 {
		var fProperty float64
		if len(connRegion) == 0 {
			fProperty = 0
		} else if len(connRegion) == 1 {
			fProperty = connRegion[0]
		}
		fInviteForce := math.Pow(fProperty, 0.3)
		return fInviteForce, nil
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(connRegion)))
	fInviteForce = math.Pow(connRegion[0], 0.3)
	n := len(connRegion)
	for i := 1; i < n; i++ {
		fInviteForce += math.Pow(connRegion[i], 0.7)
	}
	return fInviteForce, nil
}

// 判断用户是否在白名单
func isInWhiteList(uid string) bool {
	sie := config.SIE
	for _, account := range sie.SIEWhiteList {
		if uid == account {
			return true
		}
	}
	return false
}

func safeDiv(f1, f2 float64) (float64, bool) {
	d1 := decimal.NewFromFloat(f1)
	d2 := decimal.NewFromFloat(f2)
	return d1.Div(d2).Float64()
}
