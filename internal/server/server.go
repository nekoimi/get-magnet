package server

import (
	"context"
	"fmt"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/pkg/jwt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Server struct {
	// 配置信息
	cfg  *config.Config
	http *http.Server
}

func NewHttpServer(cfg *config.Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Name() string {
	return "HttpServer"
}

func (s *Server) Start(ctx context.Context) error {
	jwt.SetSecret(s.cfg.JwtSecret)

	router := newRouter()

	s.http = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: router,
	}

	if err := s.http.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe error: %v", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	// 创建 shutdown 上下文：最多等待 10 秒退出
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		log.Errorf("HTTP server Shutdown error: %v", err)
		return err
	} else {
		log.Infoln("HTTP server 已优雅退出")
	}
	return nil
}
