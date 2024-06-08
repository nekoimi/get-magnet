package core

import "github.com/gocolly/colly"

type GetMagnet struct {
	c            *colly.Collector
	torrentLinks chan string
}

func New() *GetMagnet {
	return &GetMagnet{
		c: colly.NewCollector(),
	}
}

func (gm *GetMagnet) Run() {

}
