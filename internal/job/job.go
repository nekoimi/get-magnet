package job

import (
	log "github.com/sirupsen/logrus"
)

type CronJob struct {
	// 任务名称
	Name string
	// 任务执行逻辑
	Exec func()
}

func (job *CronJob) Run() {
	log.Infof("执行Job[%s]...", job.Name)
	func(c *CronJob) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("执行Job[%s] panic: %v", c.Name, r)
			}
		}()

		c.Exec()
	}(job)
	log.Infof("执行Job[%s]完成", job.Name)
}
