package migrate

import "xorm.io/xorm"

type projectDatasetModel struct{}

func init() { registerMigrate(new(projectDatasetModel)) }

func (*projectDatasetModel) Version() int64 { return 2026_09_25_005 }
func (*projectDatasetModel) Desc() string   { return "新增项目与版本化数据集 Schema" }

func (*projectDatasetModel) Exec(e *xorm.Engine) error {
	s := e.NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	defer s.Rollback()
	_, err := s.Exec(`
CREATE TABLE IF NOT EXISTS projects (
 id BIGSERIAL PRIMARY KEY, code VARCHAR(128) NOT NULL UNIQUE, name VARCHAR(256) NOT NULL,
 goal TEXT NOT NULL DEFAULT '', owner VARCHAR(128) NOT NULL DEFAULT '',
 status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active','paused','archived')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS datasets (
 id BIGSERIAL PRIMARY KEY, project_id BIGINT NOT NULL REFERENCES projects(id),
 code VARCHAR(128) NOT NULL, name VARCHAR(256) NOT NULL, record_type VARCHAR(64) NOT NULL,
 schema_version INTEGER NOT NULL DEFAULT 1 CHECK (schema_version > 0),
 unique_key_fields JSONB NOT NULL, empty_value_policy VARCHAR(32) NOT NULL DEFAULT 'preserve'
  CHECK (empty_value_policy IN ('preserve','overwrite')),
 status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active','paused','archived')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (project_id, code),
 CONSTRAINT datasets_key_fields_array CHECK (jsonb_typeof(unique_key_fields) = 'array' AND jsonb_array_length(unique_key_fields) > 0)
);
CREATE TABLE IF NOT EXISTS dataset_fields (
 id BIGSERIAL PRIMARY KEY, dataset_id BIGINT NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
 schema_version INTEGER NOT NULL CHECK (schema_version > 0),
 field_key VARCHAR(128) NOT NULL, label VARCHAR(256) NOT NULL,
 field_type VARCHAR(32) NOT NULL CHECK (field_type IN ('string','integer','number','boolean','datetime','url','json')),
 required BOOLEAN NOT NULL DEFAULT FALSE, multiple BOOLEAN NOT NULL DEFAULT FALSE,
 ordinal INTEGER NOT NULL DEFAULT 0, UNIQUE (dataset_id, schema_version, field_key)
);
CREATE TABLE IF NOT EXISTS dataset_schema_versions (
 dataset_id BIGINT NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
 version INTEGER NOT NULL CHECK (version > 0), unique_key_fields JSONB NOT NULL,
 empty_value_policy VARCHAR(32) NOT NULL CHECK (empty_value_policy IN ('preserve','overwrite')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY (dataset_id, version)
);
CREATE INDEX IF NOT EXISTS idx_datasets_project ON datasets(project_id, id);
CREATE INDEX IF NOT EXISTS idx_dataset_fields_version ON dataset_fields(dataset_id, schema_version, ordinal);
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS project_id BIGINT REFERENCES projects(id);
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS dataset_id BIGINT REFERENCES datasets(id);
CREATE INDEX IF NOT EXISTS idx_workflows_project_dataset ON workflows(project_id, dataset_id);
INSERT INTO projects (code, name, goal) VALUES ('default', '默认项目', '管理现有采集器与数据集') ON CONFLICT (code) DO NOTHING;
INSERT INTO datasets (project_id, code, name, record_type, unique_key_fields)
 SELECT id, 'magnet', '磁力数据', 'magnet', '["number"]'::jsonb FROM projects WHERE code = 'default'
 ON CONFLICT (project_id, code) DO NOTHING;
INSERT INTO dataset_fields (dataset_id, schema_version, field_key, label, field_type, required, ordinal)
 SELECT d.id, d.schema_version, f.field_key, f.label, f.field_type, f.required, f.ordinal
 FROM datasets d JOIN projects p ON p.id = d.project_id,
 (VALUES ('number','番号','string',true,0),('title','标题','string',false,1),('links','磁力链接','url',false,2))
 AS f(field_key,label,field_type,required,ordinal)
 WHERE p.code = 'default' AND d.code = 'magnet'
 ON CONFLICT (dataset_id, schema_version, field_key) DO NOTHING;
UPDATE dataset_fields SET multiple = true WHERE field_key = 'links' AND dataset_id IN
 (SELECT d.id FROM datasets d JOIN projects p ON p.id = d.project_id WHERE p.code = 'default' AND d.code = 'magnet');
INSERT INTO datasets (project_id, code, name, record_type, unique_key_fields)
 SELECT id, 'article', '文章数据', 'article', '["url"]'::jsonb FROM projects WHERE code = 'default'
 ON CONFLICT (project_id, code) DO NOTHING;
INSERT INTO dataset_fields (dataset_id, schema_version, field_key, label, field_type, required, ordinal)
 SELECT d.id, d.schema_version, f.field_key, f.label, f.field_type, f.required, f.ordinal
 FROM datasets d JOIN projects p ON p.id = d.project_id,
 (VALUES ('url','文章链接','url',true,0),('title','标题','string',false,1),('published_at','发布时间','datetime',false,2),('body','正文','string',false,3))
 AS f(field_key,label,field_type,required,ordinal)
 WHERE p.code = 'default' AND d.code = 'article'
 ON CONFLICT (dataset_id, schema_version, field_key) DO NOTHING;
INSERT INTO dataset_schema_versions (dataset_id, version, unique_key_fields, empty_value_policy)
 SELECT id, schema_version, unique_key_fields, empty_value_policy FROM datasets WHERE true
 ON CONFLICT (dataset_id, version) DO NOTHING;
UPDATE workflows SET project_id = (SELECT id FROM projects WHERE code = 'default') WHERE project_id IS NULL;
UPDATE workflows SET dataset_id = (SELECT d.id FROM datasets d JOIN projects p ON p.id = d.project_id
 WHERE p.code = 'default' AND d.code = 'magnet') WHERE dataset_id IS NULL AND resource_type = 'magnet';
ALTER TABLE workflows ALTER COLUMN project_id SET NOT NULL;
`)
	if err != nil {
		return err
	}
	return s.Commit()
}
