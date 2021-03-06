package sugar

import (
	"fmt"
	"net/http"
	"server-sugar-app/internal/app/group"
	"server-sugar-app/internal/app/warn"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
	"server-sugar-app/internal/dao"
	"server-sugar-app/internal/pkg/generr"
)

var (
	expectToken string

	calcSugar struct {
		sync.Mutex
		files          []string
		sourceFileName []string
	}
)

func ReceiveCalcFile(c *gin.Context) {
	var err error
	token := c.Param("token")
	if token == "" {
		err = errors.New("no token")
		log.Errorf("err: %+v", errors.Wrap(err, "parse 'token'"))
		c.JSON(http.StatusBadRequest, generr.SugarNoToken)
		return
	}

	// 验证临时token
	if token != expectToken {
		err = errors.Errorf("token: %s, receive token: %s", expectToken, token)
		log.Errorf("err: %+v", errors.Wrap(err, "wrong 'token'"))
		c.JSON(http.StatusBadRequest, generr.SugarWrongToken)
		return
	}

	askFilename := c.Param("filename")
	if askFilename == "" {
		err = errors.New("no filename")
		log.Errorf("err: %+v", errors.Wrap(err, "parse 'filename'"))
		c.JSON(http.StatusBadRequest, generr.SugarNoFile)
		return
	}

	// 检查是否是未知文件
	if !checkFile(askFilename) {
		err = errors.Errorf("received filename: %s", askFilename)
		log.Errorf("err: %+v", errors.Wrap(err, "wrong 'filename'"))
		c.JSON(http.StatusBadRequest, generr.SugarWrongFile)
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "parse file"))
		c.JSON(http.StatusBadRequest, generr.SugarFormFile)
		return
	}
	filename := fmt.Sprintf("%s_%s.zip", askFilename, time.Now().Format("20060102150405"))
	err = c.SaveUploadedFile(fh, curFilePath+filename)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "save file"))
		c.JSON(http.StatusBadRequest, generr.SugarSaveFile)
		return
	}

	calcSugar.Lock()
	// 检查是否重复上传
	for _, filename := range calcSugar.sourceFileName {
		if askFilename == filename {
			calcSugar.Unlock()
			err = errors.Errorf("filename: %s", askFilename)
			log.Errorf("err: %+v", errors.Wrap(err, "received same filename"))
			c.JSON(http.StatusBadRequest, generr.SugarRepeatFile)
			return
		}
	}

	calcSugar.files = append(calcSugar.files, filename)
	calcSugar.sourceFileName = append(calcSugar.sourceFileName, askFilename)
	calcSugar.Unlock()

	// 待所需上传文件皆以上传，开始计算 sugar 业绩
	sie := config.SIE
	if len(calcSugar.files) == len(sie.Sugars) {
		go func() {
			defer func() {
				calcSugar.files = []string{}
				calcSugar.sourceFileName = []string{}
			}()
			now := time.Now()

			err = warn.Must("calc sugar", CalcReward(now))
			if err != nil {
				return
			}
		}()
	}

	c.JSON(http.StatusOK, struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}{Code: 200, Msg: "success"})
	return
}

// 下载奖励文件
func DownloadRewardFile(c *gin.Context) {
	var err error
	filename := c.Param("filename")
	if filename == "" {
		err = errors.New("no param")
		log.Errorf("err: %+v", errors.Wrap(err, "parse 'filename'"))
		c.JSON(http.StatusBadRequest, generr.SugarNoToken)
		return
	}

	if curFilePath == "" {
		dirname := time.Now().Format("2006-01-02") + "/"
		curFilePath = "sugar/" + dirname
	}

	c.File(curFilePath + filename)
}

func GetUserRewardDetail(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	res, err := dao.RewardDetail.Get(userID)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get user reward detail"))
		c.JSON(http.StatusBadRequest, generr.ServerError)
		return
	}

	c.JSON(http.StatusOK, struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}{200, "success", res})
}

func Prepare(c *gin.Context) {
	prepare := c.Param("prepare")
	switch prepare {
	case "pledge":
		if err := persistDEFIPledgeSIE(); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	case "lock_sie":
		if err := persistLockSIE(); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	case "account_in":
		sie := config.SIE
		accInMap, _, err := getAccountsBalanceInOrOut(sie.SIEAddAccounts, 1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, fmt.Errorf("get account balance in failed: %v", err))
			return
		}
		dirname := time.Now().Format("2006-01-02") + "/"
		curFilePath = "sugar/" + dirname
		writeFile(accInMap, 1)
	case "account_out":
		sie := config.SIE
		accOutMap, _, err := getAccountsBalanceInOrOut(sie.SIESubAccounts, 2)
		if err != nil {
			c.JSON(http.StatusInternalServerError, fmt.Errorf("get account balance out failed: %v", err))
			return
		}
		dirname := time.Now().Format("2006-01-02") + "/"
		curFilePath = "sugar/" + dirname
		writeFile(accOutMap, 2)
	case "relation": // update relation
		go group.GetLatestGroupRela()
	default:
		c.JSON(http.StatusBadRequest, "unknown prepare")
		return
	}
	c.JSON(http.StatusOK, "ok")
}

func ManualStart(c *gin.Context) {
	req := struct {
		Filenames   []string `form:"filenames" binding:"required"`
		CurFilePath string   `form:"cur_file_path" binding:"required"`
	}{}

	err := c.ShouldBind(&req)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "should bind"))
		c.JSON(http.StatusBadRequest, generr.ParseParam)
		return
	}

	curFilePath = req.CurFilePath

	go group.GetLatestGroupRela()

	go func() {
		err = CalcReward(time.Now())
		if err != nil {
			log.Errorf("err: %+v", errors.Wrap(err, "calc sugar reward"))
			return
		}
	}()

	c.JSON(http.StatusOK, struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}{Code: 200, Msg: "success"})
	return
}

func checkFile(filename string) bool {
	sie := config.SIE
	for _, v := range sie.Sugars {
		if filename == v.Origin {
			return true
		}
	}
	return false
}
