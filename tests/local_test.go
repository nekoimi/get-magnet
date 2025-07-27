package tests

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/core"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/sehuatang"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/downloader/aria2_downloader"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/logger"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	"github.com/nekoimi/get-magnet/internal/server"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
	"time"
)

func Test_Run(t *testing.T) {
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")

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

	cfg := config.Load()

	logger.Initialize(cfg.LogLevel, cfg.LogDir)
	log.Infof("配置信息：\n%s", cfg)

	ctx := context.Background()
	lc := core.NewLifecycleManager(ctx)
	// 注册数据库管理
	lc.Register(db.NewDBLifecycle(cfg.DB))
	// 定时任务
	cronScheduler := job.NewCronScheduler()
	lc.Register(cronScheduler)
	// RodBrowser
	browser := rod_browser.NewRodBrowser(cfg.Browser)
	lc.Register(browser)
	// 下载器
	downloadService := aria2_downloader.NewAria2DownloadService(cfg.Aria2, cronScheduler)
	lc.Register(downloadService)
	// 任务管理器
	crawlerManager := crawler.NewCrawlerManager(cronScheduler)
	crawlerManager.Register(javdb.NewJavDBCrawler(cfg.JavDB, browser))
	crawlerManager.Register(javdb.NewJavDBActorCrawler(cfg.JavDB, browser))
	crawlerManager.Register(sehuatang.NewSeHuaTangCrawler(browser))
	// 任务处理引擎
	engine := crawler.NewCrawlerEngine(cfg.Crawler, downloadService, crawlerManager)
	lc.Register(engine)
	// http服务
	httpServer := server.NewHttpServer(cfg)
	lc.Register(httpServer)

	time.AfterFunc(30*time.Second, func() {
		t.Log("提交测试任务...")
		bus.Event().Publish(bus.SubmitJavDB.Topic(), "https://javdb.com/login")
	})

	// StartAll and Waiting
	lc.StartAndWait()
}
