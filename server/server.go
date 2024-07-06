package server

import (
	"context"
	"fmt"
	"github.com/nekoimi/get-magnet/config"
	"github.com/nekoimi/get-magnet/database"
	"github.com/nekoimi/get-magnet/engine"
	"github.com/nekoimi/get-magnet/router"
	"github.com/robfig/cron/v3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	signalChan chan os.Signal
	cfg        *config.Config
	http       *http.Server
	cron       *cron.Cron
	engine     *engine.Engine
}

func New(cfg *config.Config) *Server {
	database.Init(cfg.DB)

	s := &Server{
		signalChan: make(chan os.Signal, 1),
		cfg:        cfg,
		http: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router.New(),
		},
		cron:   cron.New(),
		engine: engine.New(cfg.Engine),
	}

	signal.Notify(s.signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	return s
}

func (s *Server) Run() {
	go s.cron.Run()
	go s.engine.Run()

	go func() {
		err := s.http.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	log.Printf("Service is running, listening on port %s\n", fmt.Sprintf(":%d", s.cfg.Port))
	for range s.signalChan {
		s.Stop()
	}
}

func (s *Server) Stop() {
	<-s.cron.Stop().Done()
	s.engine.Stop()
	_ = database.Get().Close()
	_ = s.http.Shutdown(context.Background())
}
