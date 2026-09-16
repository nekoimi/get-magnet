package migrate

import "xorm.io/xorm"

type magnetPlayFileSize struct {
}

func init() {
	registerMigrate(new(magnetPlayFileSize))
}

func (m *magnetPlayFileSize) Version() int64 {
	return 2026_06_02_002
}

func (m *magnetPlayFileSize) Desc() string {
	return "为磁力信息增加播放文件大小字段"
}

func (m *magnetPlayFileSize) Exec(e *xorm.Engine) error {
	_, err := e.Exec("ALTER TABLE magnets ADD COLUMN IF NOT EXISTS play_file_size bigint NOT NULL DEFAULT 0")
	return err
}
