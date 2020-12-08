package dao

import (
	"database/sql"
	"strings"
	"time"

	"server-sugar-app/internal/db"
	"server-sugar-app/internal/model"
)

type sugar struct {
}

var Sugar = new(sugar)

func (*sugar) Create(s model.Sugar) (err error) {
	sqlStr := "insert into sugars (sugar,currency,real_currency,shop_sie,shop_used_sie,account_in,account_out) " +
		"values (?,?,?,?,?,?,?)"
	_, err = db.MysqlCli.Exec(sqlStr, s.Sugar, s.Currency, s.RealCurrency,
		s.ShopSIE, s.ShopUsedSIE, s.AccountIn, s.AccountOut)
	return
}

func (*sugar) CreateWithTx(tx *sql.Tx, s model.Sugar) (err error) {
	sqlStr := "insert into sugars (sugar,currency,real_currency,shop_sie,shop_used_sie,account_in,account_out) " +
		"values (?,?,?,?,?,?,?)"
	_, err = tx.Exec(sqlStr, s.Sugar, s.Currency, s.RealCurrency, s.ShopSIE, s.ShopUsedSIE, s.AccountIn, s.AccountOut)
	return
}

func (*sugar) GetLastRecord() (s model.Sugar, err error) {
	sqlStr := "select create_time,sugar,currency,real_currency,shop_sie,shop_used_sie,account_in,account_out from " +
		"sugars order by id desc limit 1"
	row := db.MysqlCli.QueryRow(sqlStr)
	var createTime string
	err = row.Scan(&createTime, &s.Sugar, &s.Currency, &s.RealCurrency, &s.ShopSIE, &s.ShopUsedSIE, &s.AccountIn,
		&s.AccountOut)
	if err == sql.ErrNoRows {
		s.RealCurrency = 85410020.2505906400000000 * 1.02
		s.CreateTime = time.Now().Add(-24 * time.Hour)
		return
	}
	s.CreateTime, err = time.Parse("2006-01-02 15:04:05", createTime)
	return
}

type userReward struct {
}

var UserReward = new(userReward)

func (*userReward) Get() (m map[string]float64, err error) {
	sqlStr := "select uid,sum(reward_a) from user_reward group	by uid"
	rows, err := db.MysqlCli.Query(sqlStr)
	if err != nil {
		return
	}

	m = make(map[string]float64)
	for rows.Next() {
		var (
			uid    string
			reward float64
		)
		err = rows.Scan(&uid, &reward)
		if err != nil && !strings.Contains(err.Error(), "converting") {
			err = nil
			return
		}
		m[uid] = reward
	}
	return
}

func (*userReward) GetRewardA() (reward float64, err error) {
	sqlStr := "select sum(reward_a) as reward from user_reward"
	rows := db.MysqlCli.QueryRow(sqlStr)
	err = rows.Scan(&reward)
	if err != nil {
		if strings.Contains(err.Error(), "converting") {
			err = nil
			reward = 0
			return
		}
	}
	return
}

func (u *userReward) CreateWithTx(tx *sql.Tx, data []model.UserReward) (err error) {
	var vals []interface{}

	sqlStr := "insert into user_reward (uid,reward_a) values "
	for _, row := range data {
		sqlStr += "(?,?),"
		vals = append(vals, row.UID, row.RewardA)
	}
	// trim the last ,
	sqlStr = sqlStr[0 : len(sqlStr)-1]
	// prepare the statement
	stmt, err := tx.Prepare(sqlStr)
	if err != nil {
		return
	}

	// format all vals at once
	_, err = stmt.Exec(vals...)
	return
}

type rewardDetail struct {
}

var RewardDetail = new(rewardDetail)

func (*rewardDetail) Create(data []model.RewardDetail) error {
	var vals []interface{}

	sqlStr := "insert into reward_detail (user_id,yesterday_bal,today_bal,destroy_hash_rate,yesterday_growth_rate," +
		"growth_rate,balance_hash_rate,invite_hash_rate,balance_reward,invite_reward,parent_uid,team_hash_rate) values "
	for _, row := range data {
		sqlStr += "(?,?,?,?,?,?,?,?,?,?,?,?),"
		vals = append(vals, row.UserID, row.YesterdayBal, row.TodayBal, row.DestroyHashRate, row.YesterdayGrowthRate,
			row.GrowthRate, row.BalanceHashRate, row.InviteHashRate, row.BalanceReward, row.InviteReward, row.ParentUID,
			row.TeamHashRate)
	}
	// trim the last ,
	sqlStr = sqlStr[0 : len(sqlStr)-1]
	// prepare the statement
	stmt, err := db.MysqlCli.Prepare(sqlStr)
	if err != nil {
		return err
	}

	// format all vals at once
	_, err = stmt.Exec(vals...)
	return err
}

func (*rewardDetail) Get(userID string) (res model.RewardDetail, err error) {
	t := time.Now().Add(-24 * time.Hour)
	sqlStr := "select create_time,today_bal,growth_rate,balance_hash_rate,invite_hash_rate from reward_detail where " +
		"user_id = ? and date(create_time) >= date(?) order by id desc limit 1"
	row := db.MysqlCli.QueryRow(sqlStr, t, userID)
	var createTime string
	err = row.Scan(&createTime, &res.TodayBal, &res.GrowthRate, &res.BalanceHashRate, &res.InviteHashRate)
	if err == sql.ErrNoRows {
		return res, nil
	}

	res.CreateTime, err = time.Parse("2006-01-02 15:04:05", createTime)
	return res, err
}
