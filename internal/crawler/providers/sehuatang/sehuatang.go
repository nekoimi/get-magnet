package sehuatang

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/drission_rod"
)

const Name = "SeHuaTang"
const FC2PPV = "FC2PPV"

type Crawler struct {
	Parser
}

func NewSeHuaTangCrawler() crawler.BuilderFunc {
	return func(ctx context.Context) crawler.Crawler {
		rod := bean.PtrFromContext[drission_rod.DrissionRod](ctx)
		return &Crawler{Parser{
			downloader: newDrissionRodDownloader(ctx, rod),
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

	//// FC2PPV
	//bus.Event().Publish(bus.SubmitTask.Topic(), crawler.NewCrawlerTask(
	//	"https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=36&filter=typeid&typeid=368",
	//	FC2PPV,
	//	crawler.WithHandle(c.parseList),
	//	crawler.WithDownloader(c.downloader),
	//))
}
