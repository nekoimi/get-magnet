package crawler

import (
	"context"
	"fmt"

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

type ProviderSnapshot struct {
	Name     string `json:"name"`
	CronSpec string `json:"cron"`
	Enabled  bool   `json:"enabled"`
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

func (m *Manager) Run(name string) error {
	if name == "" {
		m.RunAll()
		return nil
	}
	for _, provider := range m.crawlers {
		if provider.Name() == name {
			go safeRun(provider)
			return nil
		}
	}
	return fmt.Errorf("采集源不存在: %s", name)
}

func (m *Manager) Providers() []ProviderSnapshot {
	result := make([]ProviderSnapshot, 0, len(m.crawlers))
	for _, provider := range m.crawlers {
		result = append(result, ProviderSnapshot{
			Name: provider.Name(), CronSpec: provider.CronSpec(), Enabled: true,
		})
	}
	return result
}

func safeRun(provider Crawler) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Errorf("执行Crawler[%s] panic: %v", provider.Name(), recovered)
		}
	}()
	provider.Run()
}

func (m *Manager) ScheduleAll() {
	for _, crawler := range m.crawlers {
		m.cronScheduler.Register(crawler.CronSpec(), &job.CronJob{
			Name: crawler.Name(),
			Exec: crawler.Run,
		})
	}
}
