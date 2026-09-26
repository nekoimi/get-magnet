package table

import "time"

type Record struct {
	Id           int64     `json:"id"`
	DatasetId    int64     `xorm:"dataset_id" json:"dataset_id"`
	CanonicalKey string    `xorm:"canonical_key" json:"canonical_key"`
	Normalized   string    `xorm:"jsonb normalized" json:"normalized"`
	Status       string    `json:"status"`
	ContentHash  string    `xorm:"content_hash" json:"content_hash"`
	FirstSeenAt  time.Time `xorm:"first_seen_at" json:"first_seen_at"`
	LastSeenAt   time.Time `xorm:"last_seen_at" json:"last_seen_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Record) TableName() string { return "records" }

type RecordObservation struct {
	Id                int64     `json:"id"`
	RecordId          int64     `xorm:"record_id" json:"record_id"`
	DatasetId         int64     `xorm:"dataset_id" json:"dataset_id"`
	SchemaVersion     int       `xorm:"schema_version" json:"schema_version"`
	WorkflowId        *int64    `xorm:"workflow_id" json:"workflow_id,omitempty"`
	WorkflowVersionId *int64    `xorm:"workflow_version_id" json:"workflow_version_id,omitempty"`
	RunId             *int64    `xorm:"run_id" json:"run_id,omitempty"`
	TaskId            *int64    `xorm:"task_id" json:"task_id,omitempty"`
	DocumentId        *int64    `xorm:"document_id" json:"document_id,omitempty"`
	SourceId          *int64    `xorm:"source_id" json:"source_id,omitempty"`
	SourceURL         string    `xorm:"source_url" json:"source_url"`
	RawFields         string    `xorm:"jsonb raw_fields" json:"raw_fields"`
	NormalizedFields  string    `xorm:"jsonb normalized_fields" json:"normalized_fields"`
	Decision          string    `json:"decision"`
	LegacyResourceId  *int64    `xorm:"legacy_resource_id" json:"legacy_resource_id,omitempty"`
	ObservedAt        time.Time `xorm:"observed_at" json:"observed_at"`
}

func (RecordObservation) TableName() string { return "record_observations" }

type RecordRevision struct {
	Id            int64     `json:"id"`
	RecordId      int64     `xorm:"record_id" json:"record_id"`
	ObservationId int64     `xorm:"observation_id" json:"observation_id"`
	BeforeFields  string    `xorm:"jsonb before_fields" json:"before_fields"`
	AfterFields   string    `xorm:"jsonb after_fields" json:"after_fields"`
	ChangedFields string    `xorm:"jsonb changed_fields" json:"changed_fields"`
	CreatedAt     time.Time `json:"created_at"`
}

func (RecordRevision) TableName() string { return "record_revisions" }
