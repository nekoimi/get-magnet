package migrate

import "xorm.io/xorm"

type aiExtractionsModel struct{}

func init() { registerMigrate(new(aiExtractionsModel)) }

func (m *aiExtractionsModel) Version() int64 { return 2026_09_25_003 }
func (m *aiExtractionsModel) Desc() string   { return "新增 AI 结构化提取缓存表" }

func (m *aiExtractionsModel) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
CREATE TABLE IF NOT EXISTS ai_extractions (
  id BIGSERIAL PRIMARY KEY,
  cache_key VARCHAR(128) NOT NULL UNIQUE,
  document_id BIGINT REFERENCES documents(id) ON DELETE SET NULL,
  workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE SET NULL,
  model VARCHAR(128) NOT NULL,
  result JSONB NOT NULL DEFAULT '{}'::jsonb,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (confidence >= 0 AND confidence <= 1),
  review_status VARCHAR(32) NOT NULL DEFAULT 'pending_review',
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT ai_extractions_result_size CHECK (pg_column_size(result) <= 1048576)
);
CREATE INDEX IF NOT EXISTS idx_ai_extractions_document ON ai_extractions (document_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_extractions_review ON ai_extractions (review_status, created_at DESC);
`)
	return err
}
