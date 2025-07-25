package job

import (
	"context"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type CronScheduler interface {
	// Start 启动定时任务调度
	Start()

	// Register 注册定时任务
	Register(spec string, job *CronJob)
}

type CronJobScheduler struct {
	// context
	ctx context.Context
	// 定时任务
	cron *cron.Cron
}

func NewCronScheduler(ctx context.Context) CronScheduler {
	return &CronJobScheduler{
		ctx:  ctx,
		cron: cron.New(),
	}
}

func (c *CronJobScheduler) Start() {
	c.cron.Start()

	go func() {
		select {
		case <-c.ctx.Done():
			c.Close()
			log.Infoln("Stop CronScheduler...")
			return
		}
	}()
	log.Infoln("Start CronScheduler...")
}

func (c *CronJobScheduler) Register(spec string, job *CronJob) {
	_, err := c.cron.AddJob(spec, job)
	if err != nil {
		log.Errorf("注册Job[%s]出现异常：%s", job.Name, err.Error())
		return
	}
	log.Infof("注册Job[%s]完成", job.Name)
}

func (c *CronJobScheduler) Close() error {
	<-c.cron.Stop().Done()
	return nil
}
