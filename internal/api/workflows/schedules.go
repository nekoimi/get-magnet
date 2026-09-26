package workflows

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/nekoimi/scrapio/internal/api/middleware"
	"github.com/nekoimi/scrapio/internal/pkg/error_ext"
	"github.com/nekoimi/scrapio/internal/pkg/request"
	"github.com/nekoimi/scrapio/internal/pkg/respond"
	"github.com/nekoimi/scrapio/internal/repo/audit_repo"
	"github.com/nekoimi/scrapio/internal/repo/workflow_repo"
)

type scheduleRequest struct {
	WorkflowID        int64  `json:"workflow_id"`
	Cron              string `json:"cron"`
	Timezone          string `json:"timezone"`
	Enabled           bool   `json:"enabled"`
	ConcurrencyPolicy string `json:"concurrency_policy"`
}

func ScheduleDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("workflow_id"), 10, 64)
	if err != nil || id <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	row, has, err := workflow_repo.GetSchedule(id)
	if err != nil {
		respond.Error(w, err)
		return
	}
	events, err := workflow_repo.ListScheduleEvents(id)
	if err != nil {
		respond.Error(w, err)
		return
	}
	if !has {
		row = nil
	}
	respond.Ok(w, map[string]any{"schedule": row, "events": events})
}

func ScheduleSave(w http.ResponseWriter, r *http.Request) {
	var input scheduleRequest
	if err := request.Parse(r, &input); err != nil || input.WorkflowID <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	row, err := workflow_repo.SaveSchedule(workflow_repo.Schedule{WorkflowID: input.WorkflowID, Cron: input.Cron, Timezone: input.Timezone, Enabled: input.Enabled, ConcurrencyPolicy: input.ConcurrencyPolicy})
	if err != nil {
		respond.Error(w, err)
		return
	}
	_ = audit_repo.Record(middleware.RequestID(r.Context()), "workflow.schedule_saved", "workflow", &input.WorkflowID, map[string]any{"enabled": input.Enabled, "cron": input.Cron, "timezone": input.Timezone, "policy": input.ConcurrencyPolicy})
	respond.Ok(w, row)
}

// APITrigger deliberately has no caller-controlled URL, page state or secret.
// The published workflow definition is the only source of fetch credentials.
func APITrigger(w http.ResponseWriter, r *http.Request) {
	id, err := parseAPITrigger(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		respond.Error(w, err)
		return
	}
	run, task, err := workflow_repo.StartRunWithTrigger(id, "{}", nil, "api")
	if err != nil {
		respond.Error(w, err)
		return
	}
	_ = audit_repo.Record(middleware.RequestID(r.Context()), "workflow.api_triggered", "workflow_run", &run.Id, map[string]any{"workflow_id": id, "task_id": task.Id})
	respond.Ok(w, map[string]any{"run_id": run.Id, "task_id": task.Id, "workflow_version_id": run.WorkflowVersionId})
}

func parseAPITrigger(body io.Reader) (int64, error) {
	var payload map[string]json.RawMessage
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&payload); err != nil || payload == nil {
		return 0, error_ext.ValidateError
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return 0, error_ext.ValidateError
	}
	for key := range payload {
		if key != "workflow_id" && key != "input" {
			return 0, errors.New("unsupported API trigger field: " + key)
		}
	}
	var id int64
	if err := json.Unmarshal(payload["workflow_id"], &id); err != nil || id <= 0 {
		return 0, error_ext.ValidateError
	}
	if len(payload["input"]) > 0 {
		var input map[string]json.RawMessage
		if err := json.Unmarshal(payload["input"], &input); err != nil || input == nil || len(input) > 0 {
			return 0, errors.New("API trigger input overrides are not configured")
		}
	}
	return id, nil
}
