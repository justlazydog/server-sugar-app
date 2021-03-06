package out

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
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
	BossMultiple  = 2
	UserMultiple  = 10

	SIE  = "SIE"
	SUSD = "SUSD"
	CNY  = "CNY"

	PayUrl            = "/payment/create"
	getLatestPriceUrl = "/api/getLatestPrice"
	getMarketInfo     = "/api/ctc/tickers/market"

	Remark = "第三方销毁金额"
)

func GetUserSumDestructAmount(c *gin.Context) {
	req := struct {
		AppID  string `form:"app_id" binding:"required"` // 应用ID
		OpenID string `form:"open_id" binding:"required"`
	}{}

	err := c.ShouldBindWith(&req, binding.Form)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	log.Infof("req: %+v", req)

	amount, err := dao.User.GetAmount(req.AppID, req.OpenID)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get user used amount"))
		c.JSON(http.StatusBadRequest, generr.ReadDB)
		return
	}

	c.JSON(http.StatusOK, struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}{200, "success", map[string]interface{}{
		"amount": amount,
	}})
	return
}

func Put(c *gin.Context) {
	req := struct {
		AppID        string  `form:"app_id" binding:"required"`        // 应用ID
		OpenID       string  `form:"open_id" binding:"required"`       // 用户ID
		OrderID      string  `form:"order_id" binding:"required"`      // 挂单ID
		Amount       float64 `form:"amount" binding:"required"`        // 挂单数量
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

	var amount string
	if strings.ToUpper(req.Token) == CNY {
		sieAmount, err := cnyToSie(req.Amount * req.Rate)
		if err != nil {
			log.Errorf("err: %+v", errors.Wrap(err, "transfer cny to sie"))
			c.JSON(http.StatusInternalServerError, generr.CnyToSieErr)
			return
		}
		amount = fmt.Sprintf("%.5f", math.Ceil(sieAmount*100000)/100000)
	} else if strings.ToUpper(req.Token) == SUSD {
		sieAmount, err := susdToSie(req.Amount * req.Rate)
		if err != nil {
			log.Errorf("err: %+v", errors.Wrap(err, "transfer susd to sie"))
			c.JSON(http.StatusInternalServerError, generr.SusdToSieErr)
			return
		}
		amount = fmt.Sprintf("%.5f", math.Ceil(sieAmount*100000)/100000)
	} else {
		amount = fmt.Sprintf("%.5f", math.Ceil(req.Amount*req.Rate*100000)/100000)
	}

	err = deductDestructAmount(config.Server.MerchantUUID, req.OrderID, req.MerchantUUID, SIE, Remark, amount)
	if err != nil {
		if strings.Contains(err.Error(), "enough") {
			log.Errorf("err: %+v", errors.Wrap(err, "deduct destruct"))
			c.JSON(http.StatusInternalServerError, generr.BalanceNotEnough)
			return
		}
		log.Errorf("err: %+v", errors.Wrap(err, "deduct destruct"))
		c.JSON(http.StatusInternalServerError, generr.DestructAmountError)
		return
	}

	userUID, err := dao.Oauth.GetUIDByAppID(req.OpenID, req.AppID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, generr.SugarNoTargetUser)
			return
		}
		log.Errorf("err: %+v", errors.Wrap(err, "get uid from open-cloud"))
		c.JSON(http.StatusInternalServerError, generr.ServerError)
		return
	}

	bossOpenID, err := dao.Oauth.GetOpenIDByAppID(req.MerchantUUID, req.AppID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, generr.SugarNoTargetUser)
			return
		}
		log.Errorf("err: %+v", errors.Wrap(err, "get open_id from open-cloud"))
		c.JSON(http.StatusInternalServerError, generr.ServerError)
		return
	}

	req.Amount, _ = strconv.ParseFloat(amount, 64)

	user := model.User{
		AppID:         req.AppID,
		UID:           userUID,
		OpenID:        req.OpenID,
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Credit:        req.Amount * UserMultiple * ExtraMultiple,
		Multiple:      UserMultiple,
		ExtraMultiple: ExtraMultiple,
		Flag:          1,
	}

	err = dao.User.Add(user)
	if err != nil {
		if strings.Contains(err.Error(), "Error 1062") {
			c.JSON(http.StatusBadRequest, generr.RepeatOrderID)
			return
		}
		log.Errorf("err: %+v", errors.Wrap(err, "add user record"))
		c.JSON(http.StatusBadRequest, generr.ServerError)
		return
	}

	boss := model.Boss{
		AppID:         req.AppID,
		UID:           req.MerchantUUID,
		OpenID:        bossOpenID,
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Credit:        req.Amount * BossMultiple * ExtraMultiple,
		Multiple:      BossMultiple,
		ExtraMultiple: ExtraMultiple,
		Flag:          1,
	}
	err = dao.Boss.Add(boss)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "add shop record"))
		c.JSON(http.StatusBadRequest, generr.UpdateDB)
		return
	}

	log.Infof("rsp :%+v", struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}{200, "success", map[string]interface{}{
		"sie": amount,
	}})

	c.JSON(http.StatusOK, struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}{200, "success", map[string]interface{}{
		"sie": amount,
	}})
	return
}

func NewPut(c *gin.Context) {
	req := struct {
		AppID        string  `form:"app_id" binding:"required"`   // 应用ID
		OpenID       string  `form:"open_id" binding:"required"`  // 用户ID
		BossID       string  `form:"boss_id"`                     // 商家ID
		MerchantUUID string  `form:"merchant_uuid" `              // 商户号
		OrderID      string  `form:"order_id" binding:"required"` // 挂单ID
		Amount       float64 `form:"amount" binding:"required"`   // 挂单数量
		Token        string  `form:"token" binding:"required"`    // 币种
		Rate         float64 `form:"rate" binding:"required"`     // 销毁比例
	}{}

	err := c.ShouldBindWith(&req, binding.Form)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	if req.BossID == "" && req.MerchantUUID == "" {
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	log.Infof("req: %+v", req)

	var amount string
	if strings.ToUpper(req.Token) == CNY {
		sieAmount, err := cnyToSie(req.Amount * req.Rate)
		if err != nil {
			log.Errorf("err: %+v", errors.Wrap(err, "transfer cny to sie"))
			c.JSON(http.StatusInternalServerError, generr.CnyToSieErr)
			return
		}
		amount = fmt.Sprintf("%.5f", math.Ceil(sieAmount*100000)/100000)
	} else if strings.ToUpper(req.Token) == SUSD {
		sieAmount, err := susdToSie(req.Amount * req.Rate)
		if err != nil {
			log.Errorf("err: %+v", errors.Wrap(err, "transfer susd to sie"))
			c.JSON(http.StatusInternalServerError, generr.SusdToSieErr)
			return
		}
		amount = fmt.Sprintf("%.5f", math.Ceil(sieAmount*100000)/100000)
	} else {
		amount = fmt.Sprintf("%.5f", math.Ceil(req.Amount*req.Rate*100000)/100000)
	}

	err = deductDestructAmount(config.Server.MerchantUUID, req.OrderID, req.MerchantUUID, SIE, Remark, amount)
	if err != nil {
		if strings.Contains(err.Error(), "enough") {
			log.Errorf("err: %+v", errors.Wrap(err, "deduct destruct"))
			c.JSON(http.StatusInternalServerError, generr.BalanceNotEnough)
			return
		}
		log.Errorf("err: %+v", errors.Wrap(err, "deduct destruct"))
		c.JSON(http.StatusInternalServerError, generr.DestructAmountError)
		return
	}

	userUID, err := dao.Oauth.GetUIDByAppID(req.OpenID, req.AppID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, generr.SugarNoTargetUser)
			return
		}
		log.Errorf("err: %+v", errors.Wrap(err, "get uid from open-cloud"))
		c.JSON(http.StatusInternalServerError, generr.ServerError)
		return
	}

	var bossUID, boosOpenID string
	if req.BossID != "" {
		bossUID, err = dao.Oauth.GetUIDByAppID(req.BossID, req.AppID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusInternalServerError, generr.SugarNoTargetUser)
				return
			}
			log.Errorf("err: %+v", errors.Wrap(err, "get uid from open-cloud"))
			c.JSON(http.StatusInternalServerError, generr.ServerError)
			return
		}
		boosOpenID = req.BossID
	} else {
		boosOpenID, err = dao.Oauth.GetOpenIDByAppID(req.MerchantUUID, req.AppID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusInternalServerError, generr.SugarNoTargetUser)
				return
			}
			log.Errorf("err: %+v", errors.Wrap(err, "get open_id from open-cloud"))
			c.JSON(http.StatusInternalServerError, generr.ServerError)
			return
		}
		bossUID = req.MerchantUUID
	}

	req.Amount, _ = strconv.ParseFloat(amount, 64)
	user := model.User{
		AppID:         req.AppID,
		UID:           userUID,
		OpenID:        req.OpenID,
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Credit:        req.Amount * UserMultiple * ExtraMultiple,
		Multiple:      UserMultiple,
		ExtraMultiple: ExtraMultiple,
		Flag:          1,
	}

	err = dao.User.Add(user)
	if err != nil {
		if strings.Contains(err.Error(), "Error 1062") {
			c.JSON(http.StatusBadRequest, generr.RepeatOrderID)
			return
		}
		log.Errorf("err: %+v", errors.Wrap(err, "add user record"))
		c.JSON(http.StatusBadRequest, generr.ServerError)
		return
	}

	boss := model.Boss{
		AppID:         req.AppID,
		UID:           bossUID,
		OpenID:        boosOpenID,
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Credit:        req.Amount * BossMultiple * ExtraMultiple,
		Multiple:      BossMultiple,
		ExtraMultiple: ExtraMultiple,
		Flag:          1,
	}
	err = dao.Boss.Add(boss)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "add shop record"))
		c.JSON(http.StatusBadRequest, generr.UpdateDB)
		return
	}

	c.JSON(http.StatusOK, struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}{200, "success", map[string]interface{}{
		"sie":         amount,
		"user_credit": math.Ceil(user.Credit*100000) / 100000,
		"boss_credit": math.Ceil(boss.Credit*100000) / 100000,
	}})
	return
}

func deductDestructAmount(openID, orderID, merchantUUID, token, remark, value string) (err error) {
	var appID string
	if config.Server.Env == "test" {
		appID = "04565e551f7ff066"
	} else {
		appID = "576ae8b341e42274"
	}

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
	form.Set("amount", value)
	form.Set("flag", "1")
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
	defer rsp.Body.Close()

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

func cnyToSie(cnyVolume float64) (sieVolume float64, err error) {
	var groupID string
	if config.Server.Env == "test" {
		groupID = "3eb59eae-1f3a-45f9-a8bf-16ec65d0e7c9"
	} else {
		groupID = "0f0c8525-0fed-467f-888e-9bb45985eb78"
	}

	susd := fmt.Sprintf("%s?group_market_id=%s&market_id=%s",
		config.Server.OTCHost+getLatestPriceUrl, groupID, "SUSD/CNY")
	req, _ := http.NewRequest("GET", susd, nil)
	req.Header.Set("otc-session-id", "isecret")
	client := http.Client{Timeout: 10 * time.Second}
	rsp, err := client.Do(req)
	if err != nil {
		return
	}
	defer rsp.Body.Close()

	res := make(map[string]interface{}, 0)
	decoder := json.NewDecoder(rsp.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return
	}

	m, ok := res["result"].(map[string]interface{})
	if !ok {
		err = errors.New("result not a map")
		return
	}

	str, _ := m["latest_price"].(string)

	susdPrice, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return
	}

	if susdPrice == 0 {
		err = errors.New("susd price is zero")
		return
	}

	susdVolume := cnyVolume / susdPrice

	sie := fmt.Sprintf("%s?market=%s",
		config.Server.OTCHost+getMarketInfo, "siesusd")
	req, _ = http.NewRequest("GET", sie, nil)
	req.Header.Set("otc-session-id", "isecret")
	rsp, err = client.Do(req)
	if err != nil {
		return
	}
	defer rsp.Body.Close()

	res = make(map[string]interface{}, 0)
	decoder = json.NewDecoder(rsp.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return
	}

	m, ok = res["result"].(map[string]interface{})
	if !ok {
		err = errors.New("result is not a map")
		return
	}

	ticker, ok := m["ticker"].(map[string]interface{})
	if !ok {
		err = errors.New("ticker is not a map")
		return
	}

	str, _ = ticker["last"].(string)

	lastPrice, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return
	}

	sieVolume = susdVolume / lastPrice
	return
}

func susdToSie(susdVolume float64) (sieVolume float64, err error) {
	sie := fmt.Sprintf("%s?market=%s",
		config.Server.OTCHost+getMarketInfo, "siesusd")
	req, _ := http.NewRequest("GET", sie, nil)
	req.Header.Set("otc-session-id", "isecret")

	client := http.Client{Timeout: 10 * time.Second}
	rsp, err := client.Do(req)
	if err != nil {
		return
	}
	defer rsp.Body.Close()

	res := make(map[string]interface{}, 0)
	decoder := json.NewDecoder(rsp.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return
	}

	m, ok := res["result"].(map[string]interface{})
	if !ok {
		err = errors.New("result is not a map")
		return
	}

	ticker, ok := m["ticker"].(map[string]interface{})
	if !ok {
		err = errors.New("ticker is not a map")
		return
	}

	str, _ := ticker["last"].(string)

	lastPrice, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return
	}

	sieVolume = susdVolume / lastPrice
	return
}
