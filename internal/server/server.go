package server

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
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	shutdown chan struct{}
	http     *http.Server
	starters []core.Starter
}

func NewServer(ctx context.Context, cfg *config.Config) *Server {
	// 初始化数据库
	db.Initialize(ctx, cfg.DB)

	cronScheduler := job.NewCronScheduler()

	browser := rod_browser.NewRodBrowser(ctx, cfg.Browser)

	crawlerManager := crawler.NewCrawlerManager(ctx, cronScheduler)
	crawlerManager.Register(javdb.NewJavDBCrawler(cfg.JavDB, browser))
	crawlerManager.Register(javdb.NewJavDBActorCrawler(cfg.JavDB, browser))
	crawlerManager.Register(sehuatang.NewSeHuaTangCrawler(browser))

	downloadService := aria2_downloader.NewAria2DownloadService(ctx, cfg.Aria2, cronScheduler)
	engine := crawler.NewCrawlerEngine(ctx, cfg.Crawler, downloadService, crawlerManager)

	s := &Server{
		shutdown: make(chan struct{}),
		http:     HttpServer(cfg.Port, cfg.JwtSecret),
		starters: make([]core.Starter, 0),
	}

	s.AddStarter(cronScheduler)
	s.AddStarter(browser)
	s.AddStarter(engine)

	return s
}

func (s *Server) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	go handleSignal(cancel)

	for _, starter := range s.starters {
		starter.Start(ctx)
	}

	go func() {
		if err := s.http.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-ctx.Done()

	// 创建 shutdown 上下文：最多等待 10 秒退出
	shutdownCtx, shutdownCancel := context.WithTimeout(parent, 10*time.Second)
	defer shutdownCancel()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		log.Errorf("HTTP server Shutdown error: %v", err)
	} else {
		log.Infoln("HTTP server 已优雅退出")
	}
}

func (s *Server) AddStarter(starter core.Starter) {
	s.starters = append(s.starters, starter)
}

// 捕获信号并取消上下文
func handleSignal(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh
	log.Infof("收到退出信号: %v，正在关闭...", sig)
	cancel()
}
