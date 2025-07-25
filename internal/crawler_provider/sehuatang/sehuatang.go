package sehuatang

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler"
)

const Name = "SeHuaTang"
const FC2PPV = "FC2PPV"

type Crawler struct {
	Parser
}

func NewSeHuaTangCrawler() crawler.Crawler {
	return &Crawler{Parser{
		downloader: GetBypassDownloader(),
	}}
}

func (c *Crawler) Name() string {
	return Name
}

func (c *Crawler) CronSpec() string {
	return "50 3 * * *"
}

func (c *Crawler) Run(ctx context.Context) {
	bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
		"https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=1",
		Name,
		crawler.WithHandle(c.parseList),
		crawler.WithDownloader(c.downloader),
	))

	// FC2PPV
	bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
		"https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=36&filter=typeid&typeid=368",
		FC2PPV,
		crawler.WithHandle(c.parseList),
		crawler.WithDownloader(c.downloader),
	))
}
