package record_repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/db/table"
	"github.com/nekoimi/scrapio/internal/record"
)

// ImportLegacyResource lets the old magnet writer converge on the new model.
// The legacy resource is retained as the compatibility read path.
func ImportLegacyResource(id int64) (Result, error) {
	candidate, err := legacyCandidate(id)
	if err != nil {
		return Result{}, err
	}
	candidate.IdempotencyKey = "legacy-resource:" + strconv.FormatInt(id, 10) + ":" + record.Hash(candidate.Values)
	candidate.LegacyResourceID = &id
	return Save(candidate)
}

func ObserveLegacyWorkflowResource(id, workflowVersionID, runID, taskID, documentID int64) (Result, error) {
	candidate, err := legacyCandidate(id)
	if err != nil {
		return Result{}, err
	}
	candidate.WorkflowVersionID = &workflowVersionID
	candidate.RunID = &runID
	candidate.TaskID = &taskID
	if documentID > 0 {
		candidate.DocumentID = &documentID
	}
	candidate.IdempotencyKey = "workflow-task:" + strconv.FormatInt(taskID, 10)
	return Save(candidate)
}

func legacyCandidate(id int64) (Candidate, error) {
	if db.Instance() == nil {
		return Candidate{}, errors.New("database is not initialized")
	}
	resource := new(table.Resource)
	has, err := db.Instance().ID(id).Get(resource)
	if err != nil {
		return Candidate{}, err
	}
	if !has {
		return Candidate{}, fmt.Errorf("resource %d not found", id)
	}
	if resource.ResourceType != "magnet" {
		return Candidate{}, errors.New("only magnet resources have a legacy adapter")
	}
	rows, err := db.Instance().QueryString("SELECT d.id FROM datasets d JOIN projects p ON p.id = d.project_id WHERE p.code = ? AND d.code = ?", "default", "magnet")
	if err != nil {
		return Candidate{}, err
	}
	if len(rows) == 0 {
		return Candidate{}, errors.New("default magnet dataset not found")
	}
	datasetID, err := strconv.ParseInt(rows[0]["id"], 10, 64)
	if err != nil {
		return Candidate{}, err
	}
	values := map[string]any{"canonical_key": record.LegacyMagnetKey(strings.TrimSpace(resource.CanonicalKey))}
	if !strings.HasPrefix(strings.ToLower(resource.CanonicalKey), "http") && !strings.HasPrefix(resource.CanonicalKey, "legacy-id:") {
		values["number"] = values["canonical_key"]
	}
	if resource.Title != "" {
		values["title"] = resource.Title
	}
	var attributes map[string]any
	if json.Unmarshal([]byte(resource.Attributes), &attributes) == nil {
		if actress, ok := attributes["actress"].(string); ok && actress != "" {
			values["actress"] = actress
		}
	}
	var links []table.ResourceLink
	if err := db.Instance().Where("resource_id = ?", id).Asc("priority").Find(&links); err != nil {
		return Candidate{}, err
	}
	list := make([]any, 0, len(links))
	for _, link := range links {
		if link.Link != "" {
			list = append(list, link.Link)
		}
	}
	values["links"] = list
	return Candidate{DatasetID: datasetID, Values: values, SourceID: &resource.SourceId, SourceURL: resource.SourceURL}, nil
}

type ReconcileReport struct {
	NextID    int64            `json:"next_id"`
	Processed int              `json:"processed"`
	Succeeded int              `json:"succeeded"`
	Failures  map[int64]string `json:"failures"`
}

// ReconcileLegacy can be called repeatedly after a partial old-path write.
func ReconcileLegacy(afterID int64, limit int) (ReconcileReport, error) {
	if db.Instance() == nil {
		return ReconcileReport{}, errors.New("database is not initialized")
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	var rows []table.Resource
	if err := db.Instance().Where("resource_type = ? AND id > ?", "magnet", afterID).Asc("id").Limit(limit).Find(&rows); err != nil {
		return ReconcileReport{}, err
	}
	report := ReconcileReport{NextID: afterID, Failures: map[int64]string{}}
	for _, row := range rows {
		report.NextID = row.Id
		report.Processed++
		if _, err := ImportLegacyResource(row.Id); err != nil {
			report.Failures[row.Id] = err.Error()
			if _, writeErr := db.Instance().Exec("INSERT INTO record_migration_issues (legacy_resource_id,reason) VALUES (?,?) ON CONFLICT (legacy_resource_id) DO UPDATE SET reason = EXCLUDED.reason", row.Id, err.Error()); writeErr != nil {
				return report, writeErr
			}
		} else {
			report.Succeeded++
			if _, writeErr := db.Instance().Exec("DELETE FROM record_migration_issues WHERE legacy_resource_id = ?", row.Id); writeErr != nil {
				return report, writeErr
			}
		}
	}
	return report, nil
}
