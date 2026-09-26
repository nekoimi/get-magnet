package migrate

import "xorm.io/xorm"

type runBudgets struct{}

func init()                        { registerMigrate(new(runBudgets)) }
func (*runBudgets) Version() int64 { return 2026_09_26_004 }
func (*runBudgets) Desc() string   { return "持久化运行预算、页面预留与截断证据" }
func (*runBudgets) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS budget JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE crawl_tasks ADD COLUMN IF NOT EXISTS depth INTEGER NOT NULL DEFAULT 0;
ALTER TABLE crawl_tasks ADD COLUMN IF NOT EXISTS page_reserved BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE crawl_tasks ADD COLUMN IF NOT EXISTS page_host TEXT NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS run_limit_events (
 id BIGSERIAL PRIMARY KEY, run_id BIGINT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
 task_id BIGINT NOT NULL REFERENCES crawl_tasks(id) ON DELETE CASCADE,
 page_role TEXT NOT NULL, url TEXT NOT NULL, reason TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(run_id,task_id,page_role,url,reason)
);
CREATE INDEX IF NOT EXISTS idx_run_limit_events_run ON run_limit_events(run_id,id);
CREATE TABLE IF NOT EXISTS run_domains (
 run_id BIGINT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
 hostname TEXT NOT NULL, PRIMARY KEY(run_id,hostname)
);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_url ON crawl_tasks(run_id,step_name,(input->>'url')) WHERE task_type='workflow';
`)
	return err
}
