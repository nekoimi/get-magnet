package rod_browser

import (
	"context"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/core"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"time"
)

type Config struct {
	// Rod启动路径
	Bin string `json:"bin,omitempty" mapstructure:"bin"`
	// Rod调试模式
	Headless bool `json:"headless,omitempty" mapstructure:"headless"`
	// Rod浏览器数据存储目录
	DataDir string `json:"data_dir,omitempty" mapstructure:"data_dir"`
}

type Browser struct {
	// 配置信息
	cfg *Config
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
	cfg := core.PtrFromContext[config.Config](ctx)
	b.cfg = cfg.Browser
	proxyEnv := httpproxy.FromEnvironment()
	launchBuilder := launcher.New().
		Headless(b.cfg.Headless).
		Bin(b.cfg.Bin).
		UserDataDir(b.cfg.DataDir).
		Set("lang", "zh-CN")

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
	timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Minute)
	page := stealth.MustPage(b.browser).Context(timeoutCtx)
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
