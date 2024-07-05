package main

import (
	"flag"
	"github.com/nekoimi/get-magnet/admin"
	"github.com/nekoimi/get-magnet/core/engine"
	"github.com/nekoimi/get-magnet/storage"
	"log"
)

var cfg = engine.Config{
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
	//flag.Parse()
	//e := engine.New(&cfg)
	//go e.Run()

	srv := admin.NewServer()

	log.Printf("Service is running, listening on port %s\n", ":8080")

	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
