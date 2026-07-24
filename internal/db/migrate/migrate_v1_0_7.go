package migrate

import "xorm.io/xorm"

type magnetManagementIndexes struct {
}

func init() {
	registerMigrate(new(magnetManagementIndexes))
}

func (m *magnetManagementIndexes) Version() int64 {
	return 2026_07_24_001
}

func (m *magnetManagementIndexes) Desc() string {
	return "为在线管理端补充磁力资源常用查询索引"
}

func (m *magnetManagementIndexes) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
CREATE INDEX IF NOT EXISTS idx_magnets_number ON magnets (number);
CREATE INDEX IF NOT EXISTS idx_magnets_status ON magnets (status);
CREATE INDEX IF NOT EXISTS idx_magnets_origin ON magnets (origin);
CREATE INDEX IF NOT EXISTS idx_magnets_followed_by ON magnets (followed_by);
CREATE INDEX IF NOT EXISTS idx_magnets_created_at ON magnets (created_at);
CREATE INDEX IF NOT EXISTS idx_magnets_last_submit_at ON magnets (last_submit_at);
`)
	return err
}
