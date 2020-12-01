package config

import (
	"flag"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "conf", "configs/", "default config path")
}

var (
	Server        server
	MySql         mysql
	ExchangeMysql mysql
	SIE           sie
)

// Server 配置
type server struct {
	Env          string `yaml:"env"`
	Host         string `yaml:"host"`
	Port         string `yaml:"port"`
	DomainName   string `yaml:"domain_name"`
	OpenCloud    string `yaml:"open_cloud"`
	OTCHost      string `yaml:"otc_host"`
	MerchantUUID string `yaml:"merchant_uuid"`
}

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

type sie struct {
	SIESchedule       string   `yaml:"sie_schedule"`
	SIERewardAccounts []string `yaml:"sie_reward_accounts"`
	SIEWhiteList      []string `yaml:"sie_white_list"`
	SIEAddAccounts    []string `yaml:"sie_add_accounts"`
	SIESubAccounts    []string `yaml:"sie_sub_accounts"`
	Sugars            []Sugar  `yaml:"sugars"`
	SIERewardAccount  string   `yaml:"sie_reward_account"`
}

func Init() {
	unmarshalServer()
	unmarshalMysql()
	unmarshalExchangeMysql()
	unmarshalSIE()
}

func unmarshalServer() {
	viper.SetConfigName("server")
	viper.AddConfigPath(confPath)
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
	viper.AddConfigPath(confPath)
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

func unmarshalExchangeMysql() {
	viper.SetConfigName("exchange_mysql")
	viper.AddConfigPath(confPath)
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	err = viper.Unmarshal(&ExchangeMysql, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	})
	if err != nil {
		panic(fmt.Errorf("Fatal error unmarshal config file: %s \n", err))
	}
}

func unmarshalSIE() {
	viper.SetConfigName("sie")
	viper.AddConfigPath(confPath)
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
