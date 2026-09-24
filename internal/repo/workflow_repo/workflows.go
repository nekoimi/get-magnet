package workflow_repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/repo/resource_repo"
	"github.com/nekoimi/get-magnet/internal/workflow"
	"xorm.io/xorm"
)

const (
	VersionDraft     = "draft"
	VersionPublished = "published"
	VersionRetired   = "retired"
)

type WorkflowFilter struct {
	SourceID *int64
	Enabled  *bool
	Page     int
	Size     int
}

type CreateWorkflowInput struct {
	SourceID     *int64
	Source       string
	SourceName   string
	Code         string
	Name         string
	ResourceType string
	Definition   string
	CreatedBy    *int64
}

func List(filter WorkflowFilter) ([]table.Workflow, int64, error) {
	if db.Instance() == nil {
		return nil, 0, errors.New("database is not initialized")
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	count := db.Instance().NewSession()
	defer count.Close()
	applyFilter(count, filter)
	total, err := count.Count(new(table.Workflow))
	if err != nil {
		return nil, 0, err
	}
	s := db.Instance().NewSession()
	defer s.Close()
	applyFilter(s, filter)
	rows := make([]table.Workflow, 0)
	err = s.Desc("updated_at").Limit(filter.Size, (filter.Page-1)*filter.Size).Find(&rows)
	return rows, total, err
}

func applyFilter(s *xorm.Session, filter WorkflowFilter) {
	if filter.SourceID != nil {
		s.Where("source_id = ?", *filter.SourceID)
	}
	if filter.Enabled != nil {
		s.And("enabled = ?", *filter.Enabled)
	}
}

// Get returns a workflow without mutating its versions.
func Get(id int64) (*table.Workflow, bool, error) {
	if id <= 0 || db.Instance() == nil {
		return nil, false, errors.New("workflow id is required")
	}
	row := new(table.Workflow)
	has, err := db.Instance().ID(id).Get(row)
	return row, has, err
}

func ListVersions(workflowID int64) ([]table.WorkflowVersion, error) {
	if workflowID <= 0 || db.Instance() == nil {
		return nil, errors.New("workflow id is required")
	}
	rows := make([]table.WorkflowVersion, 0)
	err := db.Instance().Where("workflow_id = ?", workflowID).Desc("version").Find(&rows)
	return rows, err
}

func GetVersion(id int64) (*table.WorkflowVersion, bool, error) {
	if id <= 0 || db.Instance() == nil {
		return nil, false, errors.New("workflow version id is required")
	}
	row := new(table.WorkflowVersion)
	has, err := db.Instance().ID(id).Get(row)
	return row, has, err
}

func Create(input CreateWorkflowInput) (*table.Workflow, *table.WorkflowVersion, error) {
	if db.Instance() == nil {
		return nil, nil, errors.New("database is not initialized")
	}
	input.Code, input.Name = strings.TrimSpace(input.Code), strings.TrimSpace(input.Name)
	if input.Code == "" || input.Name == "" {
		return nil, nil, errors.New("workflow code and name are required")
	}
	if input.ResourceType == "" {
		input.ResourceType = "magnet"
	}
	if strings.TrimSpace(input.Definition) == "" {
		return nil, nil, errors.New("workflow definition is required")
	}
	if _, err := workflow.ParseDefinition(input.Definition); err != nil {
		return nil, nil, err
	}
	sourceID, err := resource_repo.ResolveSourceID(input.SourceID, input.Source, input.SourceName)
	if err != nil {
		return nil, nil, err
	}
	row := &table.Workflow{SourceId: sourceID, Code: input.Code, Name: input.Name, ResourceType: input.ResourceType, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	version := &table.WorkflowVersion{Version: 1, Status: VersionDraft, Definition: input.Definition, CreatedBy: input.CreatedBy, CreatedAt: time.Now()}
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return nil, nil, err
	}
	if _, err := s.Insert(row); err != nil {
		_ = s.Rollback()
		return nil, nil, err
	}
	version.WorkflowId = row.Id
	if _, err := s.Insert(version); err != nil {
		_ = s.Rollback()
		return nil, nil, err
	}
	if err := s.Commit(); err != nil {
		return nil, nil, err
	}
	return row, version, nil
}

func CreateDraft(workflowID int64, definition string, createdBy *int64) (*table.WorkflowVersion, error) {
	if _, has, err := Get(workflowID); err != nil || !has {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("workflow not found")
	}
	if _, err := workflow.ParseDefinition(definition); err != nil {
		return nil, err
	}
	var latest struct {
		Version int `xorm:"version"`
	}
	if _, err := db.Instance().Table(new(table.WorkflowVersion)).Select("COALESCE(MAX(version), 0) AS version").Where("workflow_id = ?", workflowID).Get(&latest); err != nil {
		return nil, err
	}
	version := &table.WorkflowVersion{WorkflowId: workflowID, Version: latest.Version + 1, Status: VersionDraft, Definition: definition, CreatedBy: createdBy, CreatedAt: time.Now()}
	_, err := db.Instance().InsertOne(version)
	return version, err
}

func ValidateVersion(id int64) error {
	version, has, err := GetVersion(id)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("workflow version not found")
	}
	_, err = workflow.ParseDefinition(version.Definition)
	return err
}

func PublishVersion(id int64) error {
	version, has, err := GetVersion(id)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("workflow version not found")
	}
	if err := ValidateVersion(id); err != nil {
		return err
	}
	now := time.Now()
	s := db.Instance().NewSession()
	defer s.Close()
	if err := s.Begin(); err != nil {
		return err
	}
	if _, err := s.Where("workflow_id = ? AND status = ?", version.WorkflowId, VersionPublished).Cols("status").Update(&table.WorkflowVersion{Status: VersionRetired}); err != nil {
		_ = s.Rollback()
		return err
	}
	if _, err := s.ID(id).Cols("status", "published_at").Update(&table.WorkflowVersion{Status: VersionPublished, PublishedAt: &now}); err != nil {
		_ = s.Rollback()
		return err
	}
	if _, err := s.ID(version.WorkflowId).Cols("published_version_id", "updated_at", "enabled").Update(&table.Workflow{PublishedVersionId: &id, UpdatedAt: now, Enabled: true}); err != nil {
		_ = s.Rollback()
		return err
	}
	return s.Commit()
}

func RollbackVersion(id int64) error { return PublishVersion(id) }

func Stop(id int64) error {
	if _, has, err := Get(id); err != nil || !has {
		if err != nil {
			return err
		}
		return errors.New("workflow not found")
	}
	_, err := db.Instance().ID(id).Cols("enabled", "updated_at").Update(&table.Workflow{Enabled: false, UpdatedAt: time.Now()})
	return err
}

func DiffVersions(leftID, rightID int64) (map[string]any, error) {
	left, has, err := GetVersion(leftID)
	if err != nil || !has {
		return nil, fmt.Errorf("left workflow version not found")
	}
	right, has, err := GetVersion(rightID)
	if err != nil || !has {
		return nil, fmt.Errorf("right workflow version not found")
	}
	var leftValue, rightValue any
	if err := json.Unmarshal([]byte(left.Definition), &leftValue); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(right.Definition), &rightValue); err != nil {
		return nil, err
	}
	return map[string]any{"left_version_id": leftID, "right_version_id": rightID, "changed": !jsonEqual(leftValue, rightValue), "left": leftValue, "right": rightValue}, nil
}

func jsonEqual(left, right any) bool {
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return string(a) == string(b)
}
