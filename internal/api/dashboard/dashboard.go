package dashboard

import (
	"net/http"
	"time"

	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
)

type StatusCount struct {
	Status uint8  `json:"status"`
	Name   string `json:"name"`
	Count  int64  `json:"count"`
}

type SummaryResponse struct {
	Total        int64         `json:"total"`
	TodayCreated int64         `json:"today_created"`
	StatusCounts []StatusCount `json:"status_counts"`
	PendingCount int64         `json:"pending_count"`
	Downloading  int64         `json:"downloading"`
	Completed    int64         `json:"completed"`
	Failed       int64         `json:"failed"`
	GeneratedAt  time.Time     `json:"generated_at"`
}

func Summary(w http.ResponseWriter, r *http.Request) {
	counts, err := magnet_repo.CountByStatus()
	if err != nil {
		respond.Error(w, err)
		return
	}
	total, err := magnet_repo.CountAll()
	if err != nil {
		respond.Error(w, err)
		return
	}
	todayCreated, err := magnet_repo.CountCreatedSince(startOfDay(time.Now()))
	if err != nil {
		respond.Error(w, err)
		return
	}

	statusCounts := make([]StatusCount, 0, len(table.MagnetStatusOptions()))
	for _, opt := range table.MagnetStatusOptions() {
		statusCounts = append(statusCounts, StatusCount{
			Status: opt.Value,
			Name:   opt.Label,
			Count:  counts[opt.Value],
		})
	}

	respond.Ok(w, SummaryResponse{
		Total:        total,
		TodayCreated: todayCreated,
		StatusCounts: statusCounts,
		PendingCount: counts[table.MagnetStatusCollected],
		Downloading:  counts[table.MagnetStatusDownloading],
		Completed:    counts[table.MagnetStatusCompleted],
		Failed:       counts[table.MagnetStatusFailed],
		GeneratedAt:  time.Now(),
	})
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
