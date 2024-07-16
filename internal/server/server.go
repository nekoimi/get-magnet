package server

import (
	"fmt"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/database"
	"github.com/nekoimi/get-magnet/internal/engine"
	"github.com/nekoimi/get-magnet/internal/router"
	"github.com/robfig/cron/v3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	signalChan chan os.Signal
	http       *http.Server
	cron       *cron.Cron
	engine     *engine.Engine
}

func New(cfg *config.Config) *Server {
	database.Init(cfg.DB)

	s := &Server{
		signalChan: make(chan os.Signal, 1),
		http: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router.New(),
		},
		cron:   cron.New(),
		engine: engine.New(),
	}

	signal.Notify(s.signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	return s
}

func (s *Server) Run() {
	go s.cron.Run()
	// go s.engine.Run()

	go func() {
		err := s.http.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	log.Println("Service is running")

	for range s.signalChan {
		s.Stop()
	}
}

func (s *Server) Stop() {
	<-s.cron.Stop().Done()
	//s.engine.Stop()
	_ = database.Instance().Close()
	os.Exit(0)
}