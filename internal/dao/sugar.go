package dao

import (
	"database/sql"
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