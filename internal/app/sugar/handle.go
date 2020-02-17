package sugar

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/config"
	"server-sugar-app/internal/pkg/generr"
)

var (
	ExpectToken string

	calcSugar struct {
		sync.Mutex
		files []string
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
	if token != ExpectToken {
		err = errors.Errorf("token: %s, receive token: %s", ExpectToken, token)
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

	// 检查是否重复上传
	for _, filename := range calcSugar.files {
		if askFilename == filename {
			err = errors.Errorf("filename: %s", askFilename)
			log.Errorf("err: %+v", errors.Wrap(err, "received same filename"))
			c.JSON(http.StatusBadRequest, generr.SugarRepeatFile)
			return
		}
	}

	fh, err := c.FormFile("file")
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "parse file"))
		c.JSON(http.StatusBadRequest, generr.SugarFormFile)
		return
	}
	filename := fmt.Sprintf("%s_%s.zip", askFilename, time.Now().Format("20060102150405"))
	err = c.SaveUploadedFile(fh, FilePath+filename)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "save file"))
		c.JSON(http.StatusBadRequest, generr.SugarSaveFile)
		return
	}

	calcSugar.Lock()
	calcSugar.files = append(calcSugar.files, filename)
	calcSugar.Unlock()

	// 待所需上传文件皆以上传，开始计算 sugar 业绩
	sie := config.SIE
	if len(calcSugar.files) == len(sie.Sugars) {
		go func() {
			defer func() {
				calcSugar.files = []string{}
			}()
			err = calcReward(calcSugar.files)
			if err != nil {
				log.Errorf("err: %+v", errors.Wrap(err, "calc sugar reward"))
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

func checkFile(filename string) bool {
	sie := config.SIE
	for _, v := range sie.Sugars {
		if filename == v.Origin {
			return true
		}
	}
	return false
}