package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

var (
	err error
	db  *sql.DB
)

func Init(dsn string) {
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalln(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalln(err)
	}
}

func Get() *sql.DB {
	return db
}
