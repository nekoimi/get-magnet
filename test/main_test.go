package test

import (
	"fmt"
	"get-magnet/internal/storage/file_storage"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/gocolly/colly/proxy"
	"log"
	"net/url"
	"strings"
	"testing"
	"time"
)

const JavdbRootDomain = "https://javdb.com"

func TestFetch(t *testing.T) {
	s := file_storage.New("output")

	c := colly.NewCollector(
		colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)

	// 限制速率
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*javdb.*",
		Parallelism: 2,
		RandomDelay: 3 * time.Second,
	})

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
		if pageUrl, err := url.JoinPath(JavdbRootDomain, e.Attr("href")); err != nil {
			t.Log(err.Error())
		} else {
			fmt.Println(e.Attr("title"), pageUrl)
			e.Request.Visit(pageUrl)
		}
	})

	// 获取详情
	c.OnHTML("#magnets-content>.item>div>a", func(e *colly.HTMLElement) {
		torrentUrl := e.Attr("href")
		aLinkText := strings.ReplaceAll(strings.ReplaceAll(e.Text, "\n", " "), "  ", " ")
		if strings.Contains(aLinkText, "高清") && strings.Contains(aLinkText, "字幕") {
			fmt.Println("高清字幕: ", torrentUrl)
			s.Save(torrentUrl)
		} else {
			fmt.Println("非高清字幕: ", torrentUrl, " ", aLinkText)
		}
	})

	for i := range [11]int{} {
		visitUrl := fmt.Sprintf("https://javdb.com/censored?page=%d&vft=2&vst=2", i+1)
		c.Visit(visitUrl)
	}

	c.Wait()
}
