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
		// appID = "04565e551f7ff066"
		appID = "zd7b3n7nazce89bf"
	} else {
		appID = "zd7b3n7nazce89bf"
	}

	row := db.MysqlCli.QueryRow("select uid from oauth where open_id = ? and app_id = ?",
		openID, appID)
	err = row.Scan(&uid)
	return
}
