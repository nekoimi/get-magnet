package javdb

import (
	"github.com/gocolly/colly"
	"gmagnet/internal/provider"
)

const StartUrl = "https://javdb.com/censored?vft=2&vst=2"

type javDBProvider struct {
	c       *colly.Collector
	details *colly.Collector
}

func New() provider.MagnetProvider {
	return &javDBProvider{}
}

func (p *javDBProvider) Initiate() {
	p.c = colly.NewCollector()
	p.details = p.c.Clone()
}

func (p *javDBProvider) RunGet() {
	_ = p.c.Visit(StartUrl)
}

// 解析列表页
// 再列表页判断是否需要继续下一页

// 解析详情页
