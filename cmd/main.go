package main

import (
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/server"
)

var cfg = config.Default()

func init() {
	cfg.LoadFromEnv()
}

func main() {
	s := server.Default(cfg)

	s.Run()
}
