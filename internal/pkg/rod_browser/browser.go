package rod_browser

import (
	"context"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	log "github.com/sirupsen/logrus"
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
	// context
	ctx context.Context
	// 配置信息
	cfg *Config
	// 浏览器实例
	browser *rod.Browser
}

func NewRodBrowser(ctx context.Context, cfg *Config) *Browser {
	return &Browser{
		ctx: ctx,
		cfg: cfg,
	}
}

func (b *Browser) Start(ctx context.Context) {
	b.RunInBackground()
}

func (b *Browser) RunInBackground() {
	launch := launcher.New().
		Headless(b.cfg.Headless).
		Bin(b.cfg.Bin).
		UserDataDir(b.cfg.DataDir).
		Set("lang", "zh-CN").
		MustLaunch()
	b.browser = rod.New().ControlURL(launch).MustConnect()
	// 打开一个持久页面（about:blank），保持浏览器存活
	b.browser.MustPage("about:blank")

	go func() {
		select {
		case <-b.ctx.Done():
			b.Close()
		}
	}()
}

func (b *Browser) NewTabPage() (*rod.Page, func()) {
	page := stealth.MustPage(b.browser)

	closeFunc := func() {
		if err := page.Close(); err != nil {
			log.Errorf("关闭标签页异常：%s", err.Error())
			return
		}
		log.Debugln("退出页面浏览...")
	}

	return page, closeFunc
}

func (b *Browser) Close() error {
	if err := b.browser.Close(); err != nil {
		log.Errorf("关闭browser异常：%s", err.Error())
		return err
	}
	log.Debugln("关闭browser...")
	return nil
}
