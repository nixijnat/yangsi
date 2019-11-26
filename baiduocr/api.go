package baiduocr

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"yangsi/log"
)

var (
	client = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
)

func get(url string, out apiError) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	return readResponse(resp, out)
}

func post(url string, reqData []byte, out apiError) error {
	url = fmt.Sprintf("%s?access_token=%s", url, getTokenVar())
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqData))
	if err != nil {
		return log.NewError("new request failed: %s, %s", url, err.Error())
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	err = readResponse(resp, out)
	if err != nil {
		log.WarnLog("read resp failed: %s, %v", url, err)
		return err
	}
	return nil
}

var errFlag = new(int32)

type ErrShouldExit string

func (e ErrShouldExit) Error() string {
	return string(e)
}

var shouldExit = "token err or no free ocr, should exit"

func readResponse(resp *http.Response, out apiError) error {
	if resp.StatusCode != http.StatusOK {
		return log.NewWarn("get failed: %d", resp.StatusCode)
	}
	respData, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	err = json.Unmarshal(respData, out)
	if err != nil {
		return err
	}
	err = out.shift()
	if err != nil {
		if atomic.AddInt32(errFlag, 1) >= 10 {
			log.ErrorLog("request err: %s", err.Error())
			return ErrShouldExit(shouldExit)
		}
		_, ok := err.(tokenError)
		if ok {
			tmpToken, err := cloudToken()
			if err != nil {
				return err
			}
			setTokenVar(tmpToken)
		}
		return err
	}
	// log.RealtimeLog("resp is: %s, %v", string(respData), out)
	atomic.StoreInt32(errFlag, 0)
	return nil
}

type (
	apiErrType struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}
	tokenError apiErrType
)

func (e apiErrType) Error() string {
	return fmt.Sprintf("baidu api: %d, %s", e.ErrorCode, e.ErrorMsg)
}

type apiError interface {
	shift() error
}

func (e tokenError) Error() string {
	return fmt.Sprintf("baidu token: %d, %s", e.ErrorCode, e.ErrorMsg)
}

/*
100	Invalid parameter	无效的access_token参数，请检查后重新尝试
110	Access token invalid or no longer valid	access_token无效
111	Access token expired	access token过期
*/
func (e apiErrType) shift() error {
	switch e.ErrorCode {
	case 0:
		return nil
	case 100, 110, 111:
		return tokenError(e)
	}
	return e
}

type orcResult struct {
	apiErrType
	Direction   int   `json:"direction"`
	LogID       int64 `json:"log_id"`
	WordsResult []struct {
		Words string `json:"words"`
	} `json:"words_result"`
	WordsResultNums int `json:"words_result_nums"`
}

func (o *orcResult) String() string {
	var buf bytes.Buffer
	var err error
	var tmp string
	for i := range o.WordsResult {
		tmp = strings.TrimSpace(o.WordsResult[i].Words)
		if tmp == "" {
			continue
		}
		if i != 0 {
			err = buf.WriteByte('\n')
			if err != nil {
				log.ErrorLog("buf error: %s", err)
				return ""
			}
		}
		_, err = buf.WriteString(tmp)
		if err != nil {
			log.ErrorLog("buf error: %s", err)
			return ""
		}
	}
	return buf.String()
}

func OCR(imgData []byte) (string, error) {
	// encode
	enc, err := encodeImg(imgData)
	if err != nil {
		return "", err
	}
	reqParams := url.Values{}
	reqParams.Set("language_type", "CHN_ENG")
	reqParams.Set("detect_direction", "true")
	reqParams.Set("image", string(enc))
	var respData orcResult
	const baseUrl = "https://aip.baidubce.com/rest/2.0/ocr/v1/general_basic"
	err = post(baseUrl, []byte(reqParams.Encode()), &respData)
	if err != nil {
		return "", err
	}
	// log.RealtimeLog("result: %s", respData)
	result := respData.String()
	if result == "" {
		return "", log.NewWarn("nothing recognized")
	}
	return result, nil
}
