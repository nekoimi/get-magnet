package magnet_event_repo

import (
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	log "github.com/sirupsen/logrus"
)

func Record(magnetID int64, eventType string, message string, extra string) {
	if magnetID <= 0 || eventType == "" {
		return
	}
	event := &table.MagnetEvent{
		MagnetId:  magnetID,
		EventType: eventType,
		Message:   message,
		Extra:     extra,
	}
	if _, err := db.Instance().InsertOne(event); err != nil {
		log.Errorf("记录资源事件异常：%d - %s - %s", magnetID, eventType, err.Error())
	}
}

func ListByMagnetID(magnetID int64, limit int) ([]table.MagnetEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	var list []table.MagnetEvent
	err := db.Instance().
		Where("magnet_id = ?", magnetID).
		OrderBy("created_at DESC").
		Limit(limit).
		Find(&list)
	if err != nil {
		log.Errorf("查询资源事件异常：%d - %s", magnetID, err.Error())
		return nil, err
	}
	return list, nil
}
