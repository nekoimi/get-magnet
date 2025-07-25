package server

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/sehuatang"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/downloader/aria2_downloader"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	"github.com/nekoimi/get-magnet/internal/router"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	shutdown chan struct{}
	cfg      *config.Config
	http     *http.Server
}

func Default(cfg *config.Config) *Server {
	ctx := context.TODO()

	cronScheduler := job.NewCronScheduler(ctx)

	downloadService := aria2_downloader.NewAria2DownloadService(ctx, &aria2_downloader.Aria2Config{
		JsonRpc: cfg.Aria2Ops.JsonRpc,
		Secret:  cfg.Aria2Ops.Secret,
	}, cronScheduler)

	cm := crawler.NewCrawlerManager(ctx, cronScheduler)

	cm.Register(javdb.NewJavDBCrawler())
	cm.Register(javdb.NewJavDBActorCrawler())
	cm.Register(sehuatang.NewSeHuaTangCrawler())

	e := crawler.NewCrawlerEngine(ctx, &crawler.EngineConfig{
		ExecOnStartup: false,
		WorkerNum:     0,
		OcrBin:        "",
	}, downloadService, cm)

	cronScheduler.Start()
	e.Run()

	s := &Server{
		shutdown: make(chan struct{}),
		cfg:      cfg,
		http:     router.HttpServer(cfg.Port),
	}

	return s
}

func (s *Server) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	go handleSignal(cancel)

	// 初始化数据库
	db.Init(s.cfg.DB)

	rod_browser.InitBrowser()

	go func() {
		if err := s.http.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-ctx.Done()

	s.stop()
}

func (s *Server) stop() {
	rod_browser.Close()
	_ = db.Instance().Close()

	// 创建 shutdown 上下文：最多等待 10 秒退出
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		log.Errorf("HTTP server Shutdown error: %v", err)
	} else {
		log.Debugln("HTTP server 已优雅退出")
	}
}

// 捕获信号并取消上下文
func handleSignal(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh
	log.Infof("收到退出信号: %v，正在关闭...", sig)
	cancel()
}
