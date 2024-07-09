package database

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/nekoimi/get-magnet/config"
	"github.com/nekoimi/get-magnet/database/migrate"
	"github.com/nekoimi/get-magnet/database/table"
	"github.com/nekoimi/get-magnet/pkg/util"
	"log"
	"xorm.io/xorm"
)

var (
	err    error
	engine *xorm.Engine
)

// Init 初始化数据库操作
func Init(cfg *config.Database) {
	log.Printf("连接数据库")
	engine, err = xorm.NewEngine(MySQL.String(), cfg.Dns)
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

	// 初始化数据迁移
	initMigrates(engine)
	// 数据表迁移
	runMigrates(engine)
}

// Instance 获取数据库操作实例
func Instance() *xorm.Engine {
	return engine
}

// 初始化数据迁移
func initMigrates(e *xorm.Engine) {
	mg := new(table.Migrates)
	if exist, err := e.IsTableExist(mg); err != nil {
		log.Fatalf("数据表检查失败: %s\n", err.Error())
	} else if !exist {
		err := e.CreateTables(mg)
		if err != nil {
			log.Fatalf("数据表初始化失败: %s\n", err.Error())
		}
	}
}

// 初始化数据表迁移
func runMigrates(e *xorm.Engine) {
	migrates := migrate.GetAll()
	util.Sort[migrate.Migrate](migrates, func(a *migrate.Migrate, b *migrate.Migrate) bool {
		return (*a).Version() < (*b).Version()
	})
	log.Println("数据表迁移执行...")
	for _, m := range migrates {
		if exists, err := e.Exist(&table.Migrates{Version: m.Version()}); err != nil {
			log.Printf("数据表迁移异常: %s, \n details: %s \n", m.Desc(), err.Error())
			break
		} else if exists {
			// 已经存在迁移记录，直接跳过所有
			break
		}

		log.Printf("数据表迁移: %d, %s , 执行...\n\n", m.Version(), m.Desc())
		err = m.Exec(e)
		if err != nil {
			log.Printf("数据表迁移异常: %s, \n details: %s \n", m.Desc(), err.Error())
			if _, insertErr := e.InsertOne(&table.Migrates{
				Version: m.Version(),
				Success: false,
				Message: err.Error(),
			}); insertErr != nil {
				log.Fatalf("数据库操作失败: %s\n", insertErr.Error())
			}
			break
		}
		if _, insertErr := e.InsertOne(&table.Migrates{
			Version: m.Version(),
			Success: true,
			Message: "ok",
		}); insertErr != nil {
			log.Fatalf("数据库操作失败: %s\n", insertErr.Error())
		}
	}
	log.Println("数据表迁移执行完毕")
}
