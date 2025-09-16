package javdb

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/drission_rod"
	log "github.com/sirupsen/logrus"
)

type Crawler struct {
	Parser
}

func NewJavDBCrawler() crawler.BuilderFunc {
	return func(ctx context.Context) crawler.Crawler {
		rod := bean.PtrFromContext[drission_rod.DrissionRod](ctx)
		c := &Crawler{Parser{
			downloader: newDrissionRodDownloader(ctx, rod),
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
}

func (c *Crawler) Name() string {
	return "JavDB"
}

func (c *Crawler) CronSpec() string {
	return "05 3 * * *"
}

func (c *Crawler) Run() {
	bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
		"https://javdb.com/censored?vft=2&vst=2",
		c.Name(),
		crawler.WithHandle(c.parseList),
		crawler.WithDownloader(c.downloader),
	))
}
