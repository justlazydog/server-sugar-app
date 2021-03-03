package sugar

import "server-sugar-app/internal/dao"

// make sure call this func before create
func getOrCrtAvgGrthRate(date string, rewardDetails map[string]*RewardDetail, yesterdayBalSum, todayBalSum float64) float64 {
	// get yesterday average growth rate
	s, err := dao.Sugar.GetByDate(date)
	if err != nil || s.AvgGrowthRate <= 0 {
		return createAvgGrowthRate(rewardDetails, createAvgGrowthRateYesterday, yesterdayBalSum, todayBalSum)
	}
	return s.AvgGrowthRate
}

const (
	createAvgGrowthRateYesterday = 1
	createAvgGrowthRateToday     = 2
)

func createAvgGrowthRate(rewardDetails map[string]*RewardDetail, when int, yesterdayBalSum, todayBalSum float64) float64 {

	var growthRateSum, balSum float64
	switch when {
	case createAvgGrowthRateYesterday:
		for _, d := range rewardDetails {
			if d == nil {
				continue
			}
			growthRateSum += d.YesterdayGrowthRate * d.YesterdayBal
		}
		balSum = yesterdayBalSum
	case createAvgGrowthRateToday:
		fallthrough
	default:
		for _, d := range rewardDetails {
			if d == nil {
				continue
			}
			growthRateSum += d.GrowthRate * d.TodayBal
		}
		balSum = todayBalSum
	}
	if balSum == 0 {
		return 1
	}
	return growthRateSum / balSum
}
