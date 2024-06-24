package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

const Dsn = "root:mysql#123456@(10.1.1.100:3306)/get_magnet_db"

var (
	err error
	Db  *sql.DB
)

func init() {
	Db, err = sql.Open("mysql", Dsn)
	if err != nil {
		log.Fatalln(err)
	}

	err = Db.Ping()
	if err != nil {
		log.Fatalln(err)
	}
}
