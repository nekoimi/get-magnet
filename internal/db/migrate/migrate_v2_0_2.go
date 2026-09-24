package migrate

import (
	"github.com/nekoimi/get-magnet/internal/db/table"
	"xorm.io/xorm"
)

type observabilityModel struct{}

func init() { registerMigrate(new(observabilityModel)) }

func (m *observabilityModel) Version() int64 { return 2026_09_24_002 }
func (m *observabilityModel) Desc() string   { return "新增操作审计日志表" }

func (m *observabilityModel) Exec(e *xorm.Engine) error {
	if err := AutoCreateTable(e, new(table.AuditLog)); err != nil {
		return err
	}
	_, err := e.Exec(`
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs (resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_request ON audit_logs (request_id);
`)
	return err
}
