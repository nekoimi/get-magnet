package migrate

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"xorm.io/xorm"
)

// resourceModelMigration creates the v2 resource model and performs the one
// time migration from magnets. The legacy table is intentionally left intact
// so operators can validate the result before switching reads and writes.
type resourceModelMigration struct{}

func init() { registerMigrate(new(resourceModelMigration)) }

func (m *resourceModelMigration) Version() int64 { return 2026_09_24_001 }
func (m *resourceModelMigration) Desc() string {
	return "创建 v2 通用资源模型并迁移磁力数据"
}

type migrationReport struct {
	LegacyMagnets   int
	Sources         int
	Resources       int
	Links           int
	LegacyDownloads int
	Events          int
	DuplicateKeys   int
	MissingTitles   int
	MissingNumbers  int
	InvalidLinks    int
}

func (m *resourceModelMigration) Exec(e *xorm.Engine) error {
	s := e.NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = s.Rollback()
		}
	}()

	if err := createResourceSchema(s); err != nil {
		return err
	}
	report, err := migrateMagnets(s)
	if err != nil {
		return err
	}
	if report.Resources != report.LegacyMagnets {
		return fmt.Errorf("资源迁移数量不一致: magnets=%d resources=%d", report.LegacyMagnets, report.Resources)
	}
	if err := migrateMagnetEvents(s, &report); err != nil {
		return err
	}
	if err := resetResourceSequences(s); err != nil {
		return err
	}
	if err := s.Commit(); err != nil {
		return err
	}
	rollback = false

	log.WithFields(map[string]any{
		"legacy_magnets": report.LegacyMagnets,
		"sources":        report.Sources, "resources": report.Resources,
		"links": report.Links, "legacy_download_states": report.LegacyDownloads,
		"resource_events": report.Events, "duplicate_keys": report.DuplicateKeys,
		"missing_titles": report.MissingTitles, "missing_numbers": report.MissingNumbers,
		"invalid_links": report.InvalidLinks,
	}).Info("v2 资源模型迁移完成")
	return nil
}

func createResourceSchema(e *xorm.Session) error {
	_, err := e.Exec(`
CREATE TABLE IF NOT EXISTS sources (
  id BIGSERIAL PRIMARY KEY,
  code VARCHAR(64) NOT NULL UNIQUE,
  name VARCHAR(128) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  config JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT sources_config_size CHECK (pg_column_size(config) <= 1048576)
);
CREATE TABLE IF NOT EXISTS workflows (
  id BIGSERIAL PRIMARY KEY,
  source_id BIGINT NOT NULL REFERENCES sources(id),
  code VARCHAR(128) NOT NULL,
  name VARCHAR(256) NOT NULL,
  resource_type VARCHAR(64) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  published_version_id BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (source_id, code)
);
CREATE TABLE IF NOT EXISTS workflow_versions (
  id BIGSERIAL PRIMARY KEY,
  workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  version INTEGER NOT NULL CHECK (version > 0),
  status VARCHAR(32) NOT NULL,
  definition JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  published_at TIMESTAMPTZ,
  UNIQUE (workflow_id, version),
  CONSTRAINT workflow_versions_definition_size CHECK (pg_column_size(definition) <= 5242880)
);
CREATE TABLE IF NOT EXISTS workflow_runs (
  id BIGSERIAL PRIMARY KEY,
  workflow_id BIGINT NOT NULL REFERENCES workflows(id),
  workflow_version_id BIGINT NOT NULL REFERENCES workflow_versions(id),
  trigger_type VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL,
  input JSONB NOT NULL DEFAULT '{}'::jsonb,
  summary JSONB NOT NULL DEFAULT '{}'::jsonb,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_by BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT workflow_runs_input_size CHECK (pg_column_size(input) <= 1048576),
  CONSTRAINT workflow_runs_summary_size CHECK (pg_column_size(summary) <= 1048576)
);
CREATE TABLE IF NOT EXISTS documents (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT,
  document_type VARCHAR(32) NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  content_hash VARCHAR(128),
  content_size BIGINT NOT NULL DEFAULT 0 CHECK (content_size >= 0),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT documents_metadata_size CHECK (pg_column_size(metadata) <= 1048576)
);
CREATE TABLE IF NOT EXISTS crawl_tasks (
  id BIGSERIAL PRIMARY KEY,
  run_id BIGINT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  parent_task_id BIGINT REFERENCES crawl_tasks(id) ON DELETE SET NULL,
  step_name VARCHAR(128) NOT NULL,
  task_type VARCHAR(64) NOT NULL,
  input JSONB NOT NULL DEFAULT '{}'::jsonb,
  status VARCHAR(32) NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  max_attempts INTEGER NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
  next_retry_at TIMESTAMPTZ,
  lease_owner VARCHAR(128),
  lease_until TIMESTAMPTZ,
  output_document_id BIGINT REFERENCES documents(id) ON DELETE SET NULL,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at TIMESTAMPTZ,
  CONSTRAINT crawl_tasks_input_size CHECK (pg_column_size(input) <= 1048576)
);
CREATE TABLE IF NOT EXISTS task_attempts (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES crawl_tasks(id) ON DELETE CASCADE,
  attempt_no INTEGER NOT NULL CHECK (attempt_no > 0),
  worker_id VARCHAR(128),
  status VARCHAR(32) NOT NULL,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  duration_ms BIGINT NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
  request_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  response_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  error_message TEXT,
  UNIQUE (task_id, attempt_no),
  CONSTRAINT task_attempts_request_size CHECK (pg_column_size(request_snapshot) <= 1048576),
  CONSTRAINT task_attempts_response_size CHECK (pg_column_size(response_snapshot) <= 1048576)
);
CREATE TABLE IF NOT EXISTS resources (
  id BIGSERIAL PRIMARY KEY,
  resource_type VARCHAR(64) NOT NULL,
  source_id BIGINT NOT NULL REFERENCES sources(id),
  source_url TEXT NOT NULL DEFAULT '',
  canonical_key VARCHAR(512) NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL,
  attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (source_id, resource_type, canonical_key),
  CONSTRAINT resources_status CHECK (status IN ('discovered', 'collected', 'validated', 'invalid', 'duplicate', 'archived')),
  CONSTRAINT resources_attributes_size CHECK (pg_column_size(attributes) <= 1048576)
);
CREATE TABLE IF NOT EXISTS resource_links (
  id BIGSERIAL PRIMARY KEY,
  resource_id BIGINT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
  link_type VARCHAR(32) NOT NULL,
  link TEXT NOT NULL,
  name VARCHAR(256) NOT NULL DEFAULT '',
  priority INTEGER NOT NULL DEFAULT 0,
  is_optimal BOOLEAN NOT NULL DEFAULT FALSE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT resource_links_metadata_size CHECK (pg_column_size(metadata) <= 262144),
  UNIQUE (resource_id, link_type, link)
);
CREATE TABLE IF NOT EXISTS resource_events (
  id BIGSERIAL PRIMARY KEY,
  resource_id BIGINT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
  event_type VARCHAR(64) NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT resource_events_data_size CHECK (pg_column_size(data) <= 262144)
);
CREATE TABLE IF NOT EXISTS legacy_download_states (
  resource_id BIGINT PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
  followed_by VARCHAR(512) NOT NULL DEFAULT '',
  old_status SMALLINT NOT NULL DEFAULT 0,
  post_process_done BOOLEAN NOT NULL DEFAULT FALSE,
  play_file_id VARCHAR(255) NOT NULL DEFAULT '',
  play_file_path TEXT NOT NULL DEFAULT '',
  play_file_size BIGINT NOT NULL DEFAULT 0,
  strm_path TEXT NOT NULL DEFAULT '',
  download_error TEXT NOT NULL DEFAULT '',
  download_retry_count INTEGER NOT NULL DEFAULT 0,
  last_submit_at TIMESTAMPTZ,
  download_completed_at TIMESTAMPTZ,
  migrated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_workflows_source_enabled ON workflows (source_id, enabled);
CREATE INDEX IF NOT EXISTS idx_workflow_versions_workflow_status ON workflow_versions (workflow_id, status);
CREATE INDEX IF NOT EXISTS idx_workflow_runs_status_created ON workflow_runs (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_crawl_tasks_run_status_retry ON crawl_tasks (run_id, status, next_retry_at);
CREATE INDEX IF NOT EXISTS idx_crawl_tasks_lease ON crawl_tasks (status, lease_until);
CREATE INDEX IF NOT EXISTS idx_task_attempts_task_started ON task_attempts (task_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_documents_task_created ON documents (task_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_resources_source_status_seen ON resources (source_id, status, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_resources_title ON resources (title);
CREATE INDEX IF NOT EXISTS idx_resource_links_resource_priority ON resource_links (resource_id, priority);
CREATE INDEX IF NOT EXISTS idx_resource_events_resource_created ON resource_events (resource_id, created_at DESC);
`)
	if err != nil {
		return err
	}
	_, err = e.Exec(`DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'workflows_published_version_fk') THEN
    ALTER TABLE workflows ADD CONSTRAINT workflows_published_version_fk
      FOREIGN KEY (published_version_id) REFERENCES workflow_versions(id) ON DELETE SET NULL;
  END IF;
END $$`)
	return err
}

func migrateMagnets(e *xorm.Session) (migrationReport, error) {
	var report migrationReport
	rows, err := e.Query(`
SELECT id, COALESCE(created_at::text, '') AS created_at_text,
       COALESCE(updated_at::text, '') AS updated_at_text,
       COALESCE(origin, '') AS origin, COALESCE(title, '') AS title,
       COALESCE(number, '') AS number, COALESCE(optimal_link, '') AS optimal_link,
       COALESCE(links::text, '') AS links_json,
       COALESCE(raw_url_host, '') AS raw_url_host, COALESCE(raw_url_path, '') AS raw_url_path,
       status, COALESCE(actress0, '') AS actress0, COALESCE(followed_by, '') AS followed_by,
       COALESCE(post_process_done, false) AS post_process_done,
       COALESCE(play_file_id, '') AS play_file_id, COALESCE(play_file_path, '') AS play_file_path,
       COALESCE(play_file_size, 0) AS play_file_size, COALESCE(strm_path, '') AS strm_path,
       COALESCE(download_error, '') AS download_error, COALESCE(download_retry_count, 0) AS download_retry_count,
       last_submit_at, download_completed_at
FROM magnets ORDER BY id`)
	if err != nil {
		return report, err
	}
	report.LegacyMagnets = len(rows)
	seen := make(map[string]int64)
	sourceIDs := make(map[string]int64)
	for _, row := range rows {
		id, err := int64Value(row["id"])
		if err != nil {
			return report, fmt.Errorf("读取 magnets.id 失败: %w", err)
		}
		origin := stringValue(row["origin"])
		code := normalizeSourceCode(origin)
		sourceID, ok := sourceIDs[code]
		if !ok {
			sourceID, err = ensureSource(e, code, origin)
			if err != nil {
				return report, err
			}
			sourceIDs[code] = sourceID
			report.Sources++
		}

		title := stringValue(row["title"])
		number := normalizeCanonicalKey(stringValue(row["number"]))
		sourceURL := buildSourceURL(stringValue(row["raw_url_host"]), stringValue(row["raw_url_path"]))
		if title == "" {
			report.MissingTitles++
		}
		if number == "" {
			report.MissingNumbers++
		}
		if number == "" {
			number = sourceURL
		}
		if number == "" {
			number = fmt.Sprintf("legacy-id:%d", id)
		}
		duplicateKey := fmt.Sprintf("%d\x00magnet\x00%s", sourceID, number)
		status := "collected"
		links, linksErr := parseLegacyLinks(stringValue(row["links_json"]))
		if linksErr != nil {
			report.InvalidLinks++
		}
		optimal := strings.TrimSpace(stringValue(row["optimal_link"]))
		if optimal != "" && !containsString(links, optimal) {
			links = append(links, optimal)
		}
		validLinkCount := 0
		for _, link := range links {
			if isValidResourceLink(link) {
				validLinkCount++
			} else {
				report.InvalidLinks++
			}
		}
		if title == "" || validLinkCount == 0 {
			status = "invalid"
		}
		if firstID, exists := seen[duplicateKey]; exists {
			report.DuplicateKeys++
			status = "duplicate"
			number = fitCanonicalKey(fmt.Sprintf("%s#duplicate:%d", number, id))
			log.WithFields(map[string]any{"resource_id": id, "original_id": firstID, "canonical_key": number}).Warn("迁移发现重复 canonical key")
		} else {
			seen[duplicateKey] = id
		}

		createdAt := timestampValue(row["created_at_text"])
		updatedAt := timestampValue(row["updated_at_text"])
		if createdAt == "" {
			createdAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if updatedAt == "" {
			updatedAt = createdAt
		}
		attributes := attributesJSON(stringValue(row["actress0"]))
		_, err = e.Exec(`INSERT INTO resources
 (id, resource_type, source_id, source_url, canonical_key, title, status, attributes,
  first_seen_at, last_seen_at, created_at, updated_at)
 VALUES (?, 'magnet', ?, ?, ?, ?, ?, ?::jsonb, ?, ?, ?, ?)
 ON CONFLICT (id) DO NOTHING`, id, sourceID, sourceURL, number, title, status, attributes,
			createdAt, updatedAt, createdAt, updatedAt)
		if err != nil {
			return report, fmt.Errorf("迁移 magnet %d 到 resources 失败: %w", id, err)
		}
		report.Resources++
		for priority, link := range links {
			linkType := linkType(link)
			isOptimal := optimal != "" && link == optimal
			_, err = e.Exec(`INSERT INTO resource_links
 (resource_id, link_type, link, priority, is_optimal, metadata)
 VALUES (?, ?, ?, ?, ?, '{}'::jsonb)
 ON CONFLICT (resource_id, link_type, link) DO UPDATE SET is_optimal = resource_links.is_optimal OR EXCLUDED.is_optimal`,
				id, linkType, link, priority, isOptimal)
			if err != nil {
				return report, fmt.Errorf("迁移 magnet %d 的链接失败: %w", id, err)
			}
			report.Links++
		}
		_, err = e.Exec(`INSERT INTO legacy_download_states
 (resource_id, followed_by, old_status, post_process_done, play_file_id, play_file_path,
  play_file_size, strm_path, download_error, download_retry_count, last_submit_at,
  download_completed_at)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
 ON CONFLICT (resource_id) DO NOTHING`, id, stringValue(row["followed_by"]), intValue(row["status"]),
			boolValue(row["post_process_done"]), stringValue(row["play_file_id"]), stringValue(row["play_file_path"]),
			int64OrZero(row["play_file_size"]), stringValue(row["strm_path"]), stringValue(row["download_error"]),
			intValue(row["download_retry_count"]), nullableTimestamp(row["last_submit_at"]), nullableTimestamp(row["download_completed_at"]))
		if err != nil {
			return report, fmt.Errorf("迁移 magnet %d 的下载历史失败: %w", id, err)
		}
		report.LegacyDownloads++
	}
	return report, nil
}

func migrateMagnetEvents(e *xorm.Session, report *migrationReport) error {
	rows, err := e.Query(`
INSERT INTO resource_events (id, resource_id, event_type, message, data, created_at)
SELECT me.id, me.magnet_id, me.event_type, COALESCE(me.message, ''),
       CASE WHEN COALESCE(me.extra, '') = '' THEN '{}'::jsonb
            ELSE jsonb_build_object('extra', me.extra) END,
       me.created_at
FROM magnet_events me
JOIN resources r ON r.id = me.magnet_id
ON CONFLICT (id) DO NOTHING
RETURNING id`)
	if err != nil {
		// Older installations may not have magnet_events. The table is part of
		// the old migrations, but allowing the migration to proceed is safer.
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			return nil
		}
		return err
	}
	report.Events += len(rows)
	return nil
}

func resetResourceSequences(e *xorm.Session) error {
	for _, table := range []string{"sources", "workflows", "workflow_versions", "workflow_runs", "documents", "crawl_tasks", "task_attempts", "resources", "resource_links", "resource_events"} {
		if _, err := e.Exec(fmt.Sprintf(`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE(MAX(id), 1), MAX(id) IS NOT NULL) FROM %s`, table, table)); err != nil {
			return err
		}
	}
	return nil
}

func ensureSource(e *xorm.Session, code, origin string) (int64, error) {
	name := strings.TrimSpace(origin)
	if name == "" {
		name = "Legacy"
	}
	name = truncateUTF8(name, 128)
	if _, err := e.Exec(`INSERT INTO sources (code, name) VALUES (?, ?) ON CONFLICT (code) DO UPDATE SET updated_at = NOW()`, code, name); err != nil {
		return 0, err
	}
	rows, err := e.Query(`SELECT id FROM sources WHERE code = ?`, code)
	if err != nil || len(rows) == 0 {
		if err == nil {
			err = sql.ErrNoRows
		}
		return 0, err
	}
	return int64Value(rows[0]["id"])
}

func normalizeSourceCode(origin string) string {
	code := strings.TrimSpace(origin)
	if code == "" {
		return "legacy"
	}
	code = strings.ToLower(code)
	if len(code) <= 64 {
		return code
	}
	hash := sha256.Sum256([]byte(code))
	return truncateUTF8(code, 55) + "-" + hex.EncodeToString(hash[:])[:8]
}

func normalizeCanonicalKey(value string) string {
	key := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToUpper(r)
	}, strings.TrimSpace(value))
	return fitCanonicalKey(key)
}

func fitCanonicalKey(key string) string {
	if len(key) <= 512 {
		return key
	}
	hash := sha256.Sum256([]byte(key))
	return truncateUTF8(key, 495) + "-" + hex.EncodeToString(hash[:])[:16]
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	for len(value) > maxBytes {
		_, size := utf8.DecodeLastRuneInString(value)
		value = value[:len(value)-size]
	}
	return value
}

func buildSourceURL(host, path string) string {
	host, path = strings.TrimSpace(host), strings.TrimSpace(path)
	if host == "" {
		return path
	}
	if parsed, err := url.Parse(host); err == nil && parsed.Scheme != "" {
		return strings.TrimRight(host, "/") + "/" + strings.TrimLeft(path, "/")
	}
	return "https://" + strings.TrimRight(host, "/") + "/" + strings.TrimLeft(path, "/")
}

func parseLegacyLinks(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !containsString(result, value) {
			result = append(result, value)
		}
	}
	return result, nil
}

func attributesJSON(actress string) string {
	actress = strings.TrimSpace(actress)
	if actress == "" {
		return "{}"
	}
	b, _ := json.Marshal(map[string]string{"actress": actress})
	return string(b)
}

func linkType(link string) string {
	if strings.HasPrefix(strings.ToLower(link), "magnet:") {
		return "magnet"
	}
	return "url"
}

func isValidResourceLink(link string) bool {
	link = strings.TrimSpace(link)
	if strings.HasPrefix(strings.ToLower(link), "magnet:") {
		return strings.Contains(strings.ToLower(link), "xt=")
	}
	parsed, err := url.Parse(link)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringValue(value []byte) string { return strings.TrimSpace(string(value)) }

func int64Value(value []byte) (int64, error) {
	var result int64
	_, err := fmt.Sscan(string(value), &result)
	return result, err
}

func int64OrZero(value []byte) int64 { result, _ := int64Value(value); return result }
func intValue(value []byte) int      { return int(int64OrZero(value)) }

func boolValue(value []byte) bool {
	return strings.EqualFold(strings.TrimSpace(string(value)), "true") || strings.TrimSpace(string(value)) == "1"
}

func timestampValue(value []byte) string {
	value = []byte(strings.TrimSpace(string(value)))
	if len(value) == 0 {
		return ""
	}
	return string(value)
}

func nullableTimestamp(value []byte) any {
	value = []byte(strings.TrimSpace(string(value)))
	if len(value) == 0 || string(value) == "<nil>" {
		return nil
	}
	return string(value)
}
