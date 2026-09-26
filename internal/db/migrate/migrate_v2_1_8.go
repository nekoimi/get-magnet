package migrate

import "xorm.io/xorm"

type workflowSchedules struct{}

func init()                               { registerMigrate(new(workflowSchedules)) }
func (*workflowSchedules) Version() int64 { return 2026_09_26_005 }
func (*workflowSchedules) Desc() string   { return "工作流持久化调度和触发事件" }
func (*workflowSchedules) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
CREATE TABLE IF NOT EXISTS workflow_schedules (
 workflow_id BIGINT PRIMARY KEY REFERENCES workflows(id) ON DELETE CASCADE,
 cron TEXT NOT NULL, timezone TEXT NOT NULL, enabled BOOLEAN NOT NULL DEFAULT false,
 concurrency_policy TEXT NOT NULL CHECK (concurrency_policy IN ('skip','queue')),
 next_run_at TIMESTAMPTZ, last_run_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_workflow_schedules_due ON workflow_schedules(next_run_at) WHERE enabled;
CREATE TABLE IF NOT EXISTS workflow_schedule_events (
 id BIGSERIAL PRIMARY KEY, workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
 scheduled_at TIMESTAMPTZ NOT NULL, status TEXT NOT NULL CHECK (status IN ('pending','started','skipped','failed')),
 run_id BIGINT REFERENCES workflow_runs(id) ON DELETE SET NULL, reason TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(workflow_id,scheduled_at)
);
CREATE INDEX IF NOT EXISTS idx_workflow_schedule_events_pending ON workflow_schedule_events(workflow_id,id) WHERE status='pending';
`)
	return err
}
