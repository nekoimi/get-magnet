package job

import (
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type Job struct {
	// 任务名称
	Name string
	// 任务执行逻辑
	Cmd func()
}

func (job *Job) Run() {
	log.Infof("执行Job[%s]...", job.Name)
	func(j *Job) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("执行Job[%s] panic: %v", j.Name, r)
			}
		}()

		j.Cmd()
	}(job)
	log.Infof("执行Job[%s]完成", job.Name)
}

var (
	crontab = singleton.New[*cron.Cron](func() *cron.Cron {
		return cron.New()
	})
)

func Start() {
	crontab.Get().Start()
}

func Register(spec string, job *Job) {
	_, err := crontab.Get().AddJob(spec, job)
	if err != nil {
		log.Warnf("注册Job[%s]出现异常：%s", job.Name, err.Error())
		return
	}
	log.Debugf("注册Job[%s]完成", job.Name)
}

func Stop() {
	ctx := crontab.Get().Stop()
	<-ctx.Done()
}
