package tests

import (
	"context"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	"os"
	"testing"
	"time"
)

func Test_Run(t *testing.T) {
	//os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	//os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")

	os.Setenv("PORT", "11234")
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("LOG_DIR", "logs")
	os.Setenv("JWT_SECRET", "xxxxxxx")
	os.Setenv("BROWSER_BIN", "")
	os.Setenv("BROWSER_HEADLESS", "false")
	os.Setenv("BROWSER_DATA_DIR", "C:\\Users\\nekoimi\\Downloads\\rod-data")
	os.Setenv("ARIA2_JSONRPC", "http://127.0.0.1:6800/jsonrpc")
	os.Setenv("ARIA2_SECRET", "123456")
	os.Setenv("ARIA2_MOVE_TO_JAVDB_DIR", "/tmp")
	os.Setenv("CRAWLER_EXEC_ON_STARTUP", "false")
	os.Setenv("CRAWLER_WORKER_NUM", "8")
	os.Setenv("CRAWLER_OCR_BIN", "C:\\Users\\nekoimi\\Downloads\\x86_64-pc-windows-msvc-inline\\ddddocr.exe")
	os.Setenv("JAVDB_USERNAME", "111111111111")
	os.Setenv("JAVDB_PASSWORD", "222222222222")
	os.Setenv("DB_DSN", "postgres://devtest:devtest@10.1.1.100:5432/get_magnet_dev?sslmode=disable")

	//// 初始化服务
	//lifecycle := bootstrap.BeanLifecycle()
	//// 启动服务
	//lifecycle.StartAndServe()

	//cfg := config.Load()
	//
	//logger.Initialize(cfg.LogLevel, cfg.LogDir)
	//log.Infof("配置信息：\n%s", cfg)
	//
	//ctx := context.Background()
	//lc := core.NewLifecycleManager(ctx)
	//// 注册数据库管理
	//lc.Register(db.NewDBLifecycle(cfg.DB))
	//// 定时任务
	//cronScheduler := job.NewCronScheduler()
	//lc.Register(cronScheduler)
	//// RodBrowser
	//browser := rod_browser.NewRodBrowser(cfg.Browser)
	//browser.Start(ctx)
	////lc.Register(browser)
	////// 下载器
	////downloadService := aria2_downloader.NewAria2DownloadService(cfg.Aria2, cronScheduler)
	////lc.Register(downloadService)
	////// 任务管理器
	////crawlerManager := crawler.NewCrawlerManager(cronScheduler)
	////crawlerManager.Register(javdb.NewJavDBCrawler(cfg.JavDB, browser))
	////crawlerManager.Register(javdb.NewJavDBActorCrawler(cfg.JavDB, browser))
	////crawlerManager.Register(sehuatang.NewSeHuaTangCrawler(browser))
	////// 任务处理引擎
	////engine := crawler.NewCrawlerEngine(cfg.Crawler, downloadService, crawlerManager)
	////lc.Register(engine)
	////// http服务
	////httpServer := server.NewHttpServer(cfg)
	////lc.Register(httpServer)
	////
	////time.AfterFunc(30*time.Second, func() {
	////	t.Log("提交测试任务...")
	////	bus.Event().Publish(bus.SubmitJavDB.Topic(), "https://javdb.com/censored?vft=2&vst=1")
	////})
	////
	////// StartAll and Waiting
	////lc.StartAndWait()
	//

	ctx := bean.ContextWithDefaultRegistry(context.Background())
	// 加载配置
	bean.MustRegisterPtr[config.Config](ctx, config.Load())
	// RodBrowser
	browser := rod_browser.NewRodBrowser()
	bean.MustRegisterPtr[rod_browser.Browser](ctx, browser)
	browser.Start(ctx)

	//rawUrl := "https://rucaptcha.com/42"
	rawUrl := "https://mvnrepository.com/"
	//rawUrl := "https://javdb.com/censored?vft=2&vst=1"

	downloader := download.NewRodBrowserDownloader(browser, "http://10.1.1.100:8191/v1")
	_, err := downloader.Download(rawUrl)
	if err != nil {
		panic(err)
	}

	//page, closeFunc := browser.NewTabPage()
	//defer closeFunc(rawUrl)
	//page.MustNavigate(rawUrl)
	//// 等待页面加载
	//log.Debugf("等待页面 %s 加载...", rawUrl)
	//err := page.WaitLoad()
	//if err != nil {
	//	panic(err)
	//}
	//
	//// 截图，识别点击框的位置
	//page.MustScreenshot("logs/1.png")
	//
	//log.Debugf("页面 %s 加载完毕...", rawUrl)

	select {}
}

func Test_Click(t *testing.T) {
	u := launcher.New().
		Headless(false).
		NoSandbox(true).
		Proxy("http://127.0.0.1:12080").
		Set("disable-web-security").
		Set("disable-site-isolation-trials").
		RemoteDebuggingPort(9222).
		Set("no-first-run").
		Set("no-default-browser-check").
		Set("enable-privacy-sandbox-ads-apis", "false").
		Leakless(true).
		MustLaunch()

	page := rod.New().ControlURL(u).MustConnect().NoDefaultDevice().MustPage("https://dash.cloudflare.com/sign-up")
	el, err := page.Timeout(30 * time.Second).Element(`div.cf-turnstile-wrapper`)
	if err != nil {
		panic(err)
	}

	<-time.After(time.Second * 10)

	res, err := el.Eval(`() => {
const rect = this.getBoundingClientRect();
    return { x: rect.left, y: rect.top };
}`)
	if err != nil {
		panic(err)
	}

	x, y := res.Value.Get("x").Num(), res.Value.Get("y").Num()

	fmt.Println(x, y)

	page.Mouse.MoveLinear(proto.NewPoint(x+27, y+29), 20)

	// 使用JavaScript在鼠标位置添加一个视觉标记
	page.Eval(`(x, y) => {
        const marker = document.createElement('div');
        marker.style.position = 'fixed';  // 使用fixed定位确保可见
        marker.style.left = x + 'px';
        marker.style.top = y + 'px';
        marker.style.width = '10px';
        marker.style.height = '10px';
        marker.style.backgroundColor = 'red';
        marker.style.borderRadius = '50%';
        marker.style.zIndex = 9999;  // 确保在最上层
        document.body.appendChild(marker);
    }`, x+27, y+29)

	page.Mouse.MustClick(proto.InputMouseButtonLeft)
	fmt.Println("click.")
}
