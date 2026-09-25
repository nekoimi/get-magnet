package sources

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nekoimi/scrapio/internal/api/middleware"
	"github.com/nekoimi/scrapio/internal/db/table"
	"github.com/nekoimi/scrapio/internal/pkg/error_ext"
	"github.com/nekoimi/scrapio/internal/pkg/request"
	"github.com/nekoimi/scrapio/internal/pkg/respond"
	"github.com/nekoimi/scrapio/internal/repo/audit_repo"
	"github.com/nekoimi/scrapio/internal/repo/resource_repo"
)

type SourceRequest struct {
	ID     int64           `json:"id,omitempty"`
	Code   string          `json:"code,omitempty"`
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config,omitempty"`
}
type ToggleRequest struct {
	ID      int64 `json:"id"`
	Enabled bool  `json:"enabled"`
}

func List(w http.ResponseWriter, _ *http.Request) {
	rows, err := resource_repo.ListAllSources()
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, rows)
}

func Create(w http.ResponseWriter, r *http.Request) {
	input := new(SourceRequest)
	if err := request.Parse(r, input); err != nil {
		respond.Error(w, err)
		return
	}
	source := &table.Source{Code: input.Code, Name: input.Name, Enabled: true, Config: string(input.Config)}
	if err := resource_repo.SaveSource(source); err != nil {
		respond.Error(w, err)
		return
	}
	_ = audit_repo.Record(middleware.RequestID(r.Context()), "source.created", "source", &source.Id, map[string]any{"code": source.Code})
	respond.Ok(w, source)
}

func Update(w http.ResponseWriter, r *http.Request) {
	input := new(SourceRequest)
	if err := request.Parse(r, input); err != nil || input.ID <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	source := &table.Source{Id: input.ID, Name: input.Name, Config: string(input.Config)}
	if err := resource_repo.UpdateSource(source); err != nil {
		respond.Error(w, err)
		return
	}
	_ = audit_repo.Record(middleware.RequestID(r.Context()), "source.updated", "source", &source.Id, nil)
	respond.Ok(w, source)
}

func Toggle(w http.ResponseWriter, r *http.Request) {
	input := new(ToggleRequest)
	if err := request.Parse(r, input); err != nil || input.ID <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if err := resource_repo.SetSourceEnabled(input.ID, input.Enabled); err != nil {
		respond.Error(w, err)
		return
	}
	_ = audit_repo.Record(middleware.RequestID(r.Context()), "source.toggled", "source", &input.ID, map[string]any{"enabled": input.Enabled})
	respond.Ok(w, nil)
}

func ParseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
}
