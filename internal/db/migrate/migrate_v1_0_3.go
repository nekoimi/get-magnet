package migrate

import "xorm.io/xorm"

type magnetPostProcess struct {
}

func init() {
	registerMigrate(new(magnetPostProcess))
}

func (m *magnetPostProcess) Version() int64 {
	return 2026_04_16_001
}

func (m *magnetPostProcess) Desc() string {
	return "为磁力信息增加下载后处理标记"
}

func (m *magnetPostProcess) Exec(e *xorm.Engine) error {
	_, err := e.Exec("ALTER TABLE magnets ADD COLUMN IF NOT EXISTS post_process_done boolean NOT NULL DEFAULT false")
	return err
}
