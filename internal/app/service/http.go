package service

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
	"server-sugar-app/internal/app/out"
	"server-sugar-app/internal/app/shop"
	"server-sugar-app/internal/app/sugar"
	"server-sugar-app/internal/pkg/middleware"
)

var srv *http.Server

func RunHttp() {
	r := gin.Default()
	pprof.Register(r)

	shopGroup := r.Group("/shop")
	shopGroup.PUT("/order", shop.Put)
	shopGroup.GET("/user/credit", shop.GetUserCredit)
	shopGroup.GET("/user/credit/all", shop.GetAllUserCredit)
	shopGroup.GET("/user/credit/detail", shop.GetUserCreditDetail)
	shopGroup.GET("/used", shop.GetUsedAmount)
	shopGroup.GET("/boss/amount", shop.GetBossAmount)
	shopGroup.GET("/boss/credit", shop.GetBossCredit)
	shopGroup.GET("/boss/credit/detail", shop.GetBossCreditDetail)
	shopGroup.GET("/boss/credit/list", shop.ListBossCredit)
	shopGroup.GET("/boss/credit/detail/list", shop.ListBossCreditDetail)

	sugarGroup := r.Group("/sugar")
	sugarGroup.POST("/upload/:token/:filename", sugar.ReceiveCalcFile)
	sugarGroup.POST("/start/manual", sugar.ManualStart)
	sugarGroup.GET("/download/:filename", sugar.DownloadRewardFile)

	outGroup := r.Group("/out")
	outGroup.Use(middleware.ValidateSign)
	outGroup.PUT("/order", out.Put)
	outGroup.GET("/amount", out.GetUserSumDestructAmount)

	srv = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port),
		Handler: r,
	}

	log.Infof("Start to listen %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}

func GetHttp() *http.Server {
	return srv
}
