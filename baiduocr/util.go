package baiduocr

import (
	"encoding/base64"
	"encoding/json"
	"yangsi/log"
)

const (
	fourMegabyte = 1024 * 1024 * 4
)

func encodeImg(origin []byte) ([]byte, error) {
	imgSize := base64.StdEncoding.EncodedLen(len(origin))
	if imgSize >= fourMegabyte {
		return nil, log.NewWarn("image size larger than 4MB")
	}
	imgBuf := make([]byte, imgSize)
	base64.StdEncoding.Encode(imgBuf, origin)
	return imgBuf, nil
}

///
type config struct {
	TokenAuth struct {
		GrantType    string `json:"grant_type"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"token_auth"`
	TokenFile string `json:"token_file"`
}

func (c *config) check() error {
	if c.TokenFile == "" || c.TokenAuth.GrantType == "" || c.TokenAuth.ClientID == "" ||
		c.TokenAuth.ClientSecret == "" {
		return log.NewWarn("invalid img config: %+v", *c)
	}
	return nil
}

var localConf config

func Init(cfg json.RawMessage) error {
	err := json.Unmarshal(cfg, &localConf)
	if err != nil {
		return log.NewError("init config failed: %s", err.Error())
	}
	err = localConf.check()
	if err != nil {
		return err
	}
	tmpToken, err := getToken()
	if err != nil {
		return err
	}
	setTokenVar(tmpToken)
	log.RealtimeLog("token is: %s", tmpToken)
	return nil
}
