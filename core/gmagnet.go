package core

import "github.com/gocolly/colly"

type Gmagent struct {
	c            *colly.Collector
	torrentLinks chan string
}

type GetMagnet struct {
}
