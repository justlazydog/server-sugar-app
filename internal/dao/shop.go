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
	_, err = db.MysqlCli.Exec("insert into shop_user (app_id,uid,open_id,order_id,amount,credit,multiple,extra_multiple,flag) values (?,?,?,?,?,?,?,?,?)",
		user.AppID, user.UID, user.OpenID, user.OrderID, user.Amount, user.Credit, user.Multiple, user.ExtraMultiple, user.Flag)
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

func (*user) GetAmount(appID, openID string) (amount float64, err error) {
	row := db.MysqlCli.QueryRow("select sum(amount) as all_amount from shop_user where app_id = ? and open_id = ?", appID, openID)

	err = row.Scan(&amount)
	if err != nil {
		return
	}
	return
}

func (*user) QueryDestroyedAmountGroupByUID(beginAt time.Time) ([]model.User, error) {
	result := make([]model.User, 0)
	var rows *sql.Rows
	rows, err := db.MysqlCli.Query(`
select 
	uid, sum(credit) as credit
from 
	shop_user 
where 
	created_at > ?
group by
	uid`, beginAt)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var (
			user model.User
		)
		err = rows.Scan(&user.UID, &user.Credit)
		if err != nil {
			return nil, err
		}
		result = append(result, user)
	}
	return result, nil
}

func (*user) ListAppUsed(userID string, size, num int) ([]model.AppUsed, error) {
	rows, err := db.MysqlCli.Query(`select shop_user.app_id, app_name, sum(amount) amount, sum(credit) credit
from shop_user
         inner join app
                    on shop_user.app_id = app.app_id
                        and uid = ?
group by app_id
limit ? offset ?;`, userID, size, num-1)
	if err != nil {
		return nil, err
	}

	var ress []model.AppUsed
	for rows.Next() {
		var res model.AppUsed
		err = rows.Scan(&res.AppID, &res.AppName, &res.Amount, &res.Credit)
		if err != nil {
			return nil, err
		}
		ress = append(ress, res)
	}

	return ress, nil
}

func (*user) ListAppUsedDetail(userID, appID string, size, num int) ([]model.AppUsed, error) {
	rows, err := db.MysqlCli.Query(`select created_at, shop_user.app_id, app_name, amount, credit
from shop_user
         inner join app
                    on shop_user.app_id = app.app_id
                        and uid = ? and shop_user.app_id = ?
limit ? offset ?;`, userID, appID, size, num-1)
	if err != nil {
		return nil, err
	}

	var ress []model.AppUsed
	for rows.Next() {
		var (
			createdAt string
			res       model.AppUsed
		)
		err = rows.Scan(&createdAt, &res.AppID, &res.AppName, &res.Amount, &res.Credit)
		if err != nil {
			return nil, err
		}
		t, _ := time.Parse("2006-01-02 15:04:05", createdAt)
		res.CreatedAt = t.Unix()
		ress = append(ress, res)
	}

	return ress, nil
}

type boss struct {
}

var Boss = new(boss)

func (*boss) Add(boss model.Boss) (err error) {
	_, err = db.MysqlCli.Exec("insert into shop_boss (app_id,uid,open_id,order_id,amount,credit,multiple,extra_multiple,flag) values (?,?,?,?,?,?,?,?,?)",
		boss.AppID, boss.UID, boss.OpenID, boss.OrderID, boss.Amount, boss.Credit, boss.Multiple, boss.ExtraMultiple, boss.Flag)
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

func (*boss) GetAllCredit() (credit float64, err error) {
	row := db.MysqlCli.QueryRow("select sum(credit) from shop_boss where flag = 2")
	err = row.Scan(&credit)
	return
}

func (*boss) GetBossNum() (num int, err error) {
	row := db.MysqlCli.QueryRow("select count(distinct open_id) from shop_boss where flag = 2")
	err = row.Scan(&num)
	return
}

func (*boss) ListCredit(pageNum, pageSize int) (rsp []model.ListBossCreditRsp, err error) {
	rows, err := db.MysqlCli.Query("select open_id, sum(credit) as all_credit, count(*) as num "+
		"from shop_boss where flag = 2 group by open_id limit ?,?", pageSize*(pageNum-1), pageSize)
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

func (*boss) QueryDestroyedAmountGroupByBossID(beginAt time.Time) ([]model.Boss, error) {
	result := make([]model.Boss, 0)
	var rows *sql.Rows
	rows, err := db.MysqlCli.Query(`
select
	uid, sum(credit) as credit
from
	shop_boss 
where
	created_at > ?
group by uid`, beginAt)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var (
			user model.Boss
		)
		err = rows.Scan(&user.UID, &user.Credit)
		if err != nil {
			return nil, err
		}
		result = append(result, user)
	}
	return result, nil
}

func (*boss) ListAppUsed(userID string, size, num int) ([]model.AppUsed, error) {
	rows, err := db.MysqlCli.Query(`select shop_boss.app_id, app_name, sum(amount) amount, sum(credit) credit
from shop_boss
         inner join app
                    on shop_boss.app_id = app.app_id
                        and uid = ?
group by app_id
limit ? offset ?;`, userID, size, num-1)
	if err != nil {
		return nil, err
	}

	var ress []model.AppUsed
	for rows.Next() {
		var res model.AppUsed
		err = rows.Scan(&res.AppID, &res.AppName, &res.Amount, &res.Credit)
		if err != nil {
			return nil, err
		}
		ress = append(ress, res)
	}

	return ress, nil
}

func (*boss) ListAppUsedDetail(userID, appID string, size, num int) ([]model.AppUsed, error) {
	rows, err := db.MysqlCli.Query(`select created_at, shop_boss.app_id, app_name, amount, credit
from shop_boss
         inner join app
                    on shop_boss.app_id = app.app_id
                        and uid = ? and shop_boss.app_id = ?
limit ? offset ?;`, userID, appID, size, num-1)
	if err != nil {
		return nil, err
	}

	var ress []model.AppUsed
	for rows.Next() {
		var (
			createdAt string
			res       model.AppUsed
		)
		err = rows.Scan(&createdAt, &res.AppID, &res.AppName, &res.Amount, &res.Credit)
		if err != nil {
			return nil, err
		}
		t, _ := time.Parse("2006-01-02 15:04:05", createdAt)
		res.CreatedAt = t.Unix()
		ress = append(ress, res)
	}

	return ress, nil
}
