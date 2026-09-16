package job_run_repo

import (
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	log "github.com/sirupsen/logrus"
)

func Record(run *table.JobRun) {
	if run == nil || db.Instance() == nil {
		return
	}
	if _, err := db.Instance().InsertOne(run); err != nil {
		log.Errorf("记录调度任务执行结果异常: %s - %s", run.JobName, err.Error())
	}
}

func LatestByJobNames(names []string) (map[string]table.JobRun, error) {
	result := make(map[string]table.JobRun)
	if len(names) == 0 || db.Instance() == nil {
		return result, nil
	}
	var rows []table.JobRun
	err := db.Instance().In("job_name", names).Desc("started_at").Find(&rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, exists := result[row.JobName]; !exists {
			result[row.JobName] = row
		}
	}
	return result, nil
}
