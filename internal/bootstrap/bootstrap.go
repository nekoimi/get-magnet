package bootstrap

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/core"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/sehuatang"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/downloader"
	"github.com/nekoimi/get-magnet/internal/downloader/aria2_downloader"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	"github.com/nekoimi/get-magnet/internal/server"
)

func BootLifecycle() *core.LifecycleManager {
	ctx := core.ContextWithDefaultRegistry(context.Background())
	lifecycle := core.LifecycleFromContext(ctx)
	// 加载配置
	core.MustRegisterPtr[config.Config](ctx, config.Load())
	// 数据库
	core.MustRegister[core.Lifecycle](ctx, db.NewDBLifecycle())
	// 定时任务
	core.MustRegister[job.CronScheduler](ctx, job.NewCronScheduler())
	// RodBrowser
	core.MustRegisterPtr[rod_browser.Browser](ctx, rod_browser.NewRodBrowser())
	// 下载器
	core.MustRegister[downloader.DownloadService](ctx, aria2_downloader.NewAria2DownloadService())
	// 任务管理器
	crawlerManager := crawler.NewCrawlerManager(ctx)
	crawlerManager.Register(javdb.NewJavDBCrawler(ctx))
	crawlerManager.Register(javdb.NewJavDBActorCrawler(ctx))
	crawlerManager.Register(sehuatang.NewSeHuaTangCrawler(ctx))
	core.MustRegisterPtr[crawler.Manager](ctx, crawlerManager)
	// 任务处理引擎
	core.MustRegisterPtr[crawler.Engine](ctx, crawler.NewCrawlerEngine())
	// http服务
	core.MustRegisterPtr[server.Server](ctx, server.NewHttpServer())
	return lifecycle
}
