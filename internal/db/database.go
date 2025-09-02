package db

import (
	"context"
	_ "github.com/lib/pq"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/core"
	"github.com/nekoimi/get-magnet/internal/db/migrate"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	log "github.com/sirupsen/logrus"
	"sync"
	"xorm.io/xorm"
)

var (
	err        error
	engine     *xorm.Engine
	engineOnce sync.Once
)

func NewDBLifecycle() core.Lifecycle {
	return core.NewLifecycle("DB", func(ctx context.Context) error {
		cfg := core.PtrFromContext[config.Config](ctx)
		// 初始化数据库
		initialize(cfg.DB)
		return nil
	}, func(ctx context.Context) error {
		return engine.Close()
	})
}

// 初始化数据库操作
func initialize(cfg *Config) {
	engineOnce.Do(func() {
		log.Debugf("连接数据库")
		engine, err = xorm.NewEngine(Postgres.String(), cfg.Dsn)
		if err != nil {
			log.Errorf("连接数据库异常: %s", err.Error())
			panic(err)
		}

		// 初始化设置
		engine.ShowSQL(true)
		engine.SetLogger(newXormLogger())
		// 连接池设置
		engine.SetMaxIdleConns(8)
		engine.SetMaxOpenConns(8)

		err = engine.Ping()
		if err != nil {
			log.Errorf("数据库连接不可用: %s", err.Error())
			panic(err)
		}

		// 初始化数据迁移
		initMigrates(engine)
		// 数据表迁移
		runMigrates(engine)
	})
}

// Instance 获取数据库操作实例
func Instance() *xorm.Engine {
	return engine
}

// 初始化数据迁移
func initMigrates(e *xorm.Engine) {
	mg := new(table.Migrates)
	if exist, err := e.IsTableExist(mg); err != nil {
		log.Errorf("数据表检查失败: %s", err.Error())
		panic(err)
	} else if !exist {
		err := e.CreateTables(mg)
		if err != nil {
			log.Errorf("数据表初始化失败: %s", err.Error())
			panic(err)
		}
	}
}

// 初始化数据表迁移
func runMigrates(e *xorm.Engine) {
	migrates := migrate.GetAll()
	util.Sort[migrate.Migrate](migrates, func(a migrate.Migrate, b migrate.Migrate) bool {
		return a.Version() < b.Version()
	})
	log.Debugln("数据表迁移执行...")
	for _, m := range migrates {
		if exists, err := e.Exist(&table.Migrates{Version: m.Version()}); err != nil {
			log.Errorf("数据表迁移异常: %s, \n details: %s", m.Desc(), err.Error())
			break
		} else if exists {
			// 已经存在迁移记录，直接跳过所有
			break
		}

		log.Infof("数据表迁移: %d, %s , 执行...", m.Version(), m.Desc())
		err = m.Exec(e)
		if err != nil {
			log.Errorf("数据表迁移异常: %s, \n details: %s", m.Desc(), err.Error())
			if _, insertErr := e.InsertOne(&table.Migrates{
				Version: m.Version(),
				Success: false,
				Message: err.Error(),
			}); insertErr != nil {
				log.Errorf("数据库操作失败: %s", insertErr.Error())
			}
			break
		}
		if _, insertErr := e.InsertOne(&table.Migrates{
			Version: m.Version(),
			Success: true,
			Message: "ok",
		}); insertErr != nil {
			log.Errorf("数据库操作失败: %s", insertErr.Error())
		}
	}
	log.Infoln("数据表迁移执行完毕")
}
