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
)

type Server struct {
	cfg    *config.Config
	http   *http.Server
	cron   *cron.Cron
	engine *engine.Engine
}

func New(cfg *config.Config) *Server {
	database.Init(cfg.DB)

	s := &Server{
		cfg: cfg,
		http: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router.New(),
		},
		cron:   cron.New(),
		engine: engine.New(cfg.Engine),
	}

	return s
}

func (s *Server) Run() {
	// s.cron.Run()
	// s.engine.Run()

	log.Printf("Service is running, listening on port %s\n", fmt.Sprintf(":%d", s.cfg.Port))

	err := s.http.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (s *Server) Stop() {
	<-s.cron.Stop().Done()
	s.engine.Stop()
	_ = database.Get().Close()
	_ = s.http.Shutdown(context.Background())
}
