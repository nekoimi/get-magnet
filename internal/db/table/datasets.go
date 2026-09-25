package table

import "time"

type Project struct {
	Id        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Goal      string    `xorm:"text" json:"goal"`
	Owner     string    `json:"owner"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Project) TableName() string { return "projects" }

type Dataset struct {
	Id               int64     `json:"id"`
	ProjectId        int64     `xorm:"project_id" json:"project_id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	RecordType       string    `xorm:"record_type" json:"record_type"`
	SchemaVersion    int       `xorm:"schema_version" json:"schema_version"`
	UniqueKeyFields  string    `xorm:"jsonb unique_key_fields" json:"unique_key_fields"`
	EmptyValuePolicy string    `xorm:"empty_value_policy" json:"empty_value_policy"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Dataset) TableName() string { return "datasets" }

type DatasetField struct {
	Id            int64  `json:"id"`
	DatasetId     int64  `xorm:"dataset_id" json:"dataset_id"`
	SchemaVersion int    `xorm:"schema_version" json:"schema_version"`
	FieldKey      string `xorm:"field_key" json:"field_key"`
	Label         string `json:"label"`
	FieldType     string `xorm:"field_type" json:"field_type"`
	Required      bool   `json:"required"`
	Multiple      bool   `json:"multiple"`
	Ordinal       int    `json:"ordinal"`
}

func (DatasetField) TableName() string { return "dataset_fields" }

type DatasetSchemaVersion struct {
	DatasetId        int64     `xorm:"dataset_id" json:"dataset_id"`
	Version          int       `json:"version"`
	UniqueKeyFields  string    `xorm:"jsonb unique_key_fields" json:"unique_key_fields"`
	EmptyValuePolicy string    `xorm:"empty_value_policy" json:"empty_value_policy"`
	CreatedAt        time.Time `json:"created_at"`
}

func (DatasetSchemaVersion) TableName() string { return "dataset_schema_versions" }
