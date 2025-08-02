package main

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/core"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/sehuatang"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/downloader/aria2_downloader"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	"github.com/nekoimi/get-magnet/internal/server"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// 定时任务
	cronScheduler := job.NewCronScheduler()
	// RodBrowser
	browser := rod_browser.NewRodBrowser(cfg.Browser)
	// 下载器
	downloadService := aria2_downloader.NewAria2DownloadService(cfg.Aria2, cronScheduler)
	// 任务管理器
	crawlerManager := crawler.NewCrawlerManager(cronScheduler)
	crawlerManager.Register(javdb.NewJavDBCrawler(cfg.JavDB, browser))
	crawlerManager.Register(javdb.NewJavDBActorCrawler(cfg.JavDB, browser))
	crawlerManager.Register(sehuatang.NewSeHuaTangCrawler(browser))
	// 任务处理引擎
	engine := crawler.NewCrawlerEngine(cfg.Crawler, downloadService, crawlerManager)
	// http服务
	httpServer := server.NewHttpServer(cfg)

	lifecycle := core.NewLifecycleManager(ctx)
	lifecycle.Register(db.NewDBLifecycle(cfg.DB))
	lifecycle.Register(cronScheduler)
	lifecycle.Register(browser)
	lifecycle.Register(downloadService)
	lifecycle.Register(engine)
	lifecycle.Register(httpServer)
	lifecycle.StartAndWait()
}
