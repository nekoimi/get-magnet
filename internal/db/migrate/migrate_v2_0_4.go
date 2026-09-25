package migrate

import "xorm.io/xorm"

type pluginTasksModel struct{}

func init() { registerMigrate(new(pluginTasksModel)) }

func (m *pluginTasksModel) Version() int64 { return 2026_09_25_002 }
func (m *pluginTasksModel) Desc() string   { return "新增资源插件任务表" }

func (m *pluginTasksModel) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
CREATE TABLE IF NOT EXISTS plugin_tasks (
  id BIGSERIAL PRIMARY KEY,
  resource_id BIGINT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
  event_type VARCHAR(64) NOT NULL,
  plugin_code VARCHAR(128) NOT NULL,
  idempotency_key VARCHAR(512) NOT NULL,
  status VARCHAR(32) NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  max_attempts INTEGER NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
  next_retry_at TIMESTAMPTZ,
  external_id VARCHAR(256) NOT NULL DEFAULT '',
  input JSONB NOT NULL DEFAULT '{}'::jsonb,
  output JSONB NOT NULL DEFAULT '{}'::jsonb,
  error_message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at TIMESTAMPTZ,
  CONSTRAINT plugin_tasks_input_size CHECK (pg_column_size(input) <= 1048576),
  CONSTRAINT plugin_tasks_output_size CHECK (pg_column_size(output) <= 1048576),
  CONSTRAINT plugin_tasks_idempotency_unique UNIQUE (plugin_code, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_plugin_tasks_status_retry ON plugin_tasks (status, next_retry_at, created_at);
CREATE INDEX IF NOT EXISTS idx_plugin_tasks_resource ON plugin_tasks (resource_id, created_at DESC);
`)
	return err
}
