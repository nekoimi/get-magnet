package main

import (
	"fmt"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"log"
	"strings"
)

const JavdbRootDomain = "https://javdb.com"

func main() {
	c := colly.NewCollector()

	// Rotate two socks5 proxies
	rp, err := proxy.RoundRobinProxySwitcher("socks5://127.0.0.1:2080")
	if err != nil {
		log.Fatal(err)
	}
	c.SetProxyFunc(rp)

	//c.OnRequest(func(r *colly.Request) {
	//	fmt.Println("Visiting", r.URL)
	//})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Err:", err)
	})

	//c.OnResponse(func(r *colly.Response) {
	//	fmt.Println("Visited", r.Request.URL)
	//})

	//// 获取下一页
	//c.OnHTML("a.pagination-next", func(e *colly.HTMLElement) {
	//	nextUrl := JAVDB_ROOT_DOMAIN + e.Attr("href")
	//	fmt.Println("Next: ", nextUrl)
	//	// e.Request.Visit(nextUrl)
	//})

	// 获取列表
	c.OnHTML(".movie-list>div>a.box", func(e *colly.HTMLElement) {
		pageUrl := JavdbRootDomain + e.Attr("href")
		fmt.Println(e.Attr("title"), pageUrl)
		e.Request.Visit(pageUrl)
	})

	// 获取详情
	c.OnHTML("#magnets-content>.item>div>a", func(e *colly.HTMLElement) {
		torrentUrl := e.Attr("href")
		aLinkText := strings.ReplaceAll(strings.ReplaceAll(e.Text, "\n", " "), "  ", " ")
		if strings.Contains(aLinkText, "高清") && strings.Contains(aLinkText, "字幕") {
			fmt.Println("高清字幕: ", torrentUrl)
		} else {
			fmt.Println("非高清字幕: ", torrentUrl, " ", aLinkText)
		}
	})

	//c.Visit("https://javdb.com/censored?vft=2&vst=2")
	//c.Visit("https://javdb.com/censored?page=2&vft=2&vst=2")
	//c.Visit("https://javdb.com/censored?page=3&vft=2&vst=2")
	//c.Visit("https://javdb.com/censored?page=4&vft=2&vst=2")
	//c.Visit("https://javdb.com/censored?page=5&vft=2&vst=2")
	//c.Visit("https://javdb.com/censored?page=6&vft=2&vst=2")
	//c.Visit("https://javdb.com/censored?page=7&vft=2&vst=2")
}
