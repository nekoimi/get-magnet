package migrate

import (
	"github.com/nekoimi/get-magnet/database/table"
	"github.com/nekoimi/get-magnet/pkg/util"
	"xorm.io/xorm"
)

const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "admin"
)

type initTable struct {
}

func init() {
	registerMigrate(new(initTable))
}

func (i *initTable) Version() int64 {
	return 2024_07_07_001
}

func (i *initTable) Desc() string {
	return "初始化数据表"
}

func (i *initTable) Exec(e *xorm.Engine) error {
	err := AutoCreateTable(e, new(table.Admin))
	if err != nil {
		return err
	}

	return initDefaultAdmin(e)
}

func initDefaultAdmin(e *xorm.Engine) error {
	if exist, err := e.Exist(&table.Admin{Username: defaultAdminUsername}); err != nil {
		return err
	} else if !exist {
		// 创建默认管理员
		sha256Str := util.Sha256Encode(defaultAdminPassword)
		encodePassword, err := util.BcryptEncode(sha256Str)
		if err != nil {
			return err
		}
		if _, err = e.InsertOne(&table.Admin{
			Username: defaultAdminUsername,
			Password: encodePassword,
		}); err != nil {
			return err
		}
	}
	return nil
}
