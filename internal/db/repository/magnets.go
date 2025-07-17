package repository

import (
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	log "github.com/sirupsen/logrus"
	"strings"
)

func ExistsByPath(rowURLPath string) bool {
	m := new(table.Magnets)
	m.RawURLPath = rowURLPath
	if exist, err := db.Instance().Exist(m); err != nil {
		log.Errorf("查询资源Path(%s)是否存在异常：%s", rowURLPath, err.Error())
		return false
	} else {
		return exist
	}
}

func ExistsByNumber(number string) bool {
	m := new(table.Magnets)
	m.Number = strings.ToUpper(number)
	if exist, err := db.Instance().Exist(m); err != nil {
		log.Errorf("查询资源Number(%s)是否存在异常：%s", number, err.Error())
		return false
	} else {
		return exist
	}
}
