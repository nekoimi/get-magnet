package rod_browser

import (
	"context"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"time"
)

type Browser struct {
	ctx context.Context
	// 配置信息
	cfg *config.BrowserConfig
	// 浏览器实例
	browser *rod.Browser
}

func NewRodBrowser() *Browser {
	return &Browser{}
}

func (b *Browser) Name() string {
	return "RodBrowser"
}

func (b *Browser) Start(ctx context.Context) error {
	b.ctx = ctx
	cfg := bean.PtrFromContext[config.Config](ctx)
	b.cfg = cfg.Browser
	proxyEnv := httpproxy.FromEnvironment()
	launchBuilder := launcher.New().
		Headless(b.cfg.Headless).
		Bin(b.cfg.Bin).
		UserDataDir(b.cfg.DataDir).
		NoSandbox(true).
		Set("lang", "zh_cn").
		Set("window-size", "1920,1080").
		Set("disable-setuid-sandbox").
		Set("disable-dev-shm-usage").
		Set("no-first-run").
		Set("disable-blink-features", "AutomationControlled").
		Set("excludeSwitches", "enable-automation")

	if proxyEnv.HTTPProxy != "" {
		launchBuilder.Proxy(proxyEnv.HTTPProxy)
	}

	launch := launchBuilder.MustLaunch()
	b.browser = rod.New().ControlURL(launch).MustConnect()
	// 打开一个持久页面（about:blank），保持浏览器存活
	b.browser.MustPage("about:blank")
	return nil
}

func (b *Browser) NewTabPage() (*rod.Page, func(url string)) {
	// 页面持续操作时间：5分钟
	timeoutCtx, cancelFunc := context.WithTimeout(b.ctx, 5*time.Minute)
	page := stealth.MustPage(b.browser).Context(timeoutCtx)
	page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
			"AppleWebKit/537.36 (KHTML, like Gecko) " +
			"Chrome/120.0.0.0 Safari/537.36",
		Platform: "Windows",
	})
	page.MustSetExtraHeaders("Accept-Language", "zh-CN,zh;q=0.9")
	closeFunc := func(url string) {
		// try close page
		if err := page.Close(); err != nil {
			log.Errorf("关闭标签页异常：%s", err.Error())
		}

		cancelFunc()
		log.Debugf("退出页面 %s 浏览...", url)
	}

	return page, closeFunc
}

func (b *Browser) Stop(ctx context.Context) error {
	if err := b.browser.Close(); err != nil {
		log.Errorf("关闭browser异常：%s", err.Error())
		return err
	}
	log.Debugln("关闭browser...")
	return nil
}
