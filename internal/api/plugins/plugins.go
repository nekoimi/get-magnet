package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/nekoimi/scrapio/internal/pkg/error_ext"
	"github.com/nekoimi/scrapio/internal/pkg/respond"
	pluginruntime "github.com/nekoimi/scrapio/internal/plugin"
	"github.com/nekoimi/scrapio/internal/repo/plugin_repo"
)

type IDRequest struct {
	ID int64 `json:"id"`
}

type PluginStatus struct {
	Code          string   `json:"code"`
	Capabilities  []string `json:"capabilities"`
	Async         bool     `json:"async"`
	WritesBack    bool     `json:"writes_back"`
	Healthy       bool     `json:"healthy"`
	HealthMessage string   `json:"health_message,omitempty"`
	LatencyMs     int64    `json:"latency_ms"`
}

func Overview(registry *pluginruntime.Registry, worker *pluginruntime.Worker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if registry == nil {
			respond.Error(w, errors.New("plugin registry is not initialized"))
			return
		}
		metrics, err := plugin_repo.Metrics()
		if err != nil {
			respond.Error(w, err)
			return
		}
		items := make([]PluginStatus, 0)
		for _, handler := range registry.List() {
			started := time.Now()
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			healthErr := handler.Health(ctx)
			cancel()
			capabilities := append([]string(nil), handler.Capabilities()...)
			sort.Strings(capabilities)
			item := PluginStatus{
				Code: handler.Code(), Capabilities: capabilities,
				Healthy: healthErr == nil, LatencyMs: time.Since(started).Milliseconds(),
			}
			_, item.Async = handler.(pluginruntime.AsyncHandler)
			_, item.WritesBack = handler.(pluginruntime.CompletionHandler)
			if healthErr != nil {
				item.HealthMessage = healthErr.Error()
			}
			items = append(items, item)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
		var workerState any = map[string]any{"running": false, "worker_count": 0, "active": 0}
		if worker != nil {
			workerState = worker.Snapshot()
		}
		respond.Ok(w, map[string]any{
			"plugins": items, "tasks": metrics, "worker": workerState, "generated_at": time.Now(),
		})
	}
}

func List(w http.ResponseWriter, r *http.Request) {
	resourceID, _ := strconv.ParseInt(r.URL.Query().Get("resource_id"), 10, 64)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	rows, total, err := plugin_repo.ListFiltered(plugin_repo.ListFilter{
		ResourceID: resourceID, PluginCode: r.URL.Query().Get("plugin_code"),
		EventType: r.URL.Query().Get("event_type"), Status: r.URL.Query().Get("status"),
		Page: page, Size: size,
	})
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, map[string]any{"list": rows, "total": total})
}

func Detail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if id <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	row, found, err := plugin_repo.Get(id)
	if err != nil {
		respond.Error(w, err)
		return
	}
	if !found {
		respond.Error(w, errors.New("plugin task not found"))
		return
	}
	respond.Ok(w, row)
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
