package migrate

import (
	"github.com/nekoimi/get-magnet/internal/db/table"
	"xorm.io/xorm"
)

type initMagnets struct {
}

func init() {
	registerMigrate(new(initMagnets))
}

func (i *initMagnets) Version() int64 {
	return 2025_06_21_001
}

func (i *initMagnets) Desc() string {
	return "初始化磁力信息数据表"
}

func (i *initMagnets) Exec(e *xorm.Engine) error {
	err := AutoCreateTable(e, new(table.Magnets))
	if err != nil {
		return err
	}
	return nil
}
