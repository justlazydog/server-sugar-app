package dao

import (
	"database/sql"
	"fmt"
	"time"

	"server-sugar-app/internal/db"
	"server-sugar-app/internal/model"
)

type user struct {
}

var User = new(user)

func (*user) Add(user model.User) (err error) {
	_, err = db.MysqlCli.Exec("insert into shop_user (uid,open_id,order_id,amount,credit,multiple,extra_multiple,flag) values (?,?,?,?,?,?,?,?)",
		user.UID, user.OpenID, user.OrderID, user.Amount, user.Credit, user.Multiple, user.ExtraMultiple, user.Flag)
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

func (*user) GetAllCredit() (offline, online float64, err error) {
	rows, err := db.MysqlCli.Query(
		"select flag,sum(credit) as all_credit from shop_user group by flag ")
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
		rows, err = db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,extra_multiple,flag,created_at from shop_user "+
			"where open_id = ? and year(created_at) = ? and month(created_at) = ? and flag = ? order by id desc limit ?",
			userID, year, month, flag, pageSize)
	} else {
		rows, err = db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,extra_multiple,flag,created_at from shop_user "+
			"where open_id = ? and year(created_at) = ? and month(created_at) = ? and flag = ? and id < ? order by id desc limit ?",
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
		err = rows.Scan(&user.ID, &user.OpenID, &user.Amount, &user.Credit, &user.OrderID, &user.Multiple,
			&user.ExtraMultiple, &user.Flag, &createAt)
		if err != nil {
			return
		}

		t, _ := time.Parse("2006-01-02 15:04:05", createAt)
		user.CreatedAt = t.Unix()
		users = append(users, user)
	}
	return
}

func (*user) GetCreditDetailNum(userID string, year int, month, flag uint8) (num int, err error) {
	row := db.MysqlCli.QueryRow("select count(*) from shop_user "+
		"where open_id = ? and year(created_at) = ? and month(created_at) = ? and flag = ?",
		userID, year, month, flag)
	err = row.Scan(&num)
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

type boss struct {
}

var Boss = new(boss)

func (*boss) Add(boss model.Boss) (err error) {
	_, err = db.MysqlCli.Exec("insert into shop_boss (uid,open_id,order_id,amount,credit,multiple,extra_multiple,flag) values (?,?,?,?,?,?,?,?)",
		boss.UID, boss.OpenID, boss.OrderID, boss.Amount, boss.Credit, boss.Multiple, boss.ExtraMultiple, boss.Flag)
	return
}

func (*boss) GetCredit(bossID string) (offline, online float64, err error) {
	rows, err := db.MysqlCli.Query(
		"select flag,sum(credit) as all_credit from shop_boss where open_id = ? group by flag ", bossID)
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

func (*boss) GetAmount(bossID string) (offline, online float64, err error) {
	rows, err := db.MysqlCli.Query(
		"select flag,sum(amount) as all_amount from shop_boss where open_id = ? group by flag ", bossID)
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

func (*boss) GetCreditDetail(bossID string, year int, month, flag uint8, lastID, pageSize int) (users []model.Boss, err error) {
	var rows *sql.Rows
	if lastID == 0 {
		rows, err = db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,extra_multiple,flag,created_at from shop_boss "+
			"where open_id = ? and year(created_at) = ? and month(created_at) = ? and flag = ? order by id desc limit ?",
			bossID, year, month, flag, pageSize)
	} else {
		rows, err = db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,extra_multiple,flag,created_at from shop_boss "+
			"where open_id = ? and year(created_at) = ? and month(created_at) = ? and flag = ? and id < ? order by id desc limit ?",
			bossID, year, month, flag, lastID, pageSize)
	}

	if err != nil {
		return
	}

	for rows.Next() {
		var (
			user     model.Boss
			createAt string
		)
		err = rows.Scan(&user.ID, &user.OpenID, &user.Amount, &user.Credit, &user.OrderID, &user.Multiple,
			&user.ExtraMultiple, &user.Flag, &createAt)
		if err != nil {
			return
		}

		t, _ := time.Parse("2006-01-02 15:04:05", createAt)
		user.CreatedAt = t.Unix()
		users = append(users, user)
	}
	return
}

func (*boss) GetCreditDetailNum(bossID string, year int, month, flag uint8) (num int, err error) {
	row := db.MysqlCli.QueryRow("select count(*) from shop_boss "+
		"where open_id = ? and year(created_at) = ? and month(created_at) = ? and flag = ?",
		bossID, year, month, flag)
	err = row.Scan(&num)
	return
}

func (*boss) GetAllOnlineCredit() (credit float64, err error) {
	row := db.MysqlCli.QueryRow("select sum(credit) from shop_boss where flag = 2")
	err = row.Scan(&credit)
	return
}

func (*boss) GetBossNum() (num int, err error) {
	row := db.MysqlCli.QueryRow("select count(distinct open_id) from shop_boss where flag = 2")
	err = row.Scan(&num)
	return
}

func (*boss) ListOnlineCredit(openID string, pageNum, pageSize int) (rsp []model.ListBossCreditRsp, err error) {
	var sqlStr string
	if openID != "" {
		sqlStr = fmt.Sprintf("select open_id, sum(credit) as all_credit, count(*) as num "+
			"from shop_boss where flag = 2 and open_id = '%s' group by open_id limit %d,%d",
			openID, pageSize*(pageNum-1), pageSize)
	} else {
		sqlStr = fmt.Sprintf("select open_id, sum(credit) as all_credit, count(*) as num "+
			"from shop_boss where flag = 2 group by open_id limit %d,%d", pageSize*(pageNum-1), pageSize)
	}
	rows, err := db.MysqlCli.Query(sqlStr)
	if err != nil {
		return
	}

	for rows.Next() {
		var r model.ListBossCreditRsp
		err = rows.Scan(&r.OpenID, &r.AllCredit, &r.Num)
		if err != nil {
			return
		}
		rsp = append(rsp, r)
	}
	return
}

func (*boss) ListCreditDetail(bossID string, pageNum, pageSize int) (rsp []model.Boss, err error) {
	rows, err := db.MysqlCli.Query("select id,open_id,amount,credit,order_id,multiple,extra_multiple,flag,created_at "+
		"from shop_boss where open_id = ? and flag = 2 order by id desc limit ?,?", bossID, pageSize*(pageNum-1), pageSize)
	if err != nil {
		return
	}

	for rows.Next() {
		var (
			user     model.Boss
			createAt string
		)
		err = rows.Scan(&user.ID, &user.OpenID, &user.Amount, &user.Credit, &user.OrderID, &user.Multiple,
			&user.ExtraMultiple, &user.Flag, &createAt)
		if err != nil {
			return
		}

		t, _ := time.Parse("2006-01-02 15:04:05", createAt)
		user.CreatedAt = t.Unix()
		rsp = append(rsp, user)
	}
	return
}

func (*boss) GetBossRecordNum(bossID string) (num int, err error) {
	row := db.MysqlCli.QueryRow("select count(*) from shop_boss where open_id = ? and flag = 2", bossID)
	err = row.Scan(&num)
	return
}
