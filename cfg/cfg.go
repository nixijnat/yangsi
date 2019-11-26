package cfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"yangsi/log"
)

const (
	defaultOCRConfig = `{
	"token_file": "./token",
	"token_auth":{
		"grant_type": "client_credentials",
		"client_id": "",
		"client_secret":""
	}
}`
	defaultDBConfig = `{
	"db_name": "./default.db",
	"tb_name": "yangsi"
}`
	defaultIMGConfig = `{
	"out_dir": "%s/out",
	"out_img": {
		"max_pixel": 1920,
		"quality": 75
	}
}`
	defaultRootDir = "./origin"
)

type global struct {
	Root string          `json:"root"`
	OCR  json.RawMessage `json:"ocr"`
	DB   json.RawMessage `json:"db"`
	IMG  json.RawMessage `json:"img"`
}

const path = "./conf.json"

func Init() (*global, error) {
	var conf = new(global)
	file, err := ioutil.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, log.NewError("read conf.json failed: %s", err.Error())
		}
		conf, err = generate()
		if err != nil {
			return nil, err
		}
	} else {
		err = json.Unmarshal(file, conf)
		if err != nil {
			return nil, log.NewError("unmarshal conf failed: %s, %s", string(file), err.Error())
		}
		if conf.Root == "" || len(conf.DB) == 0 || len(conf.IMG) == 0 ||
			len(conf.OCR) == 0 {
			return nil, log.NewError("invalid conf: %s, %s", string(file), err.Error())
		}
	}
	_, err = os.Stat(conf.Root)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, log.NewError("root dir failed: %s, %s", conf.Root, err.Error())
		}
		err = os.MkdirAll(conf.Root, os.ModePerm)
		if err != nil {
			return nil, log.NewError("mkdir failed: %s", conf.Root)
		}
	}
	return conf, nil
}

func imgConf() (json.RawMessage, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, log.NewError("cannot stat current dir: %s", err.Error())
	}
	dir = strings.ReplaceAll(dir, "\\", "/")
	log.RealtimeLog("cur dir: %s", dir)
	return json.RawMessage(fmt.Sprintf(defaultIMGConfig, dir)), nil
}

func generate() (*global, error) {
	var conf = &global{
		Root: defaultRootDir,
		DB:   json.RawMessage(defaultDBConfig),
		OCR:  json.RawMessage(defaultOCRConfig),
	}
	var err error
	conf.IMG, err = imgConf()
	if err != nil {
		return nil, err
	}

	confStr, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		return nil, log.NewError("marshal conf failed: %s", err.Error())
	}
	err = ioutil.WriteFile(path, confStr, os.ModePerm)
	if err != nil {
		return nil, log.NewError("write conf file failed: %s", err.Error())
	}
	return conf, nil
}
