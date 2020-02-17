package config

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var (
	Server server
	MySql  mysql
	SIE    sie
)

// Server 配置
type server struct {
	Env        string `yaml:"env"`
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	DomainName string `yaml:"domain_name"`
}

// MySQL 配置
// type mysqls struct {
// 	OpenCloud   mysql `yaml:"open_cloud"`
// 	OfflineShop mysql `yaml:"offline_shop"`
// 	OnlineShop  mysql `yaml:"online_shop"`
// }

type mysql struct {
	Host         string `yaml:"host"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	Charset      string `yaml:"charset"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

// SIE 糖果相关配置
type Sugar struct {
	Origin  string `yaml:"origin"`
	Request string `yaml:"request"`
}

type Perf struct {
	Origin  string `yaml:"origin"`
	Request string `yaml:"request"`
}

type sie struct {
	SIESchedule       string   `yaml:"sie_schedule"`
	SIERewardAccounts []string `yaml:"sie_reward_accounts"`
	SIEWhiteList      []string `yaml:"sie_white_list"`
	SIEAddAccounts    []string `yaml:"sie_add_accounts"`
	SIESubAccounts    []string `yaml:"sie_sub_accounts"`
	Sugars            []Sugar  `yaml:"sugars"`
	Perfs             []Perf   `yaml:"perfs"`
}

func Init() {
	unmarshalServer()
	unmarshalMysql()
	unmarshalSIE()
}

func unmarshalServer() {
	viper.SetConfigName("server")
	viper.AddConfigPath("configs/")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	err = viper.Unmarshal(&Server, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	})
	if err != nil {
		panic(fmt.Errorf("Fatal error unmarshal config file: %s \n", err))
	}
}

func unmarshalMysql() {
	viper.SetConfigName("mysql")
	viper.AddConfigPath("configs/")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	err = viper.Unmarshal(&MySql, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	})
	if err != nil {
		panic(fmt.Errorf("Fatal error unmarshal config file: %s \n", err))
	}
}

func unmarshalSIE() {
	viper.SetConfigName("sie")
	viper.AddConfigPath("configs/")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	err = viper.Unmarshal(&SIE, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	})
	if err != nil {
		panic(fmt.Errorf("Fatal error unmarshal config file: %s \n", err))
	}
}
