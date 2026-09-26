package migrate

import "xorm.io/xorm"

type listingSampleRole struct{}

func init()                               { registerMigrate(new(listingSampleRole)) }
func (*listingSampleRole) Version() int64 { return 2026_09_26_003 }
func (*listingSampleRole) Desc() string   { return "区分列表与详情样例" }
func (*listingSampleRole) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`ALTER TABLE workflow_samples ADD COLUMN IF NOT EXISTS page_role VARCHAR(16) NOT NULL DEFAULT 'trigger';
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'workflow_samples_page_role_check') THEN
ALTER TABLE workflow_samples ADD CONSTRAINT workflow_samples_page_role_check CHECK (page_role IN ('trigger','list','detail'));
END IF; END $$;`)
	return err
}
