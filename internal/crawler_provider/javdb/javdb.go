package javdb

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler"
)

type Crawler struct {
	Parser
}

func NewJavDBCrawler() crawler.Crawler {
	return &Crawler{Parser{
		downloader: GetBypassDownloader(),
	}}
}

func (c *Crawler) Name() string {
	return "JavDB"
}

func (c *Crawler) CronSpec() string {
	return "05 3 * * *"
}

func (c *Crawler) Run(ctx context.Context) {
	bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
		"https://javdb.com/censored?vft=2&vst=1", c.Name(),
		crawler.WithHandle(c.parseList),
		crawler.WithDownloader(c.downloader),
	))
}
