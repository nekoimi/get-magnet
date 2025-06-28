package server

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/db"
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
	engine   *crawler.Engine
}

func Default(cfg *config.Config) *Server {
	s := &Server{
		shutdown: make(chan struct{}),
		cfg:      cfg,
		http:     router.HttpServer(cfg.Port),
		engine:   crawler.New(),
	}

	return s
}

func (s *Server) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	go handleSignal(cancel)

	// 初始化数据库
	db.Init(s.cfg.DB)

	go s.engine.Run()

	go func() {
		if err := s.http.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-ctx.Done()

	s.stop()
}

func (s *Server) stop() {
	s.engine.Stop()
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
	log.Infof("收到退出信号: %v，正在关闭...\n", sig)
	cancel()
}
