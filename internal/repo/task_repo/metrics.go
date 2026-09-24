package task_repo

import (
	"errors"

	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
)

type MetricsSnapshot struct {
	RunCounts       map[string]int64 `json:"run_counts"`
	TaskCounts      map[string]int64 `json:"task_counts"`
	AttemptCount    int64            `json:"attempt_count"`
	FailedAttempts  int64            `json:"failed_attempts"`
	AverageDuration int64            `json:"average_duration_ms"`
	SourceCounts    []SourceCount    `json:"source_counts"`
	SourceMetrics   []SourceMetric   `json:"source_metrics"`
}

type SourceCount struct {
	SourceID   int64  `json:"source_id"`
	SourceName string `json:"source_name"`
	Total      int64  `json:"total"`
}

type SourceMetric struct {
	SourceID          int64  `json:"source_id"`
	SourceName        string `json:"source_name"`
	PageAttempts      int64  `json:"page_attempts"`
	Succeeded         int64  `json:"succeeded"`
	Failed            int64  `json:"failed"`
	AverageDurationMs int64  `json:"average_duration_ms"`
}

func Metrics() (MetricsSnapshot, error) {
	if db.Instance() == nil {
		return MetricsSnapshot{}, errors.New("database is not initialized")
	}
	result := MetricsSnapshot{RunCounts: map[string]int64{}, TaskCounts: map[string]int64{}, SourceCounts: []SourceCount{}, SourceMetrics: []SourceMetric{}}
	type countRow struct {
		Status string `xorm:"status"`
		Count  int64  `xorm:"count"`
	}
	var runs []countRow
	if err := db.Instance().Table(new(table.WorkflowRun)).Select("status, COUNT(*) AS count").GroupBy("status").Find(&runs); err != nil {
		return result, err
	}
	for _, row := range runs {
		result.RunCounts[row.Status] = row.Count
	}
	var tasks []countRow
	if err := db.Instance().Table(new(table.CrawlTask)).Select("status, COUNT(*) AS count").GroupBy("status").Find(&tasks); err != nil {
		return result, err
	}
	for _, row := range tasks {
		result.TaskCounts[row.Status] = row.Count
	}
	var attempts struct {
		Count   int64 `xorm:"count"`
		Failed  int64 `xorm:"failed"`
		Average int64 `xorm:"average"`
	}
	if _, err := db.Instance().Table(new(table.TaskAttempt)).Select("COUNT(*) AS count, COUNT(*) FILTER (WHERE status = 'failed') AS failed, COALESCE(AVG(duration_ms), 0)::bigint AS average").Get(&attempts); err != nil {
		return result, err
	}
	result.AttemptCount, result.FailedAttempts, result.AverageDuration = attempts.Count, attempts.Failed, attempts.Average
	if err := db.Instance().Table(new(table.Resource)).Select("resources.source_id, sources.name AS source_name, COUNT(*) AS total").Join("INNER", "sources", "sources.id = resources.source_id").GroupBy("resources.source_id, sources.name").OrderBy("total DESC").Find(&result.SourceCounts); err != nil {
		return result, err
	}
	if err := db.Instance().Table("task_attempts").Select("sources.id AS source_id, sources.name AS source_name, COUNT(*) AS page_attempts, COUNT(*) FILTER (WHERE task_attempts.status = 'succeeded') AS succeeded, COUNT(*) FILTER (WHERE task_attempts.status = 'failed') AS failed, COALESCE(AVG(task_attempts.duration_ms), 0)::bigint AS average_duration_ms").Join("INNER", "crawl_tasks", "crawl_tasks.id = task_attempts.task_id").Join("INNER", "workflow_runs", "workflow_runs.id = crawl_tasks.run_id").Join("INNER", "workflows", "workflows.id = workflow_runs.workflow_id").Join("INNER", "sources", "sources.id = workflows.source_id").Where("crawl_tasks.task_type = ?", "browser").GroupBy("sources.id, sources.name").OrderBy("page_attempts DESC").Find(&result.SourceMetrics); err != nil {
		return result, err
	}
	return result, nil
}
