package migrate

import (
	"log"
	"xorm.io/xorm"
)

type Migrate interface {
	Version() int64
	Desc() string
	Exec(e *xorm.Engine) error
}

var (
	migrates = make([]Migrate, 0)
)

// registerMigrate 注册数据表迁移
func registerMigrate(m Migrate) {
	migrates = append(migrates, m)
}

// AutoCreateTable 自动创建数据表
func AutoCreateTable(e *xorm.Engine, tableBean any) error {
	if exists, err := e.IsTableExist(tableBean); err != nil {
		log.Fatalf("检查数据表状态异常: %s\n", err.Error())
		return err
	} else if !exists {
		if err := e.CreateTables(tableBean); err != nil {
			log.Fatalf("创建数据表异常: %s\n", err.Error())
			return err
		}
	}
	return nil
}

// GetAll 获取全部注册的迁移脚本
func GetAll() []Migrate {
	return migrates
}
