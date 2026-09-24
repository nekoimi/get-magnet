package migrate

import "xorm.io/xorm"

type documentAssetsModel struct{}

func init() { registerMigrate(new(documentAssetsModel)) }

func (m *documentAssetsModel) Version() int64 { return 2026_09_25_001 }
func (m *documentAssetsModel) Desc() string   { return "新增原始文档二进制资产表" }

func (m *documentAssetsModel) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
CREATE TABLE IF NOT EXISTS document_assets (
  id BIGSERIAL PRIMARY KEY,
  document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  asset_type VARCHAR(32) NOT NULL,
  content_type VARCHAR(128) NOT NULL,
  content BYTEA NOT NULL,
  content_hash VARCHAR(128) NOT NULL,
  content_size BIGINT NOT NULL CHECK (content_size >= 0 AND content_size <= 10485760),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT document_assets_type_unique UNIQUE (document_id, asset_type)
);
CREATE INDEX IF NOT EXISTS idx_document_assets_document ON document_assets (document_id, created_at DESC);
`)
	return err
}
