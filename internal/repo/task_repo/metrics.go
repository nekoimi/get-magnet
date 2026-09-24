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
}

func Metrics() (MetricsSnapshot, error) {
	if db.Instance() == nil {
		return MetricsSnapshot{}, errors.New("database is not initialized")
	}
	result := MetricsSnapshot{RunCounts: map[string]int64{}, TaskCounts: map[string]int64{}}
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
	return result, nil
}
