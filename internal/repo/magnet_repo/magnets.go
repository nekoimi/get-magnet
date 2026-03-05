package magnet_repo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	log "github.com/sirupsen/logrus"
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

// GetById 根据 ID 获取磁力链接
func GetById(id int64) (*table.Magnets, bool) {
	m := new(table.Magnets)
	if has, err := db.Instance().ID(id).Get(m); err != nil {
		log.Errorf("查询资源ID(%d)异常：%s", id, err.Error())
		return nil, false
	} else {
		return m, has
	}
}

// PageList 分页查询磁力链接列表
func PageList(pageNum, pageSize int, keyword string, status *uint8) ([]table.Magnets, int64, error) {
	session := db.Instance().NewSession()
	defer session.Close()

	// 构建查询条件
	if keyword != "" {
		session = session.Where("(title LIKE ? OR number LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status != nil {
		session = session.Where("status = ?", *status)
	}

	// 获取总数
	total, err := session.Count(new(table.Magnets))
	if err != nil {
		log.Errorf("查询磁力链接总数异常：%s", err.Error())
		return nil, 0, err
	}

	// 分页查询
	var list []table.Magnets
	err = session.OrderBy("created_at DESC").Limit(pageSize, (pageNum-1)*pageSize).Find(&list)
	if err != nil {
		log.Errorf("查询磁力链接列表异常：%s", err.Error())
		return nil, 0, err
	}

	return list, total, nil
}

// Update 更新磁力链接
func Update(m *table.Magnets) error {
	_, err := db.Instance().ID(m.Id).AllCols().Update(m)
	if err != nil {
		log.Errorf("更新磁力链接异常：%s", err.Error())
		return err
	}
	return nil
}

// Delete 删除磁力链接
func Delete(id int64) error {
	_, err := db.Instance().ID(id).Delete(new(table.Magnets))
	if err != nil {
		log.Errorf("删除磁力链接异常：%s", err.Error())
		return err
	}
	return nil
}

// BatchDelete 批量删除磁力链接
func BatchDelete(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := db.Instance().In("id", ids).Delete(new(table.Magnets))
	if err != nil {
		log.Errorf("批量删除磁力链接异常：%s", err.Error())
		return err
	}
	return nil
}
