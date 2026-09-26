package migrate

import "xorm.io/xorm"

type recordsModel struct{}

func init()                          { registerMigrate(new(recordsModel)) }
func (*recordsModel) Version() int64 { return 2026_09_25_006 }
func (*recordsModel) Desc() string   { return "新增通用记录、观察、修订及旧资源映射" }

func (*recordsModel) Exec(e *xorm.Engine) error {
	s := e.NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	defer s.Rollback()
	_, err := s.Exec(`
CREATE TABLE IF NOT EXISTS records (
 id BIGSERIAL PRIMARY KEY, dataset_id BIGINT NOT NULL REFERENCES datasets(id),
 canonical_key VARCHAR(512) NOT NULL, normalized JSONB NOT NULL,
 status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
 content_hash CHAR(64) NOT NULL, first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (dataset_id, canonical_key),
 CONSTRAINT records_normalized_object CHECK (jsonb_typeof(normalized) = 'object')
);
CREATE TABLE IF NOT EXISTS record_observations (
 id BIGSERIAL PRIMARY KEY, record_id BIGINT NOT NULL REFERENCES records(id),
 dataset_id BIGINT NOT NULL REFERENCES datasets(id), schema_version INTEGER NOT NULL,
 workflow_id BIGINT REFERENCES workflows(id) ON DELETE SET NULL,
 workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE SET NULL,
 run_id BIGINT REFERENCES workflow_runs(id) ON DELETE SET NULL,
 task_id BIGINT REFERENCES crawl_tasks(id) ON DELETE SET NULL,
 document_id BIGINT REFERENCES documents(id) ON DELETE SET NULL,
 source_id BIGINT REFERENCES sources(id) ON DELETE SET NULL,
 source_url TEXT NOT NULL DEFAULT '', raw_fields JSONB NOT NULL,
 legacy_resource_id BIGINT REFERENCES resources(id) ON DELETE SET NULL,
 idempotency_key VARCHAR(256),
 normalized_fields JSONB NOT NULL, decision VARCHAR(32) NOT NULL
  CHECK (decision IN ('created','updated','unchanged','invalid','conflict')),
 observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 CONSTRAINT observations_raw_object CHECK (jsonb_typeof(raw_fields) = 'object'),
 CONSTRAINT observations_normalized_object CHECK (jsonb_typeof(normalized_fields) = 'object')
);
CREATE TABLE IF NOT EXISTS record_revisions (
 id BIGSERIAL PRIMARY KEY, record_id BIGINT NOT NULL REFERENCES records(id),
 observation_id BIGINT NOT NULL UNIQUE REFERENCES record_observations(id),
 before_fields JSONB NOT NULL, after_fields JSONB NOT NULL, changed_fields JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS legacy_resource_records (
 legacy_resource_id BIGINT PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
 record_id BIGINT NOT NULL REFERENCES records(id),
 observation_id BIGINT NOT NULL UNIQUE REFERENCES record_observations(id),
 migrated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS record_migration_issues (
 legacy_resource_id BIGINT PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
 reason TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_records_dataset_seen ON records(dataset_id, last_seen_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_observations_record_time ON record_observations(record_id, observed_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_observations_source ON record_observations(source_id, observed_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uq_observations_idempotency ON record_observations(dataset_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_revisions_record_time ON record_revisions(record_id, created_at DESC);
`)
	if err != nil {
		return err
	}
	return s.Commit()
}
