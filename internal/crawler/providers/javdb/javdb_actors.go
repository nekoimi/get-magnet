package javdb

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
)

type ActorCrawler struct {
	Parser
}

func NewJavDBActorCrawler() crawler.BuilderFunc {
	return func(ctx context.Context) crawler.Crawler {
		cfg := bean.PtrFromContext[config.Config](ctx)
		browser := bean.PtrFromContext[rod_browser.Browser](ctx)
		return &ActorCrawler{Parser{
			downloader: newBypassDownloader(cfg.JavDB, browser, cfg.CloudflarePassApi),
		}}
	}
}

func (c *ActorCrawler) Name() string {
	return "JavDB"
}

func (c *ActorCrawler) CronSpec() string {
	return "30 3 * * 0"
}

func (c *ActorCrawler) Run() {
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
