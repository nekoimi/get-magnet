package dashboard

import (
	"net/http"
	"time"

	"github.com/nekoimi/scrapio/internal/db/table"
	"github.com/nekoimi/scrapio/internal/pkg/respond"
	"github.com/nekoimi/scrapio/internal/repo/resource_repo"
)

type StatusCount struct {
	Status string `json:"status"`
	Name   string `json:"name"`
	Count  int64  `json:"count"`
}

type SummaryResponse struct {
	Total        int64         `json:"total"`
	TodayCreated int64         `json:"today_created"`
	StatusCounts []StatusCount `json:"status_counts"`
	GeneratedAt  time.Time     `json:"generated_at"`
}

func Summary(w http.ResponseWriter, r *http.Request) {
	counts, err := resource_repo.CountByStatus()
	if err != nil {
		respond.Error(w, err)
		return
	}
	total, err := resource_repo.CountAll()
	if err != nil {
		respond.Error(w, err)
		return
	}
	todayCreated, err := resource_repo.CountCreatedSince(startOfDay(time.Now()))
	if err != nil {
		respond.Error(w, err)
		return
	}

	statusCounts := make([]StatusCount, 0, len(table.ResourceStatusOptions()))
	for _, opt := range table.ResourceStatusOptions() {
		statusCounts = append(statusCounts, StatusCount{
			Status: opt["value"],
			Name:   opt["label"],
			Count:  counts[opt["value"]],
		})
	}

	respond.Ok(w, SummaryResponse{
		Total:        total,
		TodayCreated: todayCreated,
		StatusCounts: statusCounts,
		GeneratedAt:  time.Now(),
	})
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
