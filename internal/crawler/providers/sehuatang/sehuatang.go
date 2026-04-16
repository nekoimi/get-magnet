package sehuatang

import (
	"context"

	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler"
)

const Name = "SeHuaTang"

type Crawler struct {
	Parser
}

func NewSeHuaTangCrawler() crawler.BuilderFunc {
	return func(ctx context.Context) crawler.Crawler {
		return &Crawler{Parser{
			downloader: newDrissionRodDownloader(ctx),
		}}
	}
}

func (c *Crawler) Name() string {
	return Name
}

func (c *Crawler) CronSpec() string {
	return "50 3 * * *"
}

func (c *Crawler) Run() {
	bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
		"https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=1",
		Name,
		crawler.WithHandle(c.parseList),
		crawler.WithDownloader(c.downloader),
	))
}
