package migrate

import "xorm.io/xorm"

type workflowSamples struct{}

func init()                             { registerMigrate(new(workflowSamples)) }
func (*workflowSamples) Version() int64 { return 2026_09_26_002 }
func (*workflowSamples) Desc() string   { return "保存 v2.1 工作流版本样例" }
func (*workflowSamples) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`CREATE TABLE IF NOT EXISTS workflow_samples (
 id BIGSERIAL PRIMARY KEY,
 workflow_version_id BIGINT NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
 source VARCHAR(16) NOT NULL CHECK (source IN ('live','document','paste')),
 document_id BIGINT REFERENCES documents(id) ON DELETE SET NULL,
 page_url TEXT NOT NULL DEFAULT '',
 content_type VARCHAR(16) NOT NULL CHECK (content_type IN ('html','json')),
 content TEXT NOT NULL,
 content_hash CHAR(64) NOT NULL,
 note TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 CONSTRAINT workflow_samples_nonempty CHECK (length(content) > 0)
);
CREATE INDEX IF NOT EXISTS idx_workflow_samples_version ON workflow_samples(workflow_version_id,id);`)
	return err
}
