package wechat

import (
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
)

// implements UUIDProcessor
type defaultUUIDProcessor struct {
	path string
}

func (dp *defaultUUIDProcessor) ProcessUUID(uuid string) error {
	// 2.``
	path, err := fetchORCodeImage(uuid)

	if err != nil {
		return err
	}
	logrus.Infof(`二维码图片地址: %s`, path)

	// 3.
	go func() {
		dp.path = path
		open.Start(path)
	}()
	logrus.Info(`请用手机微信扫描二维码...`)

	return nil
}

func (dp *defaultUUIDProcessor) UUIDDidConfirm(err error) {
	if len(dp.path) > 0 {
		deleteFile(dp.path)
	}
}
