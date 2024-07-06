package main

import (
	"flag"
	"github.com/nekoimi/get-magnet/config"
	"github.com/nekoimi/get-magnet/server"
	"github.com/nekoimi/get-magnet/storage"
	"log"
)

var cfg = config.Config{
	Storage: storage.Db,
}

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	flag.IntVar(&cfg.WorkerNum, "worker", 1, "start worker count")
	flag.StringVar(&cfg.DbDsn, "dsn", "", "db dsn")
	flag.StringVar(&cfg.Jsonrpc, "jsonrpc", "", "aria2 jsonrpc address")
	flag.StringVar(&cfg.Secret, "secret", "", "aria2 jsonrpc secret")
}

func main() {
	flag.Parse()

	srv := server.New(cfg)
	srv.Run(func(s *server.Server) {
		log.Printf("Service is running, listening on port %s\n", ":8080")
	})
}
