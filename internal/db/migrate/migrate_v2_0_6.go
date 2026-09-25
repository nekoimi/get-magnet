package migrate

import "xorm.io/xorm"

type pluginTaskLeasesModel struct{}

func init() { registerMigrate(new(pluginTaskLeasesModel)) }

func (m *pluginTaskLeasesModel) Version() int64 { return 2026_09_25_004 }
func (m *pluginTaskLeasesModel) Desc() string   { return "为插件任务增加租约字段" }

func (m *pluginTaskLeasesModel) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
ALTER TABLE plugin_tasks ADD COLUMN IF NOT EXISTS lease_owner VARCHAR(128);
ALTER TABLE plugin_tasks ADD COLUMN IF NOT EXISTS lease_until TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_plugin_tasks_lease ON plugin_tasks (status, lease_until);
`)
	return err
}
