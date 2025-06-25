package test

import (
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"golang.org/x/net/http/httpproxy"
	"testing"
	"time"
)

func TestRod(t *testing.T) {
	proxyEnv := httpproxy.FromEnvironment()
	t.Log(proxyEnv.HTTPProxy)
	t.Log(proxyEnv.HTTPSProxy)
	t.Log(proxyEnv.NoProxy)

	u := launcher.New().
		Proxy("http://127.0.0.1:2080").
		MustLaunch()

	page := rod.New().ControlURL(u).MustConnect().MustPage("https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=1")
	page.MustWaitStable()
	page.MustScreenshot("a.png")

	btn := page.MustElementByJS(`() => document.querySelector("body > a:nth-child(5)")`)
	btn.MustClick()
	// 等待跳转或后台处理完成
	page.MustWaitLoad()
	page.MustScreenshot("b.png")

	// 可选：保存 cookie
	cookies := page.MustCookies()
	for _, c := range cookies {
		fmt.Printf("Cookie: %s = %s\n", c.Name, c.Value)
	}

	// 保持几秒观察（调试用）
	time.Sleep(2 * time.Second)
}
