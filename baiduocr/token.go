package baiduocr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
	"yangsi/log"
)

type (
	// {		"error": "invalid_client",		"error_description": "unknown client id"	}
	tokenAPIErrType struct {
		Err     string `json:"error"`
		ErrDesc string `json:"error_description"`
	}
	cloudTokenType struct {
		tokenAPIErrType
		AccessToken      string `json:"access_token"`
		ExpiresIn        int64  `json:"expires_in"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	cacheTokenType struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
)

func (e tokenAPIErrType) Error() string {
	return fmt.Sprintf("baidu token api: %s, %s", e.Err, e.ErrDesc)
}
func (e tokenAPIErrType) shift() error {
	if e.Err == "" {
		return nil
	}
	return e
}

var (
	accessToken string
	tokenMutex  sync.RWMutex
)

func getTokenVar() string {
	tokenMutex.RLock()
	defer tokenMutex.RUnlock()
	return accessToken
}

func setTokenVar(v string) {
	tokenMutex.Lock()
	accessToken = v
	tokenMutex.Unlock()
}

func getToken() (string, error) {
	token, err := cacheToken()
	if err == nil && token != "" {
		return token, nil
	}

	token, err = cloudToken()
	if err != nil {
		return "", err
	}
	return token, nil
}

func storeToken(token *cloudTokenType) error {
	data, err := json.Marshal(cacheTokenType{
		AccessToken: token.AccessToken,
		ExpiresIn:   (time.Now().Add((time.Duration)(token.ExpiresIn) * time.Second)).Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		return log.NewWarn("marshal token failed: %+v", token)
	}
	err = ioutil.WriteFile(localConf.TokenFile, data, os.ModePerm)
	if err != nil {
		return log.NewWarn("write token failed: %s", err.Error())
	}
	return nil
}

const tokenUrl = "https://aip.baidubce.com/oauth/2.0/token?grant_type=%s&client_id=%s&client_secret=%s"

func cloudToken() (string, error) {
	log.RealtimeLog("get token from cloud Token From Internet..")
	url := fmt.Sprintf(tokenUrl, localConf.TokenAuth.GrantType,
		localConf.TokenAuth.ClientID, localConf.TokenAuth.ClientSecret)

	var respData = new(cloudTokenType)
	err := get(url, respData)
	if err != nil {
		return "", err
	}
	if respData.Error != "" || respData.ErrorDescription != "" {
		return "", log.NewWarn("get token failed: %s, %s", respData.Error, respData.ErrorDescription)
	}
	err = storeToken(respData)
	if err != nil {
		log.WarnLog("store token failed: %s", err.Error())
	}
	return respData.AccessToken, nil
}

func cacheToken() (string, error) {
	log.RealtimeLog("get token from cloud Token From local ..")
	data, err := ioutil.ReadFile(localConf.TokenFile)
	if err != nil {
		return "", err
	}
	var tmp = new(cacheTokenType)
	err = json.Unmarshal(data, tmp)
	if err != nil {
		return "", log.NewWarn("invalid token cache file: %s", string(data))
	}
	if tmp.AccessToken == "" || tmp.ExpiresIn == "" {
		return "", errors.New("invalid token cache")
	}
	deadline, err := time.ParseInLocation("2006-01-02 15:04:05", tmp.ExpiresIn, time.Local)
	if !time.Now().Before(deadline) {
		return "", log.NewWarn("token cache is expired: %s", tmp.ExpiresIn)
	}
	return tmp.AccessToken, nil
}
