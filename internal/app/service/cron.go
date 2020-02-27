package service

import (
	"github.com/robfig/cron"

	"server-sugar-app/config"
	"server-sugar-app/internal/app/group"
	"server-sugar-app/internal/app/sugar"
)

func SugarTicker() {
	c := cron.New()
	sieCfg := config.SIE
	_ = c.AddFunc(sieCfg.SIESchedule, sugar.StartSugar)
	_ = c.AddFunc(sieCfg.SIESchedule, group.GetLatestGroupRela)
	c.Start()
}
