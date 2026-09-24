package ops

import (
	"net/http"
	"time"

	"github.com/nekoimi/get-magnet/internal/api/middleware"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/repo/task_repo"
)

func Metrics(engine *crawler.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		tasks, err := task_repo.Metrics()
		if err != nil {
			respond.Error(w, err)
			return
		}
		respond.Ok(w, map[string]any{
			"http":         middleware.HTTPMetricsSnapshot(),
			"tasks":        tasks,
			"crawler":      engine.Snapshot(),
			"generated_at": time.Now(),
		})
	}
}
