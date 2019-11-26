package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"yangsi/baiduocr"
	"yangsi/cfg"
	"yangsi/db"
	"yangsi/img"
	"yangsi/log"
)

func main() {
	process(root)
	deinit()
	log.InfoLog("处理成功：%d 张, 处理失败：%d 张", okNum, failedNum)
	time.Sleep(time.Hour * 24)
}

var (
	okNum     int32
	failedNum int32
)

func addOK() {
	atomic.AddInt32(&okNum, 1)
}
func addFailed() {
	atomic.AddInt32(&failedNum, 1)
}

var root string

func init() {
	conf, err := cfg.Init()
	if err != nil {
		log.ErrorLog("cfg init failed: %s", err.Error())
		os.Exit(1)
	}
	root = conf.Root
	err = baiduocr.Init(conf.OCR)
	if err != nil {
		log.ErrorLog("orc init failed: %s", err.Error())
		os.Exit(1)
	}
	err = db.Init(conf.DB)
	if err != nil {
		log.ErrorLog("db init failed: %s", err.Error())
		os.Exit(1)
	}
	err = img.Init(conf.IMG)
	if err != nil {
		log.ErrorLog("img init failed: %s", err.Error())
		os.Exit(1)
	}
}

var (
	wait    sync.WaitGroup
	closeCh = make(chan struct{})
)

func stop() {
	defer func() {
		if e := recover(); e != nil {
			log.ErrorLog("非法退出，处理失败: %v", e)
		}
	}()
	close(closeCh)
}

func deinit() {
	stop()
	wait.Wait()
}

func process(root string) {
	var fileCh = make(chan *img.Image, 5)
	wait.Add(1)
	go func() {
		err := walk(root, &walkOption{true}, fileCh)
		if err != nil {
			log.ErrorLog(err.Error())
		}
		close(fileCh)
		wait.Done()
	}()
	var queue = make(chan struct{}, 5)
	for file := range fileCh {
		time.Sleep(time.Millisecond * 100) // 5 次 / s

		wait.Add(1)
		queue <- struct{}{}
		go func(img *img.Image) {
			handleImage(img)
			<-queue
			wait.Done()
		}(file)
	}
	close(queue)
}

type walkOption struct {
	Recursion bool
}

func transformPath(path string) string {
	return strings.Replace(path, "\\", "/", -1)
}

func walk(dir string, o *walkOption, file chan *img.Image) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	var filename string
	for _, f := range files {
		filename = fmt.Sprintf("%s/%s", dir, f.Name())
		if f.IsDir() && o.Recursion {
			err = walk(filename, o, file)
			if err != nil {
				return err
			}
			continue
		}
		image, err := img.NewImage(dir, f.Name())
		if err != nil {
			log.WarnLog("invalid image file: %s", err)
			addFailed()
			continue
		}
		select {
		case file <- image:
		case <-closeCh:
			return nil
		}
	}
	return nil
}

///////////////////////////////
func handleImage(img *img.Image) {
	var err error
	defer func() {
		if err != nil {
			log.WriteError(err, "%s failed", img.Path())
			addFailed()
		} else {
			log.RealtimeLog("%s ok", img.Path())
			addOK()
		}
	}()
	err = img.Load()
	if err != nil {
		return
	}
	imgData, err := img.Smaller()
	if err != nil {
		return
	}
	orcData, err := baiduocr.OCR(imgData)
	if err != nil {
		if _, ok := err.(baiduocr.ErrShouldExit); ok {
			stop()
		}
		return
	}
	// log.WarnLog("ocr data: %s", orcData)
	dbh := db.Get()
	tx, err := dbh.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	path, err := img.Store()
	if err != nil {
		return
	}
	err = db.Insert(tx, img.ModTime, path, orcData)
	if err != nil {
		return
	}
	err = os.Remove(img.Path())
	if err != nil {
		return
	}
	err = tx.Commit()
	if err != nil {
		return
	}
}
