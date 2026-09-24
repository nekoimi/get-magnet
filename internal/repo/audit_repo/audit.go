package audit_repo

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"xorm.io/xorm"
)

type Filter struct {
	ResourceType string
	Action       string
	Page         int
	Size         int
}

func Record(requestID, action, resourceType string, resourceID *int64, details any) error {
	if db.Instance() == nil {
		return errors.New("database is not initialized")
	}
	if details == nil {
		details = map[string]any{}
	}
	b, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = db.Instance().InsertOne(&table.AuditLog{RequestID: requestID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Details: string(b), CreatedAt: time.Now()})
	return err
}

func List(filter Filter) ([]table.AuditLog, int64, error) {
	if db.Instance() == nil {
		return nil, 0, errors.New("database is not initialized")
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Size <= 0 {
		filter.Size = 50
	}
	count := db.Instance().NewSession()
	defer count.Close()
	apply(count, filter)
	total, err := count.Count(new(table.AuditLog))
	if err != nil {
		return nil, 0, err
	}
	s := db.Instance().NewSession()
	defer s.Close()
	apply(s, filter)
	rows := make([]table.AuditLog, 0)
	err = s.Desc("created_at").Limit(filter.Size, (filter.Page-1)*filter.Size).Find(&rows)
	return rows, total, err
}

func apply(s *xorm.Session, filter Filter) {
	if filter.ResourceType != "" {
		s.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.Action != "" {
		s.And("action = ?", filter.Action)
	}
}
