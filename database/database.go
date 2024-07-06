package database

import (
	"log"
	"xorm.io/xorm"
	xormLog "xorm.io/xorm/log"
)

var (
	err    error
	engine *xorm.Engine
)

// Init 初始化数据库操作
func Init(dataSource string) {
	engine, err = xorm.NewEngine(Postgres.String(), dataSource)
	if err != nil {
		log.Fatalf("连接数据库异常: %s\n", err.Error())
	}

	err = engine.Ping()
	if err != nil {
		log.Fatalf("数据库连接不可用: %s\n", err.Error())
	}

	// 初始化设置
	engine.ShowSQL(true)
	engine.Logger().SetLevel(xormLog.LOG_DEBUG)
	// 连接池设置
	engine.SetConnMaxIdleTime(8)
	engine.SetMaxOpenConns(8)

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
