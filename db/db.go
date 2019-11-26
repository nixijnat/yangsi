package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"yangsi/log"

	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	DBName string `json:"db_name"`
	TBName string `json:"tb_name"`
}

func (c *config) check() error {
	if c.DBName == "" || c.TBName == "" {
		return log.NewError("invalid db config: %+v", *c)
	}
	return nil
}

var (
	localConf config
)

const (
	ctbTpl = "CREATE TABLE IF NOT EXISTS `%s` (`id` INTEGER PRIMARY KEY,`time` DATETIME NOT NULL,`path` VARCHAR(512) NOT NULL DEFAULT '',`text` TEXT)"
	insTpl = "INSERT INTO `%s`(`time`,`path`,`text`) VALUES(?,?,?)"
	qryTpl = "SELECT `id`,`time`,`path`,`text` FROM `%s` WHERE `text` LIKE ?"
)

var (
	createSentence string
	insertSentence string
	querySentence  string
)

func Init(cfgStr json.RawMessage) error {
	err := json.Unmarshal(cfgStr, &localConf)
	if err != nil {
		return log.NewError("unmarshal db config failed: %s, %s", string(cfgStr), err.Error())
	}
	err = localConf.check()
	if err != nil {
		return err
	}
	db, err = sql.Open("sqlite3", localConf.DBName)
	if err != nil {
		return log.NewError("open sqlite failed: %s", localConf.DBName)
	}
	createSentence = fmt.Sprintf(ctbTpl, localConf.TBName)
	insertSentence = fmt.Sprintf(insTpl, localConf.TBName)
	querySentence = fmt.Sprintf(qryTpl, localConf.TBName)
	_, err = db.Exec(createSentence)
	if err != nil {
		return log.NewError("create table failed: %s", err.Error())
	}
	return nil
}

var (
	db *sql.DB
)

func Get() *sql.DB {
	return db
}

func insert(tx *sql.Tx, modtime, path, text string) error {
	_, err := tx.Exec(insertSentence, modtime, path, text)
	if err != nil {
		return log.NewError("insert failed: %s", err.Error())
	}
	return nil
}

func query(str string) ([][3]string, error) {
	rows, err := db.Query(querySentence, str)
	if err != nil {
		return nil, log.NewError("query failed: %s", err.Error())
	}
	defer rows.Close()
	var result = make([][3]string, 16)
	var tmp [3]string
	for rows.Next() {
		err = rows.Scan(&tmp[0], &tmp[1], &tmp[2])
		if err != nil {
			return nil, log.NewError("scan rows failed: %s", err.Error())
		}
		result = append(result, tmp)
	}
	return result, nil
}

func Query(str string) ([][3]string, error) {
	str = strings.TrimSpace(str)
	if str == "" {
		return nil, nil
	}
	return query(str)
}

///////////////////////////
func Insert(tx *sql.Tx, modtime, path, text string) error {
	return insert(tx, modtime, path, text)
}
