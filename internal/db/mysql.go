package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
)

var (
	MysqlCli *sql.DB
)

func Init() {
	connMysql()
}

func connMysql() {
	var err error
	mysqlCfg := config.MySql
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", mysqlCfg.User, mysqlCfg.Password,
		mysqlCfg.Host, mysqlCfg.Database)
	MysqlCli, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Error("Connect mysql error: ", err, " Connect dsn: ", dsn)
		panic(err)
	} else {
		log.Infof("conn mysql %s success", dsn)
	}
	return
}
