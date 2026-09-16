package migrate

import (
	"github.com/nekoimi/get-magnet/internal/db/table"
	"xorm.io/xorm"
)

type jobRuns struct{}

func init() {
	registerMigrate(new(jobRuns))
}

func (m *jobRuns) Version() int64 { return 2026_07_24_003 }
func (m *jobRuns) Desc() string   { return "新增调度任务执行记录表" }

func (m *jobRuns) Exec(e *xorm.Engine) error {
	if err := AutoCreateTable(e, new(table.JobRun)); err != nil {
		return err
	}
	_, err := e.Exec(`
CREATE INDEX IF NOT EXISTS idx_job_runs_job_started ON job_runs (job_name, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_runs_status ON job_runs (status);
`)
	return err
}
