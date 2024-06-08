package javdb

import (
	"fmt"
	"github.com/gocolly/colly"
	"gmagnet/internal/provider"
	"log"
	"strings"
)

const StartUrl = "https://javdb.com/censored?vft=2&vst=2"

type javDBProvider struct {
	c            *colly.Collector
	detailsColly *colly.Collector
	proxy        colly.ProxyFunc
}

func New(proxy colly.ProxyFunc) provider.MagnetProvider {
	p := &javDBProvider{
		proxy: proxy,
	}
	p.c = colly.NewCollector()
	p.c.SetProxyFunc(p.proxy)
	p.detailsColly = p.c.Clone()
	return p
}

func (p *javDBProvider) RunGet() {
	p.getListPage()
	p.next(StartUrl)
}

// 执行下一页
func (p *javDBProvider) next(url string) {
	if err := p.c.Visit(url); err != nil {
		log.Printf("Visit 列表页URL异常: %s, %s\n", url, err.Error())
	}
}

// 解析列表页
// 再列表页判断是否需要继续下一页
func (p *javDBProvider) getListPage() {
	p.c.OnHTML("", func(e *colly.HTMLElement) {

		// 获取下一页
		p.next("")
	})
}

// 解析详情页
func (p *javDBProvider) getDetails() {
	p.detailsColly.OnHTML("#magnets-content>.item>div>a", func(e *colly.HTMLElement) {
		torrentUrl := e.Attr("href")
		aLinkText := strings.ReplaceAll(strings.ReplaceAll(e.Text, "\n", " "), "  ", " ")
		if strings.Contains(aLinkText, "高清") && strings.Contains(aLinkText, "字幕") {
			fmt.Println("高清字幕: ", torrentUrl)
		} else {
			fmt.Println("非高清字幕: ", torrentUrl, " ", aLinkText)
		}
	})
}
