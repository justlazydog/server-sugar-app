package sugar

import (
	"math"
	"os"
	"sort"

	"github.com/pkg/errors"

	"server-sugar-app/config"
	"server-sugar-app/internal/app/group"
	"server-sugar-app/internal/pkg/util"
)

/*
持币奖励规则：
	计算用户持币算力1=用户销毁金额*10
	计算商家持币算力1=min（商铺销毁金额*2，商家账户持币）
	计算用户所有算力=用户持币算力1（用户作为消费者的销毁）+商家持币算力1（用户作为商家的销毁）
	平台总算力=所有用户总持币算力之和
*/
func rewardOne(in []map[string]float64, out, outf map[string]float64, sumAmount float64) error {
	// 持有发行数量, 总算力
	hashRateTotal := 0.0
	// 计算算力
	for ink, inv := range in[1] {
		if !isInWhiteList(ink) {
			hashRate := inv
			outf[ink] = hashRate
			hashRateTotal += hashRate
		}
	}

	for ink, inv := range in[2] {
		if !isInWhiteList(ink) {
			hashRate := inv
			if in[0][ink] < hashRate {
				hashRate = in[0][ink]
			}
			outf[ink] += hashRate
			hashRateTotal += hashRate
		}
	}

	// internal account
	sie := config.SIE
	accs := sie.SIERewardAccounts

	// 送给内部账户的算力
	extraHashRate := hashRateTotal * 0.025
	for _, acc := range accs {
		outf[acc] = outf[acc] + extraHashRate
		hashRateTotal += extraHashRate
	}

	// 计算持有奖励
	for k, v := range outf {
		out[k] = v / hashRateTotal * sumAmount
	}

	go func() {
		filename := writeForceFile(outf, nil, 1)
		err := util.ZipFiles(filename+".zip", []string{filename})
		if err != nil {
			return
		}
		os.Remove(filename)
	}()

	return nil
}

/*
根据大小区计算用户邀请算力
	持币100以上的用户可获得奖励
	用户持币算力=大区团队算力的0.3次方+所有小区团队算力的0.7次方之和（每个小区的0.7次相加）
*/
func rewardTwo(in map[string]float64, opM map[string]float64, out, outf map[string]float64, sumAmount float64) error {
	group.Cond.L.Lock()
	for !group.RelateUpdated {
		group.Cond.Wait()
	}
	group.Cond.L.Unlock()

	if group.StopCalc {
		return errors.New("received relation update stop signal")
	}
	// allMinorForce 所有持币小于100的总算力
	// allForce 总算力
	// t := time.Now()
	var allMinorForce, allForce float64
	uForeM := make(map[string]float64) // 用于持币大于100的用户算力（实际发送奖励者)
	for _, user := range group.Users {
		if !isInWhiteList(user) {
			if in[user] < 100 {
				minorForce, err := calcInviteReward(user, opM)
				if err != nil {
					return errors.Wrap(err, "calc invite reward")
				}
				outf[user] = minorForce
				allMinorForce += minorForce
				allForce += minorForce
			} else {
				inviteForce, err := calcInviteReward(user, opM)
				if err != nil {
					return errors.Wrap(err, "calc invite reward")
				}
				outf[user] = inviteForce
				uForeM[user] = inviteForce
				allForce += inviteForce
			}
		}
	}

	// 特殊系统账户
	sie := config.SIE
	sysAccountA := sie.SIERewardAccounts[0]
	sysAccountB := sie.SIERewardAccounts[1]

	outf[sysAccountA] = allForce*0.025 + allMinorForce
	outf[sysAccountB] = allForce * 0.025

	uForeM[sysAccountA] = allForce*0.025 + allMinorForce
	uForeM[sysAccountB] = allForce * 0.025

	for k, v := range uForeM {
		out[k] = v / (allForce * 1.05) * sumAmount
	}
	go func() {
		filename := writeForceFile(outf, opM, 2)
		err := util.ZipFiles(filename+".zip", []string{filename})
		if err != nil {
			return
		}
		os.Remove(filename)
	}()
	return nil
}

// 计算邀请算力
func calcInviteReward(uid string, opM map[string]float64) (fInviteForce float64, err error) {
	users := group.GetDownLineUsers(uid)
	connRegion := make([]float64, 0)
	for _, user := range users {
		if !isInWhiteList(user) {
			curProperty := opM[user]
			m := make(map[string]bool)
			subUsers, err := group.GetAllDownLineUsers(user, m)
			if err != nil {
				err = errors.Wrap(err, "get all downline users")
				return fInviteForce, err
			}
			for _, v := range subUsers {
				if !isInWhiteList(v) {
					curProperty += +opM[v]
				}
			}
			connRegion = append(connRegion, curProperty)
		}
	}

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
