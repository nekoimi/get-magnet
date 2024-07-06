package database

import (
	"github.com/nekoimi/get-magnet/config"
	"log"
	"xorm.io/xorm"
)

var (
	err    error
	engine *xorm.Engine
)

// Init 初始化数据库操作
func Init(cfg config.Database) {
	engine, err = xorm.NewEngine(Postgres.String(), cfg.Dns)
	if err != nil {
		log.Fatalf("连接数据库异常: %s\n", err.Error())
	}

	err = engine.Ping()
	if err != nil {
		log.Fatalf("数据库连接不可用: %s\n", err.Error())
	}

	// 初始化设置
	engine.ShowSQL(cfg.ShowSQL)
	engine.Logger().SetLevel(cfg.LogLevel)
	// 连接池设置
	engine.SetMaxIdleConns(cfg.MaxIdleConnNum)
	engine.SetMaxOpenConns(cfg.MaxOpenConnNum)

	// 初始化数据表
	initTables(engine)
}

// Get 获取数据库操作实例
func Get() *xorm.Engine {
	return engine
}

// 初始化数据表
func initTables(e *xorm.Engine) {
	// autoCreateTable(e, nil)
}

func autoCreateTable(e *xorm.Engine, tableBean any) {
	if exists, err := e.IsTableExist(tableBean); err != nil {
		log.Fatalf("检查数据表状态异常: %s\n", err.Error())
	} else if !exists {
		if err := e.CreateTables(tableBean); err != nil {
			log.Fatalf("创建数据表异常: %s\n", err.Error())
		}
	}
}
