package rod_browser

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
)

// 浏览器实例
var rodBrowserSingleton = singleton.New[*rod.Browser](func() *rod.Browser {
	proxyEnv := httpproxy.FromEnvironment()
	launch := launcher.New().
		Headless(config.Get().RodHeadless).
		Bin(config.Get().RodBin).
		UserDataDir(config.Get().RodDataDir).
		Proxy(proxyEnv.HTTPProxy).
		Set("lang", "zh-CN").
		MustLaunch()
	browser := rod.New().ControlURL(launch).MustConnect()

	return browser
})

func NewTabPage() (*rod.Page, func()) {
	browser := rodBrowserSingleton.Get()
	page := stealth.MustPage(browser)

	closeFunc := func() {
		if err := page.Close(); err != nil {
			log.Errorf("关闭标签页异常：%s", err.Error())
			return
		}
		log.Debugln("退出页面浏览...")
	}

	return page, closeFunc
}

func InitBrowser() {
	browser := rodBrowserSingleton.Get()
	// 打开一个持久页面（about:blank），保持浏览器存活
	browser.MustPage("about:blank")
}

func Close() {
	browser := rodBrowserSingleton.Get()
	if err := browser.Close(); err != nil {
		log.Errorf("关闭browser异常：%s", err.Error())
		return
	}
	log.Debugln("关闭browser...")
}
