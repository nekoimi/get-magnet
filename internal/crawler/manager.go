package crawler

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/job"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	// context
	ctx context.Context
	// crawler集合
	crawlers map[string]Crawler
	// 定时任务调度
	cronScheduler job.CronScheduler
}

func NewCrawlerManager(ctx context.Context, cronScheduler job.CronScheduler) *Manager {
	return &Manager{
		ctx:           ctx,
		crawlers:      make(map[string]Crawler, 0),
		cronScheduler: cronScheduler,
	}
}

func (m *Manager) Register(crawler Crawler) {
	m.crawlers[crawler.Name()] = crawler
}

func (m *Manager) RunAll() {
	for _, crawler := range m.crawlers {
		go func(c Crawler) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("执行Crawler[%s] panic: %v", c.Name(), r)
				}
			}()

			c.Run(m.ctx)
		}(crawler)
	}
}

func (m *Manager) ScheduleAll() {
	for _, crawler := range m.crawlers {
		m.cronScheduler.Register(crawler.CronSpec(), &job.CronJob{
			Name: crawler.Name(),
			Exec: func() {
				crawler.Run(m.ctx)
			},
		})
	}
}
