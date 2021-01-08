package util

import (
	log "github.com/sirupsen/logrus"
	"time"
)

var ShLoc *time.Location

func init() {
	var err error
	ShLoc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Warnf("load location failed")
		ShLoc = time.Local
	}
}
