package main

import (
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/server"
)

func main() {
	cfg := config.Default()

	s := server.Default(cfg)

	s.Run()
}
