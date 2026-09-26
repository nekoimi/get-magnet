package migrate

import "xorm.io/xorm"

// The backfill is a separate, retryable transaction. It reads resources only;
// magnets and download state are deliberately left untouched.
type recordsBackfill struct{}

func init()                             { registerMigrate(new(recordsBackfill)) }
func (*recordsBackfill) Version() int64 { return 2026_09_25_007 }
func (*recordsBackfill) Desc() string {
	return "将磁力资源归并到通用记录并保存来源映射"
}
func (*recordsBackfill) Exec(e *xorm.Engine) error {
	s := e.NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	defer s.Rollback()
	_, err := s.Exec(`
CREATE EXTENSION IF NOT EXISTS pgcrypto;
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM datasets d JOIN projects p ON p.id=d.project_id
   WHERE p.code='default' AND d.code='magnet' AND d.schema_version>2) THEN
  RAISE EXCEPTION 'default magnet dataset schema was customized; reconcile it before legacy backfill';
 END IF;
END $$;
-- The original A01 template uses number. Legacy rows can have a URL fallback,
-- so v2 uses canonical_key as the required key and keeps number optional.
UPDATE datasets d SET schema_version = 2, unique_key_fields = '["canonical_key"]'::jsonb, updated_at = NOW()
FROM projects p WHERE d.project_id = p.id AND p.code = 'default' AND d.code = 'magnet' AND d.schema_version = 1;
UPDATE datasets d SET unique_key_fields = '["canonical_key"]'::jsonb, updated_at=NOW()
FROM projects p WHERE d.project_id=p.id AND p.code='default' AND d.code='magnet' AND d.schema_version=2
 AND d.unique_key_fields <> '["canonical_key"]'::jsonb;
INSERT INTO dataset_fields (dataset_id,schema_version,field_key,label,field_type,required,multiple,ordinal)
 SELECT d.id,2,f.field_key,f.label,f.field_type,f.required,f.multiple,f.ordinal
 FROM datasets d JOIN projects p ON p.id=d.project_id,
 (VALUES ('canonical_key','记录键','string',true,false,0),('number','番号','string',false,false,1),
 ('title','标题','string',false,false,2),('links','磁力链接','string',false,true,3),
 ('actress','演员','string',false,false,4))
 AS f(field_key,label,field_type,required,multiple,ordinal)
 WHERE p.code='default' AND d.code='magnet' AND d.schema_version=2
 ON CONFLICT (dataset_id,schema_version,field_key) DO NOTHING;
INSERT INTO dataset_schema_versions (dataset_id,version,unique_key_fields,empty_value_policy)
 SELECT d.id,2,'["canonical_key"]'::jsonb,d.empty_value_policy FROM datasets d JOIN projects p ON p.id=d.project_id
 WHERE p.code='default' AND d.code='magnet' AND d.schema_version=2
 ON CONFLICT (dataset_id,version) DO NOTHING;

CREATE TEMP TABLE legacy_record_candidates ON COMMIT DROP AS
 SELECT r.id AS legacy_resource_id,d.id AS dataset_id,r.source_id,r.source_url,r.status,
 r.first_seen_at,r.last_seen_at,
 trim(regexp_replace(r.canonical_key,'#duplicate:[0-9]+$','','i')) AS canonical_key,
 jsonb_strip_nulls(jsonb_build_object(
  'canonical_key',trim(regexp_replace(r.canonical_key,'#duplicate:[0-9]+$','','i')),
  'number',CASE WHEN r.canonical_key ~* '^https?://' OR r.canonical_key ~* '^legacy-id:' THEN NULL
    ELSE trim(regexp_replace(r.canonical_key,'#duplicate:[0-9]+$','','i')) END,
  'title',NULLIF(trim(r.title),''),
  'actress',NULLIF(trim(r.attributes->>'actress'),''),
  'links',COALESCE((SELECT jsonb_agg(l.link ORDER BY l.priority,l.id) FROM resource_links l WHERE l.resource_id=r.id),'[]'::jsonb)
 )) AS payload
 FROM resources r JOIN datasets d ON d.record_type='magnet' AND d.code='magnet'
 JOIN projects p ON p.id=d.project_id AND p.code='default'
 WHERE r.resource_type='magnet';
CREATE UNIQUE INDEX ON legacy_record_candidates(legacy_resource_id);
CREATE INDEX ON legacy_record_candidates(dataset_id,canonical_key);
INSERT INTO record_migration_issues (legacy_resource_id,reason)
 SELECT legacy_resource_id,'empty canonical key' FROM legacy_record_candidates WHERE canonical_key=''
 ON CONFLICT (legacy_resource_id) DO NOTHING;

WITH representative AS (
 SELECT DISTINCT ON (dataset_id,canonical_key) dataset_id,canonical_key,
  jsonb_set(c.payload,'{links}',COALESCE((
   SELECT jsonb_agg(DISTINCT link.value)
   FROM legacy_record_candidates x CROSS JOIN LATERAL jsonb_array_elements_text(x.payload->'links') AS link(value)
   WHERE x.dataset_id=c.dataset_id AND x.canonical_key=c.canonical_key
  ),'[]'::jsonb)) AS payload
 FROM legacy_record_candidates c WHERE canonical_key<>''
 ORDER BY dataset_id,canonical_key,last_seen_at DESC,legacy_resource_id DESC
), bounds AS (
 SELECT dataset_id,canonical_key,min(first_seen_at) AS first_seen_at,max(last_seen_at) AS last_seen_at
 FROM legacy_record_candidates WHERE canonical_key<>'' GROUP BY dataset_id,canonical_key
)
INSERT INTO records (dataset_id,canonical_key,normalized,content_hash,first_seen_at,last_seen_at,created_at,updated_at)
 SELECT r.dataset_id,r.canonical_key,r.payload,encode(digest(r.payload::text,'sha256'),'hex'),
 b.first_seen_at,b.last_seen_at,b.first_seen_at,b.last_seen_at
 FROM representative r JOIN bounds b USING(dataset_id,canonical_key) WHERE true
 ON CONFLICT (dataset_id,canonical_key) DO NOTHING;

INSERT INTO record_observations
 (record_id,dataset_id,schema_version,source_id,source_url,raw_fields,normalized_fields,decision,legacy_resource_id,observed_at)
 SELECT rec.id,c.dataset_id,2,c.source_id,c.source_url,
  jsonb_build_object('legacy_resource_id',c.legacy_resource_id,'status',c.status,'source_url',c.source_url,'fields',c.payload),c.payload,
  CASE WHEN c.legacy_resource_id = (SELECT min(x.legacy_resource_id) FROM legacy_record_candidates x
      WHERE x.dataset_id=c.dataset_id AND x.canonical_key=c.canonical_key) THEN 'created' ELSE 'unchanged' END,
  c.legacy_resource_id,c.last_seen_at
 FROM legacy_record_candidates c JOIN records rec ON rec.dataset_id=c.dataset_id AND rec.canonical_key=c.canonical_key
 WHERE c.canonical_key<>'' AND NOT EXISTS (SELECT 1 FROM legacy_resource_records m WHERE m.legacy_resource_id=c.legacy_resource_id);
INSERT INTO legacy_resource_records (legacy_resource_id,record_id,observation_id)
 SELECT DISTINCT ON (o.legacy_resource_id) o.legacy_resource_id,o.record_id,o.id FROM record_observations o
 WHERE o.legacy_resource_id IS NOT NULL ORDER BY o.legacy_resource_id,o.id DESC
 ON CONFLICT (legacy_resource_id) DO NOTHING;
INSERT INTO record_revisions (record_id,observation_id,before_fields,after_fields,changed_fields,created_at)
 SELECT r.id,o.id,'{}'::jsonb,r.normalized,
  COALESCE((SELECT jsonb_agg(key) FROM jsonb_object_keys(r.normalized) key),'[]'::jsonb),r.created_at
 FROM records r JOIN record_observations o ON o.record_id=r.id AND o.decision='created' AND o.legacy_resource_id IS NOT NULL
 WHERE NOT EXISTS (SELECT 1 FROM record_revisions existing WHERE existing.record_id=r.id)
 ON CONFLICT (observation_id) DO NOTHING;
`)
	if err != nil {
		return err
	}
	return s.Commit()
}
