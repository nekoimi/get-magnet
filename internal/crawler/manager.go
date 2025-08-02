package crawler

import (
	"github.com/nekoimi/get-magnet/internal/job"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	// crawler集合
	crawlers []Crawler
	// 定时任务调度
	cronScheduler job.CronScheduler
}

func NewCrawlerManager(cronScheduler job.CronScheduler) *Manager {
	return &Manager{
		crawlers:      make([]Crawler, 0),
		cronScheduler: cronScheduler,
	}
}

func (m *Manager) Register(crawler Crawler) {
	m.crawlers = append(m.crawlers, crawler)
}

func (m *Manager) RunAll() {
	for _, crawler := range m.crawlers {
		go func(c Crawler) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("执行Crawler[%s] panic: %v", c.Name(), r)
				}
			}()

			c.Run()
		}(crawler)
	}
}

func (m *Manager) ScheduleAll() {
	for _, crawler := range m.crawlers {
		m.cronScheduler.Register(crawler.CronSpec(), &job.CronJob{
			Name: crawler.Name(),
			Exec: crawler.Run,
		})
	}
}
