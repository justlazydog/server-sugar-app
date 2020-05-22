package out

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
	"server-sugar-app/internal/dao"
	"server-sugar-app/internal/model"
	"server-sugar-app/internal/pkg/generr"
	"server-sugar-app/internal/pkg/util"
)

const (
	ExtraMultiple = 1
	UserMultiple  = 10

	SIE = "sie"

	PayUrl = "/payment/create"

	Remark = "第三方销毁金额"
)

func Put(c *gin.Context) {
	req := struct {
		AppID        string  `form:"app_id" binding:"required"`        // 应用ID
		OpenID       string  `form:"open_id" binding:"required"`       // 用户ID
		OrderID      string  `form:"order_id" binding:"required"`      // 挂单ID
		Amount       float64 `form:"amount" binding:"required"`        // 挂单金额
		MerchantUUID string  `form:"merchant_uuid" binding:"required"` // 商户号
		Token        string  `form:"token" binding:"required"`         // 币种
		Rate         float64 `form:"rate" binding:"required"`          // 销毁比例
	}{}

	err := c.ShouldBindWith(&req, binding.Form)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	log.Infof("req: %+v", req)

	if strings.ToLower(req.Token) != SIE {
		req.Amount = req.Amount * req.Rate
	}

	err = deductDestructAmount(req.AppID, req.OpenID, req.OrderID, req.MerchantUUID, req.Token, Remark, req.Amount)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "deduct destruct"))
		c.JSON(http.StatusInternalServerError, generr.ServerError)
		return
	}

	userUID, err := dao.Oauth.GetUIDByAppID(req.OpenID, req.AppID)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get uid from open-cloud"))
		c.JSON(http.StatusInternalServerError, generr.ServerError)
		return
	}
	if userUID == "" {
		log.Errorf("err: %+v", errors.Errorf("user_id: %s query no uid", userUID))
		c.JSON(http.StatusBadRequest, generr.SugarNoTargetUser)
		return
	}

	user := model.User{
		UID:           userUID,
		OpenID:        req.OpenID,
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Credit:        req.Amount * UserMultiple * ExtraMultiple,
		Multiple:      UserMultiple,
		ExtraMultiple: ExtraMultiple,
	}

	err = dao.User.Add(user)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "add user record"))
		c.JSON(http.StatusBadRequest, generr.UpdateDB)
		return
	}

	c.JSON(http.StatusOK, struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}{200, "success", map[string]interface{}{
		"sie": req.Amount,
	}})
	return
}

func deductDestructAmount(appID, openID, orderID, merchantUUID, token, remark string, value float64) (err error) {
	key, err := dao.App.GetKey(appID)
	if err != nil {
		return
	}

	form := url.Values{}
	form.Set("app_id", appID)
	form.Set("open_id", openID)
	form.Set("order_id", orderID)
	form.Set("merchant_uuid", merchantUUID)
	form.Set("token", token)
	form.Set("remark", remark)
	form.Set("pay_type", "22")
	form.Set("amount", fmt.Sprintf("%f", value))
	form.Set("t", util.ToString(time.Now().Unix()))
	form.Set("s", util.GenSignCode(form, key))

	req, err := http.NewRequest(http.MethodPost, config.Server.OpenCloud+PayUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: 10 * time.Second}
	rsp, err := client.Do(req)
	if err != nil {
		return
	}

	type result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	var res result

	decoder := json.NewDecoder(rsp.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return
	}

	if rsp.StatusCode != 200 || res.Code != 200 {
		err = errors.New(res.Msg)
		return
	}
	return
}
