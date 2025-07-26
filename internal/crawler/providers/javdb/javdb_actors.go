package javdb

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
)

type ActorCrawler struct {
	Parser
}

func NewJavDBActorCrawler(cfg *Config, browser *rod_browser.Browser) crawler.Crawler {
	return &ActorCrawler{Parser{
		downloader: newBypassDownloader(cfg, browser),
	}}
}

func (c *ActorCrawler) Name() string {
	return "JavDB"
}

func (c *ActorCrawler) CronSpec() string {
	return "30 3 * * 0"
}

func (c *ActorCrawler) Run(ctx context.Context) {
	bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
		"https://javdb.com/actors/O2Q30?t=c&sort_type=0",
		c.Name(),
		crawler.WithHandle(c.parseList),
		crawler.WithDownloader(c.downloader),
	))

	bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
		"https://javdb.com/actors/x7wn?t=c&sort_type=0",
		c.Name(),
		crawler.WithHandle(c.parseList),
		crawler.WithDownloader(c.downloader),
	))
}
