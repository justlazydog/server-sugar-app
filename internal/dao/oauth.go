package dao

import (
	"server-sugar-app/config"
	"server-sugar-app/internal/db"
)

type oauth struct {
}

var Oauth = new(oauth)

func (*oauth) GetUID(openID string) (uid string, err error) {
	var appID string
	if config.Server.Env == "test" {
		appID = "04565e551f7ff066"
	} else {
		appID = "576ae8b341e42274"
	}

	row := db.MysqlCli.QueryRow("select uid from oauth where open_id = ? and app_id = ?",
		openID, appID)
	err = row.Scan(&uid)
	return
}

func (*oauth) GetUIDByAppID(openID, appID string) (uid string, err error) {
	row := db.MysqlCli.QueryRow("select uid from oauth where open_id = ? and app_id = ?",
		openID, appID)
	err = row.Scan(&uid)
	return
}

func (*oauth) GetOpenIDByAppID(UID, appID string) (openID string, err error) {
	row := db.MysqlCli.QueryRow("select open_id from oauth where uid = ? and app_id = ?",
		UID, appID)
	err = row.Scan(&openID)
	return
}
