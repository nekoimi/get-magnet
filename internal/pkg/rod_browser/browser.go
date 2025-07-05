package rod_browser

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"github.com/nekoimi/get-magnet/internal/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
)

func NewBrowser() (*rod.Page, func()) {
	proxyEnv := httpproxy.FromEnvironment()
	launch := launcher.New().
		Headless(config.Get().RodHeadless).
		Bin(config.Get().RodBin).
		UserDataDir(config.Get().RodDataDir).
		Proxy(proxyEnv.HTTPProxy).
		Set("lang", "zh-CN").
		MustLaunch()
	browser := rod.New().ControlURL(launch).MustConnect()
	page := stealth.MustPage(browser)

	closeFunc := func() {
		if err := browser.Close(); err != nil {
			log.Errorf("退出浏览页面异常：%s", err.Error())
			return
		}
		log.Debugln("退出页面浏览...")
	}

	return page, closeFunc
}
