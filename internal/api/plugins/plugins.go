package plugins

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/repo/plugin_repo"
)

type IDRequest struct {
	ID int64 `json:"id"`
}

func List(w http.ResponseWriter, r *http.Request) {
	resourceID, _ := strconv.ParseInt(r.URL.Query().Get("resource_id"), 10, 64)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	rows, total, err := plugin_repo.List(resourceID, r.URL.Query().Get("status"), page, size)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, map[string]any{"list": rows, "total": total})
}

func Cancel(w http.ResponseWriter, r *http.Request) {
	var input IDRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ID <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if err := plugin_repo.Cancel(input.ID); err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, nil)
}

func Retry(w http.ResponseWriter, r *http.Request) {
	var input IDRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ID <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if err := plugin_repo.Retry(input.ID); err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, nil)
}
