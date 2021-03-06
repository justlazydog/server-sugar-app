package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/internal/app/group"
	"server-sugar-app/internal/dao"

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

	if config.Server.IsJobServer { // offline biz
		r.GET("/test/lock/sie", func(c *gin.Context) {
			lockedSIE, err := dao.GetLockedSIE()
			if err != nil {
				c.JSON(400, gin.H{"err": err.Error()})
			}
			c.JSON(200, lockedSIE)
		})

		admin := r.Group("/admin")
		admin.POST("/prepare/:prepare", sugar.Prepare)

		admin.POST("/sugar/start", func(c *gin.Context) {
			go func() {
				skipRela := c.Query("skip_rela")
				if skipRela == "" {
					group.GetLatestGroupRela()
				}
				sugar.StartSugar()
			}()
		})

		admin.POST("/sugar/calc", func(c *gin.Context) {
			go func() {
				if err := sugar.CalcReward(time.Now()); err != nil {
					log.Errorf("CalcReward failed: %v", err.Error())
				}
			}()
			c.String(http.StatusOK, "ok")
		})

		admin.POST("/sugar/rewardDetail", func(c *gin.Context) {
			go func() {
				path := c.Query("path")
				detail, err := sugar.ParseRewardDetail(path)
				if err != nil {
					log.Error("ParseRewardDetail failed: %v", err)
					return
				}
				if err := sugar.SaveRewardDetail(detail); err != nil {
					log.Error("SaveRewardDetail failed: %v", err)
					return
				}
			}()
			c.String(http.StatusOK, "ok")
		})

		admin.GET("/relation/updated", func(c *gin.Context) {
			updated := "false"
			if group.RelateUpdated {
				updated = "true"
			}
			c.String(http.StatusOK, updated)
		})
	}
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

	shopGroup.GET("/user/used/list", shop.GetUserUsed)
	shopGroup.GET("/boss/used/list", shop.GetBossUsed)
	shopGroup.GET("/used/detail/list", shop.GetUsedDetail)

	sugarGroup := r.Group("/sugar")
	sugarGroup.POST("/upload/:token/:filename", sugar.ReceiveCalcFile)
	sugarGroup.GET("/download/:filename", sugar.DownloadRewardFile)
	sugarGroup.GET("/reward_detail", sugar.GetUserRewardDetail)

	outGroup := r.Group("/out")
	outGroup.Use(middleware.ValidateSign)
	outGroup.PUT("/order", out.Put)
	outGroup.PUT("/order/boss", out.NewPut)
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
