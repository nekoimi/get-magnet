package cloud_driver

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/downloader/cloud_downloader"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
)

type HealthResponse struct {
	OK        bool      `json:"ok"`
	Message   string    `json:"message,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

func Health(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		err := cloud_downloader.CheckHealth(ctx, cfg.CloudDriver)
		resp := HealthResponse{
			OK:        err == nil,
			CheckedAt: time.Now(),
		}
		if err != nil {
			resp.Message = err.Error()
		}
		respond.Ok(w, resp)
	}
}

func Task(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := mux.Vars(r)["taskID"]
		if taskID == "" {
			respond.Error(w, error_ext.ValidateError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		task, err := cloud_downloader.GetTask(ctx, cfg.CloudDriver, taskID)
		if err != nil {
			respond.Error(w, err)
			return
		}
		respond.Ok(w, task)
	}
}
