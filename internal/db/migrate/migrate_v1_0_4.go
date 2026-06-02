package migrate

import "xorm.io/xorm"

type magnetPlayInfo struct {
}

func init() {
	registerMigrate(new(magnetPlayInfo))
}

func (m *magnetPlayInfo) Version() int64 {
	return 2026_06_02_001
}

func (m *magnetPlayInfo) Desc() string {
	return "为磁力信息增加播放文件和strm路径字段"
}

func (m *magnetPlayInfo) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS play_file_id varchar(255) NOT NULL DEFAULT '';
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS play_file_path text NOT NULL DEFAULT '';
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS strm_path text NOT NULL DEFAULT '';
`)
	return err
}
