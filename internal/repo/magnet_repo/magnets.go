package magnet_repo

import (
	"errors"
	"fmt"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	log "github.com/sirupsen/logrus"
	"strings"
)

func Save(m *table.Magnets) {
	_, err := db.Instance().InsertOne(m)
	if err != nil {
		log.Errorf("保存资源异常：%s", err.Error())
		return
	}
}

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

func GetByFollowed(followedBy string) (*table.Magnets, bool) {
	m := new(table.Magnets)
	m.FollowedBy = followedBy
	if has, err := db.Instance().Get(m); err != nil {
		log.Errorf("查询资源FollowedBy(%s)异常：%s", followedBy, err.Error())
		return nil, false
	} else {
		return m, has
	}
}

func UpdateFollowedBy(source string, target string) error {
	m, exists := GetByFollowed(source)
	if !exists {
		// 忽略
		return errors.New(fmt.Sprintf("查询资源FollowedBy(%s)不存在", source))
	}

	m.FollowedBy = target

	if _, err := db.Instance().ID(m.Id).Cols("followed_by").Update(m); err != nil {
		return err
	}

	return nil
}
