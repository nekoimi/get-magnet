package crawler

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/job"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	// ctx
	ctx context.Context
	// crawler集合
	crawlers []Crawler
	// 定时任务调度
	cronScheduler job.CronScheduler
}

func NewCrawlerManager(ctx context.Context) *Manager {
	cronScheduler := bean.FromContext[job.CronScheduler](ctx)
	return &Manager{
		ctx:           ctx,
		crawlers:      make([]Crawler, 0),
		cronScheduler: cronScheduler,
	}
}

func (m *Manager) Register(f BuilderFunc) {
	m.crawlers = append(m.crawlers, f(m.ctx))
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
