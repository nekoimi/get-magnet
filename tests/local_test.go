package tests

import (
	"github.com/nekoimi/get-magnet/internal/bootstrap"
	"os"
	"testing"
)

func Test_Run(t *testing.T) {
	//os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	//os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")

	os.Setenv("PORT", "11235")
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("LOG_DIR", "logs")
	os.Setenv("JWT_SECRET", "xxxxxxx")
	os.Setenv("ARIA2_JSONRPC", "http://127.0.0.1:6800/jsonrpc")
	os.Setenv("ARIA2_SECRET", "123456")
	os.Setenv("ARIA2_MOVE_TO_JAVDB_DIR", "/tmp")
	os.Setenv("CRAWLER_EXEC_ON_STARTUP", "false")
	os.Setenv("CRAWLER_WORKER_NUM", "8")
	os.Setenv("DB_DSN", "postgres://devtest:devtest@10.1.1.100:5432/get_magnet_dev?sslmode=disable")
	os.Setenv("CRAWLER_DRISSION_ROD_GRPC_IP", "10.1.1.113")
	os.Setenv("CRAWLER_DRISSION_ROD_GRPC_PORT", "8191")

	// 初始化服务
	lifecycle := bootstrap.BeanLifecycle()
	// 启动服务
	lifecycle.StartAndServe()

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

	//ctx := bean.ContextWithDefaultRegistry(context.Background())
	//// 加载配置
	//bean.MustRegisterPtr[config.Config](ctx, config.Load())
	//// RodBrowser
	//browser := rod_browser.NewRodBrowser()
	//bean.MustRegisterPtr[rod_browser.Browser](ctx, browser)
	//browser.Start(ctx)
	//
	////rawUrl := "https://rucaptcha.com/42"
	//rawUrl := "https://mvnrepository.com/"
	////rawUrl := "https://javdb.com/censored?vft=2&vst=1"
	//
	//downloader := download.NewRodBrowserDownloader(browser)
	//_, err := downloader.Download(rawUrl)
	//if err != nil {
	//	panic(err)
	//}

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

	//select {}
}
