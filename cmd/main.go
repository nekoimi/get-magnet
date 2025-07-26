package main

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/logger"
	"github.com/nekoimi/get-magnet/internal/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	cfg := config.Load()
	logger.Initialize(cfg.LogLevel, cfg.LogDir)
	log.Infof("配置信息：\n%s", cfg)

	ctx := context.Background()
	s := server.NewServer(ctx, cfg)
	s.Start(ctx)
}
