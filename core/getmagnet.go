package core

import (
	"github.com/gocolly/colly"
	"log"
)

type GetMagnet struct {
	c            *colly.Collector
	torrentLinks chan string
}

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
}

func New() *GetMagnet {
	return &GetMagnet{
		c: colly.NewCollector(),
	}
}

func (gm *GetMagnet) Run() {

}
