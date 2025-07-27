package job

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/core"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type CronScheduler interface {
	core.Lifecycle
	// Register 注册定时任务
	Register(spec string, job *CronJob)
}

type CronJobScheduler struct {
	// 定时任务
	cron *cron.Cron
}

func NewCronScheduler() CronScheduler {
	return &CronJobScheduler{
		cron: cron.New(),
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
	_, err := c.cron.AddJob(spec, job)
	if err != nil {
		log.Errorf("注册Job[%s]出现异常：%s", job.Name, err.Error())
		return
	}
	log.Infof("注册Job[%s]完成", job.Name)
}

func (c *CronJobScheduler) Stop(ctx context.Context) error {
	<-c.cron.Stop().Done()
	return nil
}
