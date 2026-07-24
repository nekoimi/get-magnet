package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/repo/job_run_repo"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type CronScheduler interface {
	bean.Lifecycle
	// Register 注册定时任务
	Register(spec string, job *CronJob)
	Snapshot() []Snapshot
}

type CronJobScheduler struct {
	// 定时任务
	cron  *cron.Cron
	mu    sync.RWMutex
	tasks []registeredJob
}

type registeredJob struct {
	ID   cron.EntryID
	Name string
	Spec string
}

type Snapshot struct {
	Name     string     `json:"name"`
	Cron     string     `json:"cron"`
	NextRun  *time.Time `json:"next_run,omitempty"`
	LastRun  *time.Time `json:"last_run,omitempty"`
	Status   string     `json:"status,omitempty"`
	Duration int64      `json:"duration_ms,omitempty"`
	Error    string     `json:"error,omitempty"`
}

func NewCronScheduler() CronScheduler {
	return &CronJobScheduler{
		cron:  cron.New(),
		tasks: make([]registeredJob, 0),
	}
}

func (c *CronJobScheduler) Name() string {
	return "CronScheduler"
}

func (c *CronJobScheduler) Start(ctx context.Context) error {
	log.Infoln("Start CronScheduler...")
	c.cron.Start()
	return nil
}

func (c *CronJobScheduler) Register(spec string, job *CronJob) {
	wrapped := cron.FuncJob(func() {
		startedAt := time.Now()
		run := &table.JobRun{JobName: job.Name, Status: "success", StartedAt: startedAt}
		defer func() {
			run.FinishedAt = time.Now()
			run.DurationMs = run.FinishedAt.Sub(startedAt).Milliseconds()
			if recovered := recover(); recovered != nil {
				run.Status = "failed"
				run.Error = fmt.Sprint(recovered)
				log.Errorf("执行Job[%s] panic: %v", job.Name, recovered)
			}
			job_run_repo.Record(run)
		}()
		log.Infof("执行Job[%s]...", job.Name)
		job.Exec()
		log.Infof("执行Job[%s]完成", job.Name)
	})
	id, err := c.cron.AddJob(spec, wrapped)
	if err != nil {
		log.Errorf("注册Job[%s]出现异常：%s", job.Name, err.Error())
		return
	}
	c.mu.Lock()
	c.tasks = append(c.tasks, registeredJob{ID: id, Name: job.Name, Spec: spec})
	c.mu.Unlock()
	log.Infof("注册Job[%s]完成", job.Name)
}

func (c *CronJobScheduler) Snapshot() []Snapshot {
	c.mu.RLock()
	tasks := append([]registeredJob(nil), c.tasks...)
	c.mu.RUnlock()
	names := make([]string, 0, len(tasks))
	for _, task := range tasks {
		names = append(names, task.Name)
	}
	latest, err := job_run_repo.LatestByJobNames(names)
	if err != nil {
		log.Errorf("查询调度任务最近执行结果异常: %s", err.Error())
		latest = map[string]table.JobRun{}
	}
	result := make([]Snapshot, 0, len(tasks))
	for _, task := range tasks {
		entry := c.cron.Entry(task.ID)
		item := Snapshot{Name: task.Name, Cron: task.Spec}
		if !entry.Next.IsZero() {
			next := entry.Next
			item.NextRun = &next
		}
		if run, ok := latest[task.Name]; ok {
			last := run.StartedAt
			item.LastRun = &last
			item.Status = run.Status
			item.Duration = run.DurationMs
			item.Error = run.Error
		}
		result = append(result, item)
	}
	return result
}

func (c *CronJobScheduler) Stop(ctx context.Context) error {
	<-c.cron.Stop().Done()
	return nil
}
