package ops

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/nekoimi/get-magnet/internal/api/settings"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/downloader/cloud_downloader"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
)

var startedAt = time.Now()

type ServiceHealth struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}

func Health(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]func(context.Context) error{
			"application": func(context.Context) error { return nil },
			"database":    func(context.Context) error { return db.Instance().Ping() },
			"cloud_driver": func(ctx context.Context) error {
				return cloud_downloader.CheckHealth(ctx, cfg.CloudDriver)
			},
			"drission_rod": func(ctx context.Context) error { return settings.CheckDrissionRod(ctx, cfg.Crawler) },
			"aria2":        func(context.Context) error { return settings.CheckAria2(cfg.Aria2) },
		}
		result := make(map[string]ServiceHealth, len(checks))
		allOK := true
		for name, check := range checks {
			started := time.Now()
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			err := check(ctx)
			cancel()
			item := ServiceHealth{OK: err == nil, LatencyMs: time.Since(started).Milliseconds()}
			if err != nil {
				item.Message = err.Error()
				allOK = false
			}
			result[name] = item
		}
		respond.Ok(w, map[string]any{"ok": allOK, "services": result, "checked_at": time.Now()})
	}
}

func Jobs(scheduler job.CronScheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respond.Ok(w, scheduler.Snapshot())
	}
}

func Version(w http.ResponseWriter, _ *http.Request) {
	version := "dev"
	commit := os.Getenv("GIT_COMMIT")
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && commit == "" {
				commit = setting.Value
			}
		}
	}
	respond.Ok(w, map[string]any{
		"version": version, "commit": commit, "go_version": runtime.Version(),
		"started_at": startedAt, "uptime_seconds": int64(time.Since(startedAt).Seconds()),
	})
}
