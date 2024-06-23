package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

const Dsn = "root:mysql#123456@(10.1.1.100:3306)/get_magnet_db"

var db *sql.DB

func init() {
	db, err := sql.Open("mysql", Dsn)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalln(err)
	}
}

// GetDb get a sql database instance
func GetDb() *sql.DB {
	return db
}
