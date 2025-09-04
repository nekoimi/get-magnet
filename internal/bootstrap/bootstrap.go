package bootstrap

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
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

func BeanLifecycle() *bean.LifecycleManager {
	ctx := bean.ContextWithDefaultRegistry(context.Background())
	// 加载配置
	bean.MustRegisterPtr[config.Config](ctx, config.Load())
	// 数据库
	bean.MustRegister[bean.Lifecycle](ctx, db.NewDBLifecycle())
	// 定时任务
	bean.MustRegister[job.CronScheduler](ctx, job.NewCronScheduler())
	// RodBrowser
	bean.MustRegisterPtr[rod_browser.Browser](ctx, rod_browser.NewRodBrowser())
	// 下载器
	bean.MustRegister[downloader.DownloadService](ctx, aria2_downloader.NewAria2DownloadService())
	// 任务管理器
	crawlerManager := crawler.NewCrawlerManager(ctx)
	crawlerManager.Register(javdb.NewJavDBCrawler())
	crawlerManager.Register(javdb.NewJavDBActorCrawler())
	crawlerManager.Register(sehuatang.NewSeHuaTangCrawler())
	bean.MustRegisterPtr[crawler.Manager](ctx, crawlerManager)
	// 任务处理引擎
	bean.MustRegisterPtr[crawler.Engine](ctx, crawler.NewCrawlerEngine())
	// http服务
	bean.MustRegisterPtr[server.Server](ctx, server.NewHttpServer())
	return bean.LifecycleFromContext(ctx)
}
