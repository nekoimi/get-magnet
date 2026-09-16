package migrate

import (
	"github.com/nekoimi/get-magnet/internal/db/table"
	"xorm.io/xorm"
)

type magnetEvents struct {
}

func init() {
	registerMigrate(new(magnetEvents))
}

func (m *magnetEvents) Version() int64 {
	return 2026_07_24_002
}

func (m *magnetEvents) Desc() string {
	return "新增资源事件表"
}

func (m *magnetEvents) Exec(e *xorm.Engine) error {
	if err := AutoCreateTable(e, new(table.MagnetEvent)); err != nil {
		return err
	}
	_, err := e.Exec(`
CREATE INDEX IF NOT EXISTS idx_magnet_events_magnet_id ON magnet_events (magnet_id);
CREATE INDEX IF NOT EXISTS idx_magnet_events_event_type ON magnet_events (event_type);
CREATE INDEX IF NOT EXISTS idx_magnet_events_created_at ON magnet_events (created_at);
`)
	return err
}
