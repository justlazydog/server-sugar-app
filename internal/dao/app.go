package dao

import (
	"server-sugar-app/internal/db"
)

type app struct {
}

var App = new(app)

func (*app) GetKey(appID string) (key string, err error) {
	row := db.MysqlCli.QueryRow("select pay_secret from app where app_id = ?", appID)
	err = row.Scan(&key)
	return
}
