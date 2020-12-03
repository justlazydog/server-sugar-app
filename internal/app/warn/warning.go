package warn

import log "github.com/sirupsen/logrus"

func Must(desc string, err error) error {
	if err != nil {
		log.Errorf("%s failed: %v", desc, err)
		// TODO 发生报警
	}
	return err
}
