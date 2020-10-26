package dao

import (
	"server-sugar-app/internal/db"
	"server-sugar-app/internal/model"
)

type userInfo struct {
}

var UserInfo = new(userInfo)

func (*userInfo) Create(userInfo model.UserInfo) error {
	_, err := db.MysqlCli.Exec(`
insert into user_info 
	(uid, growth_rate) 
values 
	(?,?)`,
		userInfo.UID, userInfo.GrowthRate)
	return err
}

func (*userInfo) Update(userInfo model.UserInfo) error {
	_, err := db.MysqlCli.Exec(`
update user_info 
set 
	growth_rate = ? 
where
	uid = ?`,
		userInfo.GrowthRate, userInfo.UID)
	return err
}

func (*userInfo) GetByUID(uid string) (model.UserInfo, error) {
	result := model.UserInfo{}
	row := db.MysqlCli.QueryRow(`
select 
	uid, growth_rate
from
	user_info
where uid = ?`, uid)

	err := row.Scan(&result.UID, &result.GrowthRate)
	return result, err
}
