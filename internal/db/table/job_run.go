package table

import "time"

type JobRun struct {
	Id         int64     `json:"id"`
	JobName    string    `xorm:"index notnull" json:"job_name"`
	Status     string    `xorm:"index notnull" json:"status"`
	StartedAt  time.Time `xorm:"index" json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	DurationMs int64     `json:"duration_ms"`
	Error      string    `xorm:"text" json:"error,omitempty"`
}

func (JobRun) TableName() string {
	return "job_runs"
}
