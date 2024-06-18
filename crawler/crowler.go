package crawler

import "github.com/gocolly/colly"

type Crawler struct {
	baseC   *colly.Collector
	baseURL string
}
