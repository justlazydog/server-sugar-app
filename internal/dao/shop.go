package dao

import (
	"database/sql"
	"time"

	"server-sugar-app/internal/db"
	"server-sugar-app/internal/model"
)

type user struct {
}

var User = new(user)

func (*user) Add(user model.User) (err error) {
	_, err = db.MysqlCli.Exec("insert into shop_user (uid,open_id,order_id,amount,credit,flag) values (?,?,?,?,?,?)",
		user.UID, user.OpenID, user.OrderID, user.Amount, user.Credit, user.Flag)
	return
}

func (*user) GetCredit(userID string) (offline, online float64, err error) {
	rows, err := db.MysqlCli.Query(
		"select flag,sum(credit) as all_credit from shop_user where open_id = ? group by flag ", userID)
	if err != nil {
		return
	}

	type Res struct {
		Flag      uint8
		AllCredit float64
	}

	var res []Res
	for rows.Next() {
		var r Res
		err = rows.Scan(&r.Flag, &r.AllCredit)
		if err != nil {
			return
		}
		res = append(res, r)
	}

	for _, v := range res {
		if v.Flag == 1 {
			offline = v.AllCredit
		}
		if v.Flag == 2 {
			online = v.AllCredit
		}
	}
	return
}

func (*user) GetCreditDetail(userID string, year int, month, flag uint8, lastID, pageSize int) (users []model.User, err error) {
	var rows *sql.Rows
	if lastID == 0 {
		rows, err = db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,flag,created_at from shop_user "+
			"where open_id = ? and year(created_at) = ? month(created_at) = ? and flag = ? order by id desc limit ?",
			userID, year, month, flag, pageSize)
	} else {
		rows, err = db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,flag,created_at from shop_user "+
			"where open_id = ? and year(created_at) = ? month(created_at) = ? and flag = ? and id < ? order by id desc limit ?",
			userID, year, month, flag, lastID, pageSize)
	}

	if err != nil {
		return
	}

	for rows.Next() {
		var (
			user     model.User
			createAt string
		)
		err = rows.Scan(&user.ID, &user.OpenID, &user.Amount, &user.Credit, &user.OrderID, &user.Multiple, &user.Flag, &createAt)
		if err != nil {
			return
		}

		t, _ := time.Parse("2006-01-02 15:04:05", createAt)
		user.CreatedAt = t.Unix()
		users = append(users, user)
	}
	return
}

func (*user) GetUsedAmount() (offline, online float64, err error) {
	rows, err := db.MysqlCli.Query("select flag,sum(amount) as all_amount from shop_user group by flag")
	if err != nil {
		return
	}

	type Res struct {
		Flag      uint8
		AllAmount float64
	}

	var res []Res
	for rows.Next() {
		var r Res
		err = rows.Scan(&r.Flag, &r.AllAmount)
		if err != nil {
			return
		}
		res = append(res, r)
	}

	for _, v := range res {
		if v.Flag == 1 {
			offline = v.AllAmount
		}
		if v.Flag == 2 {
			online = v.AllAmount
		}
	}
	return
}

type shop struct {
}

var Shop = new(shop)

func (*shop) Add(shop model.Boss) (err error) {
	_, err = db.MysqlCli.Exec("insert into shop_boss (uid,open_id,order_id,amount,credit,flag) values (?,?,?,?,?,?)",
		shop.UID, shop.OpenID, shop.OrderID, shop.Amount, shop.Credit, shop.Flag)
	return
}

func (*shop) GetCredit(userID string) (offline, online float64, err error) {
	rows, err := db.MysqlCli.Query(
		"select flag,sum(credit) as all_credit from shop_boss where open_id = ? group by flag ", userID)
	if err != nil {
		return
	}

	type Res struct {
		Flag      uint8
		AllCredit float64
	}

	var res []Res
	for rows.Next() {
		var r Res
		err = rows.Scan(&r.Flag, &r.AllCredit)
		if err != nil {
			return
		}
		res = append(res, r)
	}

	for _, v := range res {
		if v.Flag == 1 {
			offline = v.AllCredit
		}
		if v.Flag == 2 {
			online = v.AllCredit
		}
	}
	return
}

func (*shop) GetCreditDetail(userID string, year int, month, flag uint8, lastID, pageSize int) (users []model.Boss, err error) {
	var rows *sql.Rows
	if lastID == 0 {
		rows, err = db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,flag,created_at from shop_boss "+
			"where open_id = ? and year(created_at) = ? and month(created_at) = ? and flag = ? order by id desc limit ?",
			userID, year, month, flag, pageSize)
	} else {
		rows, err = db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,flag,created_at from shop_boss "+
			"where open_id = ? and year(created_at) = ? month(created_at) = ? and flag = ? and id < ? order by id desc limit ?",
			userID, year, month, flag, lastID, pageSize)
	}

	if err != nil {
		return
	}

	for rows.Next() {
		var (
			user     model.Boss
			createAt string
		)
		err = rows.Scan(&user.ID, &user.OpenID, &user.Amount, &user.Credit, &user.OrderID, &user.Multiple, &user.Flag, &createAt)
		if err != nil {
			return
		}

		t, _ := time.Parse("2006-01-02 15:04:05", createAt)
		user.CreatedAt = t.Unix()
		users = append(users, user)
	}
	return
}
