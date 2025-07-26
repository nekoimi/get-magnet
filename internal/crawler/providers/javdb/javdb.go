package javdb

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	// 账号
	Username string `json:"username,omitempty" mapstructure:"username"`
	// 密码
	Password string `json:"password,omitempty" mapstructure:"password"`
}

type Crawler struct {
	Parser
}

func NewJavDBCrawler(cfg *Config, browser *rod_browser.Browser) crawler.Crawler {
	c := &Crawler{Parser{
		downloader: newBypassDownloader(cfg, browser),
	}}

	// 设置任务监听
	bus.Event().Subscribe(bus.SubmitJavDB.Topic(), func(url string) {
		log.Debugf("接收到JavDB任务：%s", url)
		bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
			url,
			c.Name(),
			crawler.WithHandle(c.parseList),
			crawler.WithDownloader(c.downloader),
		))
	})

	return c
}

func (c *Crawler) Name() string {
	return "JavDB"
}

func (c *Crawler) CronSpec() string {
	return "05 3 * * *"
}

func (c *Crawler) Run(ctx context.Context) {
	bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
		"https://javdb.com/censored?vft=2&vst=1",
		c.Name(),
		crawler.WithHandle(c.parseList),
		crawler.WithDownloader(c.downloader),
	))
}
