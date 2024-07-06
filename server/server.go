package server

import (
	"github.com/nekoimi/get-magnet/config"
	"github.com/nekoimi/get-magnet/engine"
	"github.com/nekoimi/get-magnet/router"
	"net/http"
)

type Server struct {
	http   *http.Server
	engine *engine.Engine
}

func New(cfg config.Config) *Server {
	s := &Server{
		http: &http.Server{
			Addr:    ":8080",
			Handler: router.New(),
		},
		engine: engine.New(cfg),
	}

	return s
}

func (s *Server) Run(func(s *Server)) {
	go s.engine.Run()

	err := s.http.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
