package service

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
	"server-sugar-app/internal/app/shop"
)

var srv *http.Server

func RunHttp() {
	r := gin.Default()
	r.PUT("/shop/order", shop.Put)
	r.GET("/shop/user/credit", shop.GetUserCredit)
	r.GET("/shop/user/credit/detail", shop.GetUserCreditDetail)
	r.GET("/shop/used", shop.GetUsedAmount)
	r.GET("/shop/boss/credit", shop.GetBossCredit)
	r.GET("/shop/boss/credit/detail", shop.GetBossCreditDetail)
	r.GET("/shop/boss/credit/list", shop.ListBossCredit)
	r.GET("/shop/boss/credit/detail/list", shop.ListBossCreditDetail)

	// r.POST("/sugar/upload/:token/:filename", sugar.ReceiveCalcFile)

	srv = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port),
		Handler: r,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}

func GetHttp() *http.Server {
	return srv
}
