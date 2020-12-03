package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"server-sugar-app/config"
)

var (
	GormDB           *gorm.DB
	MysqlCli         *sql.DB
	ExchangeMysqlCli *sql.DB
)

func Init() {
	connMysql()
	connMysqlInGorm()
	connExchangeMysql()
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

func connMysqlInGorm() {
	var err error

	mysqlCfg := config.MySql
	dsn := fmt.Sprintf("%s:%s@(%s)/%s?charset=%s&parseTime=true&loc=UTC", mysqlCfg.User, mysqlCfg.Password,
		mysqlCfg.Host, mysqlCfg.Database, mysqlCfg.Charset)
	GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(err)
	}

	sqlDB, err := GormDB.DB()
	if err != nil {
		panic(err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(10 * time.Minute)

	return
}

func connExchangeMysql() {
	var err error
	mysqlCfg := config.ExchangeMysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", mysqlCfg.User, mysqlCfg.Password,
		mysqlCfg.Host, mysqlCfg.Database)
	ExchangeMysqlCli, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Error("Connect mysql error: ", err, " Connect dsn: ", dsn)
		panic(err)
	} else {
		log.Infof("conn mysql %s success", dsn)
	}
	return
}
