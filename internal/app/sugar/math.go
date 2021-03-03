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
	YesterdayBal             float64 // 昨日持币
	TodayBal                 float64 // 今日持币
	DestroyHashRate          float64 // 销毁算力
	YesterdayGrowthRate      float64 // 昨日增长率
	GrowthRate               float64 // 今日增长率
	BalanceHashRateForInvite float64 // 持币算力(用于邀请算力计算)
	BalanceHashRate          float64 // 持币算力
	PureBalanceHashRate      float64 // 持币算力 持币部分
	InviteHashRate           float64 // 邀请算力
	BalanceReward            float64 // 持币奖励
	InviteReward             float64 // 邀请奖励
	ParentUID                string  // 邀请人
	TeamHashRate             float64 // 区域算力
}

func (r *RewardDetail) Droppable() bool {
	return r.YesterdayBal == 0 &&
		r.TodayBal == 0 &&
		r.DestroyHashRate == 0 &&
		r.YesterdayGrowthRate == 1 &&
		r.GrowthRate == 1 &&
		r.BalanceHashRate == 0 &&
		r.InviteHashRate == 0 &&
		r.BalanceReward == 0 &&
		r.InviteReward == 0 &&
		r.TeamHashRate == 0
}

/*
持币糖果最低发放需要持有100SIE
持币不足100SIE的奖励发放到一个特定的账户上。
*/
func rewardOne(details map[string]*RewardDetail, sumAmount, yesterdayAvgGrowthRate float64) error {
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
	// make sure internal reward account exists
	if _, ok := details[sie.SIERewardAccount]; !ok {
		details[sie.SIERewardAccount] = &RewardDetail{
			YesterdayGrowthRate: 1,
			GrowthRate:          1,
		}
	}

	// 计算持币算力
	for u, d := range details { // 包含所有用户（即使持币为0）
		if !isInWhiteList(u) {
			// 计算增长率
			calculateGrowthRate(d, yesterdayAvgGrowthRate)
			/*
				λ	用户实际持币算力：
				ν	当用户持币≥100时：
				υ	持币算力=持币部分+销毁部分=用户持币量^1.1*增长率+用户的销毁算力
				ν	当用户持币小于100时：
				υ	持币算力=销毁部分=用户的销毁算力
				υ	此时该用户持币算力中的的持币部分（用户持币量*增长率），得到的算力归0*/
			d.PureBalanceHashRate = math.Pow(d.TodayBal, 1.1) * d.GrowthRate
			if d.TodayBal >= 100 {
				d.BalanceHashRate += d.PureBalanceHashRate + d.DestroyHashRate
			} else {
				d.BalanceHashRate += d.DestroyHashRate
				//details[sie.SIERewardAccount].BalanceHashRate += d.PureBalanceHashRate // 持币部分分配给系统帐号
			}
			d.BalanceHashRateForInvite = d.BalanceHashRate // 计算邀请算力用的。显示给用户看的
			hashRateTotal += d.BalanceHashRateForInvite
		}
	}
	log.Infof("总持币算力: %f", hashRateTotal)

	extraHashRate := hashRateTotal * 0.05
	details[sysAccountA].BalanceHashRate += extraHashRate / 2
	details[sysAccountB].BalanceHashRate += extraHashRate / 2
	hashRateTotal += extraHashRate
	defer func() {
		details[sysAccountA].BalanceHashRate -= extraHashRate / 2
		details[sysAccountB].BalanceHashRate -= extraHashRate / 2
	}()

	// 计算持币奖励
	for _, d := range details {
		// 用户获得持币糖果的数量=用户个人持币算力/平台总持币算力*本次发放的糖果总和*50%, notice that sumAmount = issuerAmount / 2
		reward := d.BalanceHashRate / hashRateTotal * sumAmount
		d.BalanceReward = reward
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

// 计算增长率
func calculateGrowthRate(d *RewardDetail, yesterdayAvgGrowthRate float64) {
	if d.YesterdayBal <= 0 || d.TodayBal < 100 {
		d.GrowthRate = 1
		return
	}

	// n represents增长量
	rawN, _ := safeDiv(d.TodayBal-d.YesterdayBal, d.YesterdayBal)
	n := rawN * 100

	exponent := 0.03 * (yesterdayAvgGrowthRate - d.YesterdayGrowthRate)
	m := math.Pow(math.E, exponent)

	if d.YesterdayGrowthRate < 1 { // 增长率最小为1
		d.YesterdayGrowthRate = 1
	}

	if n >= 2 && n < 100 {
		d.GrowthRate = d.YesterdayGrowthRate + m
	} else if n >= 0 && n < 2 {
		d.GrowthRate = d.YesterdayGrowthRate - m
	} else if n < 0 {
		d.GrowthRate = d.YesterdayGrowthRate - math.Floor(-n)
	} else { // n >= 100 case
		tmp, _ := safeDiv(n, 100)
		tmp = math.Floor(tmp)
		growthRate, _ := safeDiv(1, tmp+1)
		d.GrowthRate = growthRate
	}

	if d.GrowthRate < 1 { // 增长率最小为1
		d.GrowthRate = 1
	}

	// set precision to 6
	d.GrowthRate, _ = decimal.NewFromFloat(d.GrowthRate).Round(6).Float64()
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

	var allForce float64
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
				_, err := calcInviteReward(user, details)
				if err != nil {
					return errors.Wrap(err, "calc invite reward")
				}
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

	details[sysAccountA].InviteHashRate = allForce * 0.025
	details[sysAccountB].InviteHashRate = allForce * 0.025

	uForeM[sysAccountA] = allForce * 0.025
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
	teamBalHashRate = detail.BalanceHashRateForInvite
	for _, user := range users {
		if !isInWhiteList(user) {
			var curProperty float64
			detail, ok := details[user]
			if ok {
				curProperty = detail.BalanceHashRateForInvite
				teamBalHashRate += detail.BalanceHashRateForInvite
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
						curProperty += detail.BalanceHashRateForInvite
						teamBalHashRate += detail.BalanceHashRateForInvite
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
	return d1.DivRound(d2, 10).Float64()
}
