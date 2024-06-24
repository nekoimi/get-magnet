package db_storage

import (
	"database/sql"
	"get-magnet/pkg/db"
	"get-magnet/pkg/util"
	"get-magnet/scheduler"
	"get-magnet/storage"
	"log"
)

const InsertSql = "INSERT INTO magnets (created_at, updated_at, title, number, optimal_link, links, res_host, res_path, status) VALUE (CURRENT_TIMESTAMP(),CURRENT_TIMESTAMP(),?,?,?,?,?,?,?)"

type dbStorage struct {
	db *sql.DB
}

func New() storage.Storage {
	return &dbStorage{
		db: db.Db,
	}
}

func (ds *dbStorage) Save(item scheduler.MagnetItem) error {
	stmt, err := ds.db.Prepare(InsertSql)
	if err != nil {
		log.Printf("sql err: %s \n", err.Error())
		return err
	}
	_, err = stmt.Exec(
		item.Title,
		item.Number,
		item.OptimalLink,
		util.ToJson(item.Links),
		item.ResHost,
		item.ResPath,
		0,
	)
	if err != nil {
		log.Printf("sql exec err: %s \n", err.Error())
		return err
	}
	return nil
}
