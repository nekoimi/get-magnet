package dataset_repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/db/table"
)

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)

type FieldInput struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Multiple bool   `json:"multiple"`
}
type SchemaInput struct {
	UniqueKeyFields  []string     `json:"unique_key_fields"`
	EmptyValuePolicy string       `json:"empty_value_policy"`
	Fields           []FieldInput `json:"fields"`
}
type CreateDatasetInput struct {
	ProjectID  int64  `json:"project_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	RecordType string `json:"record_type"`
	SchemaInput
}

func ValidateSchema(input SchemaInput) error {
	if len(input.Fields) == 0 || len(input.Fields) > 128 {
		return errors.New("fields: require 1–128 fields")
	}
	if len(input.UniqueKeyFields) == 0 {
		return errors.New("unique_key_fields: at least one key is required")
	}
	if input.EmptyValuePolicy != "preserve" && input.EmptyValuePolicy != "overwrite" {
		return errors.New("empty_value_policy: must be preserve or overwrite")
	}
	seen := map[string]FieldInput{}
	for i, field := range input.Fields {
		if !codePattern.MatchString(field.Key) {
			return fmt.Errorf("fields[%d].key: invalid field key", i)
		}
		if strings.TrimSpace(field.Label) == "" {
			return fmt.Errorf("fields[%d].label: required", i)
		}
		switch field.Type {
		case "string", "integer", "number", "boolean", "datetime", "url", "json":
		default:
			return fmt.Errorf("fields[%d].type: unsupported type", i)
		}
		if _, exists := seen[field.Key]; exists {
			return fmt.Errorf("fields[%d].key: duplicate", i)
		}
		seen[field.Key] = field
	}
	keys := map[string]bool{}
	for i, key := range input.UniqueKeyFields {
		field, exists := seen[key]
		if !exists || !field.Required || field.Multiple || field.Type == "json" {
			return fmt.Errorf("unique_key_fields[%d]: key must be a required scalar field", i)
		}
		if keys[key] {
			return fmt.Errorf("unique_key_fields[%d]: duplicate", i)
		}
		keys[key] = true
	}
	return nil
}

func CreateProject(code, name, goal, owner string) (*table.Project, error) {
	if !codePattern.MatchString(code) || strings.TrimSpace(name) == "" {
		return nil, errors.New("project code or name is invalid")
	}
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	now := time.Now()
	row := &table.Project{Code: code, Name: strings.TrimSpace(name), Goal: goal, Owner: owner, Status: "active", CreatedAt: now, UpdatedAt: now}
	_, err := db.Instance().InsertOne(row)
	return row, err
}

func ListProjects() ([]table.Project, error) {
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	rows := make([]table.Project, 0)
	err := db.Instance().Asc("id").Find(&rows)
	return rows, err
}

func UpdateProject(id int64, name, goal, owner, status string) (*table.Project, error) {
	if id <= 0 || strings.TrimSpace(name) == "" {
		return nil, errors.New("project id and name are required")
	}
	if status != "active" && status != "paused" && status != "archived" {
		return nil, errors.New("invalid project status")
	}
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	row := new(table.Project)
	has, err := db.Instance().ID(id).Get(row)
	if err != nil || !has {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("project not found")
	}
	row.Name, row.Goal, row.Owner, row.Status, row.UpdatedAt = strings.TrimSpace(name), goal, owner, status, time.Now()
	_, err = db.Instance().ID(id).Cols("name", "goal", "owner", "status", "updated_at").Update(row)
	return row, err
}

func CreateDataset(input CreateDatasetInput) (*table.Dataset, error) {
	if input.EmptyValuePolicy == "" {
		input.EmptyValuePolicy = "preserve"
	}
	if input.ProjectID <= 0 || !codePattern.MatchString(input.Code) || !codePattern.MatchString(input.RecordType) || strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("dataset project, code, name or record type is invalid")
	}
	if err := ValidateSchema(input.SchemaInput); err != nil {
		return nil, err
	}
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	keys, _ := json.Marshal(input.UniqueKeyFields)
	now := time.Now()
	row := &table.Dataset{ProjectId: input.ProjectID, Code: input.Code, Name: strings.TrimSpace(input.Name), RecordType: input.RecordType, SchemaVersion: 1, UniqueKeyFields: string(keys), EmptyValuePolicy: input.EmptyValuePolicy, Status: "active", CreatedAt: now, UpdatedAt: now}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return nil, err
	}
	var project table.Project
	if has, err := s.ID(input.ProjectID).Get(&project); err != nil || !has {
		_ = s.Rollback()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("project not found")
	}
	if _, err := s.InsertOne(row); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if _, err := s.InsertOne(&table.DatasetSchemaVersion{DatasetId: row.Id, Version: 1, UniqueKeyFields: row.UniqueKeyFields, EmptyValuePolicy: row.EmptyValuePolicy, CreatedAt: now}); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if err := insertFields(s, row.Id, 1, input.Fields); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if err := s.Commit(); err != nil {
		return nil, err
	}
	return row, nil
}

func UpdateSchema(id int64, input SchemaInput) (*table.Dataset, error) {
	if id <= 0 {
		return nil, errors.New("dataset id is required")
	}
	if err := ValidateSchema(input); err != nil {
		return nil, err
	}
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return nil, err
	}
	rows, err := s.QueryString("SELECT id FROM datasets WHERE id = ? FOR UPDATE", id)
	if err != nil || len(rows) == 0 {
		_ = s.Rollback()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("dataset not found")
	}
	row := new(table.Dataset)
	if _, err := s.ID(id).Get(row); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	row.SchemaVersion++
	keys, _ := json.Marshal(input.UniqueKeyFields)
	row.UniqueKeyFields, row.EmptyValuePolicy, row.UpdatedAt = string(keys), input.EmptyValuePolicy, time.Now()
	if _, err := s.InsertOne(&table.DatasetSchemaVersion{DatasetId: id, Version: row.SchemaVersion, UniqueKeyFields: row.UniqueKeyFields, EmptyValuePolicy: row.EmptyValuePolicy, CreatedAt: row.UpdatedAt}); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if err := insertFields(s, id, row.SchemaVersion, input.Fields); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if _, err := s.ID(id).Cols("schema_version", "unique_key_fields", "empty_value_policy", "updated_at").Update(row); err != nil {
		_ = s.Rollback()
		return nil, err
	}
	if err := s.Commit(); err != nil {
		return nil, err
	}
	return row, nil
}

func insertFields(s interface{ Insert(...any) (int64, error) }, datasetID int64, version int, fields []FieldInput) error {
	for i, field := range fields {
		_, err := s.Insert(&table.DatasetField{DatasetId: datasetID, SchemaVersion: version, FieldKey: field.Key, Label: strings.TrimSpace(field.Label), FieldType: field.Type, Required: field.Required, Multiple: field.Multiple, Ordinal: i})
		if err != nil {
			return err
		}
	}
	return nil
}

func ListDatasets(projectID int64) ([]table.Dataset, error) {
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	rows := make([]table.Dataset, 0)
	s := db.Instance().NewSession()
	defer s.Close()
	if projectID > 0 {
		s.Where("project_id = ?", projectID)
	}
	err := s.Asc("id").Find(&rows)
	return rows, err
}

func Detail(id int64, version int) (*table.Dataset, []table.DatasetField, error) {
	if id <= 0 {
		return nil, nil, errors.New("dataset id is required")
	}
	if db.Instance() == nil {
		return nil, nil, errors.New("database is not initialized")
	}
	row := new(table.Dataset)
	has, err := db.Instance().ID(id).Get(row)
	if err != nil || !has {
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, errors.New("dataset not found")
	}
	if version <= 0 {
		version = row.SchemaVersion
	}
	if version > row.SchemaVersion {
		return nil, nil, errors.New("schema version not found")
	}
	if version != row.SchemaVersion {
		var schema table.DatasetSchemaVersion
		has, err := db.Instance().Where("dataset_id = ? AND version = ?", id, version).Get(&schema)
		if err != nil {
			return nil, nil, err
		}
		if !has {
			return nil, nil, errors.New("schema version not found")
		}
		row.UniqueKeyFields, row.EmptyValuePolicy = schema.UniqueKeyFields, schema.EmptyValuePolicy
	}
	row.SchemaVersion = version
	fields := make([]table.DatasetField, 0)
	err = db.Instance().Where("dataset_id = ? AND schema_version = ?", id, version).Asc("ordinal").Find(&fields)
	if err == nil && len(fields) == 0 {
		return nil, nil, errors.New("schema version not found")
	}
	return row, fields, err
}
