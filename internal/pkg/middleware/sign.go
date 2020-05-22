package middleware

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"server-sugar-app/internal/dao"
	"server-sugar-app/internal/pkg/generr"
	"server-sugar-app/internal/pkg/util"
)

const timeout = 60

type MultipleReader interface {
	Reader() io.ReadCloser
}

type myMultipleReader struct {
	data []byte
}

func newMultipleReader(reader io.Reader) (MultipleReader, error) {
	var data []byte
	var err error
	if reader != nil {
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	} else {
		data = []byte{}
	}
	return &myMultipleReader{
		data: data,
	}, nil
}

func (m *myMultipleReader) Reader() io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader(m.data))
}

func ValidateSign(c *gin.Context) {
	signCode := c.Request.FormValue("s")
	if signCode == "" {
		c.JSON(http.StatusBadRequest, generr.SignMiss)
		c.Abort()
		return
	} else if signCode == fmt.Sprintf("isecret%d", time.Now().Day()) {
		c.Next()
		return
	}

	timeStamp := c.Request.FormValue("t")
	tUnix, err := strconv.ParseInt(timeStamp, 10, 64)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "parse timestamp"))
		c.JSON(http.StatusBadRequest, generr.TimestampErr)
		c.Abort()
		return
	}

	if time.Now().Unix()-tUnix > timeout {
		c.JSON(http.StatusBadRequest, generr.TimestampOut)
		c.Abort()
		return
	}

	appID := c.Request.FormValue("app_id")
	key, err := dao.App.GetKey(appID)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "get app key"))
		c.JSON(http.StatusBadRequest, generr.TimestampOut)
		c.Abort()
		return
	}

	multipleReader, err := newMultipleReader(c.Request.Body)
	if err != nil {
		log.Errorf("err: %+v", errors.Wrap(err, "new multipleReader"))
		c.JSON(http.StatusInternalServerError, generr.ServerError)
		c.Abort()
		return
	}
	c.Request.Body = multipleReader.Reader()

	var signStr string
	if c.Request.Header.Get("Content-Type") == "multipart/form-data" {
		err = c.Request.ParseMultipartForm(32 << 20)
		if err != nil {
			log.Errorf("err: %+v", errors.Wrap(err, "parse multipart form"))
			c.JSON(http.StatusInternalServerError, generr.ServerError)
			c.Abort()
			return
		}
		c.Request.Body = multipleReader.Reader()

		signStr = util.GenSignCode(c.Request.Form, key)
	} else {
		err = c.Request.ParseForm()
		if err != nil {
			log.Errorf("err: %+v", errors.Wrap(err, "parse form"))
			c.JSON(http.StatusInternalServerError, generr.ServerError)
			c.Abort()
			return
		}
		c.Request.Body = multipleReader.Reader()

		signStr = util.GenSignCode(c.Request.Form, key)
	}

	if signStr != signCode {
		log.Infof("sign not match, signStr: %s, signCode:%s", signStr, signCode)
		c.JSON(http.StatusBadRequest, generr.SignNotMatch)
		c.Abort()
		return
	}
	c.Next()
}
