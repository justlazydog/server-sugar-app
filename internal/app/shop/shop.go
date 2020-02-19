package shop

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/internal/dao"
	"server-sugar-app/internal/model"
	"server-sugar-app/internal/pkg/generr"
)

const (
	ExtraMultiple = 1
	BossMultiple  = 2
	UserMultiple  = 10
)

func Put(c *gin.Context) {
	req := struct {
		UserID  string `json:"user_id" form:"user_id" binding:"required"`   // 用户ID
		BossID  string `json:"boss_id" form:"boss_id" binding:"required"`   // 商户ID
		OrderID string `json:"order_id" form:"order_id" binding:"required"` // 挂单ID
		Amount  string `json:"amount" form:"amount" binding:"required"`     // 销毁金额
		Flag    uint8  `json:"flag" form:"flag" binding:"required"`         // 1-线下 2-线上
	}{}

	err := c.ShouldBind(&req)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "convert string to float"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	userUID, err := dao.Oauth.GetUID(req.UserID)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get uid from open-cloud"))
		c.JSON(http.StatusInternalServerError, generr.ReadDB)
		return
	}
	if userUID == "" {
		log.Errorf("err: %+v", errors.Errorf("user_id: %s query no uid", userUID))
		c.JSON(http.StatusBadRequest, generr.SugarNoTargetUser)
		return
	}

	bossUID, err := dao.Oauth.GetUID(req.BossID)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get uid from open-cloud"))
		c.JSON(http.StatusInternalServerError, generr.ReadDB)
		return
	}
	if bossUID == "" {
		log.Errorf("err: %+v", errors.Errorf("user_id: %s query no uid", bossUID))
		c.JSON(http.StatusBadRequest, generr.SugarNoTargetUser)
		return
	}

	user := model.User{
		UID:           userUID,
		OpenID:        req.UserID,
		OrderID:       req.OrderID,
		Amount:        amount,
		Credit:        amount * UserMultiple * ExtraMultiple,
		Multiple:      UserMultiple,
		ExtraMultiple: ExtraMultiple,
		Flag:          req.Flag,
	}

	err = dao.User.Add(user)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "add user record"))
		c.JSON(http.StatusBadRequest, generr.UpdateDB)
		return
	}

	shop := model.Boss{
		UID:           bossUID,
		OpenID:        req.BossID,
		OrderID:       req.OrderID,
		Amount:        amount,
		Credit:        amount * BossMultiple * ExtraMultiple,
		Multiple:      BossMultiple,
		ExtraMultiple: ExtraMultiple,
		Flag:          req.Flag,
	}
	err = dao.Shop.Add(shop)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "add shop record"))
		c.JSON(http.StatusBadRequest, generr.UpdateDB)
		return
	}

	c.JSON(http.StatusOK, struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}{200, "success"})
	return
}

func GetUserCredit(c *gin.Context) {
	req := struct {
		UserID string `form:"user_id" binding:"required"`
	}{}

	err := c.ShouldBind(&req)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	offline, online, err := dao.User.GetCredit(req.UserID)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get credit"))
		c.JSON(http.StatusInternalServerError, generr.ReadDB)
		return
	}

	m := make(map[string]interface{})
	m["offline"] = offline
	m["online"] = online
	c.JSON(http.StatusOK, struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}{200, "success", m})
	return
}

func GetUserCreditDetail(c *gin.Context) {
	req := struct {
		UserID   string `form:"user_id" binding:"required"`
		Year     int    `form:"year" binding:"required"`
		Month    uint8  `form:"month" binding:"required"`
		Flag     uint8  `form:"flag" binding:"required"`
		LastID   int    `form:"last_id"`
		PageSize int    `form:"page_size"`
	}{}

	err := c.ShouldBind(&req)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	if req.PageSize == 0 {
		req.PageSize = 10
	}

	users, err := dao.User.GetCreditDetail(req.UserID, req.Year, req.Month, req.Flag, req.LastID, req.PageSize)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get credit detail"))
		c.JSON(http.StatusInternalServerError, generr.ReadDB)
		return
	}

	c.JSON(http.StatusOK, struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}{200, "success", users})
	return
}

func GetUsedAmount(c *gin.Context) {
	offline, online, err := dao.User.GetUsedAmount()
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get used amount"))
		c.JSON(http.StatusInternalServerError, generr.ReadDB)
		return
	}

	m := make(map[string]interface{})
	m["offline"] = offline
	m["online"] = online
	c.JSON(http.StatusOK, struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}{200, "success", m})
}

func GetBossCredit(c *gin.Context) {
	req := struct {
		BossID string `form:"boss_id" binding:"required"`
	}{}

	err := c.ShouldBind(&req)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	offline, online, err := dao.Shop.GetCredit(req.BossID)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get credit"))
		c.JSON(http.StatusInternalServerError, generr.ReadDB)
		return
	}

	m := make(map[string]interface{})
	m["offline"] = offline
	m["online"] = online
	c.JSON(http.StatusOK, struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}{200, "success", m})
	return
}

func GetBossCreditDetail(c *gin.Context) {
	req := struct {
		BossID   string `form:"boss_id" binding:"required"`
		Year     int    `form:"year" binding:"required"`
		Month    uint8  `form:"month" binding:"required"`
		Flag     uint8  `form:"flag" binding:"required"`
		LastID   int    `form:"last_id"`
		PageSize int    `form:"page_size"`
	}{}

	err := c.ShouldBind(&req)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	boss, err := dao.Shop.GetCreditDetail(req.BossID, req.Year, req.Month, req.Flag, req.LastID, req.PageSize)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get credit detail"))
		c.JSON(http.StatusInternalServerError, generr.ReadDB)
		return
	}

	c.JSON(http.StatusOK, struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}{200, "success", boss})
	return
}
