package sehuatang

import (
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"golang.org/x/net/http/httpproxy"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestTaskSeeder(t *testing.T) {
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")

	testUrl := "https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=1"

	proxyEnv := httpproxy.FromEnvironment()
	t.Log(proxyEnv.HTTPProxy)
	t.Log(proxyEnv.HTTPSProxy)
	t.Log(proxyEnv.NoProxy)

	downloader := download.NewHttpDownloader()
	s1, err := downloader.Download(testUrl)
	if err != nil {
		panic(err)
	}
	html1, err := s1.Html()
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("D:\\Developer\\GoProjects\\go-library\\get-magnet\\deploy\\html1.html", []byte(html1), 0666)
	if err != nil {
		panic(err)
	}

	launch := launcher.New().
		Proxy(proxyEnv.HTTPProxy).
		MustLaunch()

	page := rod.New().ControlURL(launch).MustConnect().MustPage(testUrl)
	page.MustWaitStable()
	page.MustScreenshot("a.png")
	mustHTML := page.MustHTML()
	err = os.WriteFile("D:\\Developer\\GoProjects\\go-library\\get-magnet\\deploy\\a.html", []byte(mustHTML), 0666)
	if err != nil {
		panic(err)
	}

	btn := page.MustElementByJS(`() => document.querySelector("body > a:nth-child(5)")`)
	btn.MustClick()
	// 等待跳转或后台处理完成
	page.MustWaitLoad()
	page.MustScreenshot("b.png")
	html := page.MustHTML()
	err = os.WriteFile("D:\\Developer\\GoProjects\\go-library\\get-magnet\\deploy\\b.html", []byte(html), 0666)
	if err != nil {
		panic(err)
	}

	// 可选：保存 cookie
	cookies := page.MustCookies()

	u, err := url.Parse(testUrl)
	if err != nil {
		panic(err)
	}

	var stdCookies []*http.Cookie
	for _, c := range cookies {
		fmt.Printf("Cookie: %s = %s", c.Name, c.Value)
		stdCookies = append(stdCookies, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		})
	}
	// 设置cookies
	downloader.SetCookies(u, stdCookies)

	s2, err := downloader.Download(testUrl)
	if err != nil {
		panic(err)
	}
	html2, err := s2.Html()
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("D:\\Developer\\GoProjects\\go-library\\get-magnet\\deploy\\html2.html", []byte(html2), 0666)
	if err != nil {
		panic(err)
	}

	// 保持几秒观察（调试用）
	time.Sleep(2 * time.Second)
}

func TestUrlDecode(t *testing.T) {
	decodeHref, err := url.QueryUnescape("forum.php%3Fmod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=2")
	if err != nil {
		t.Log(err.Error())
		return
	}
	t.Log(decodeHref)
	t.Log(util.JoinUrl("https://www.sehuatang.net", decodeHref))
}
