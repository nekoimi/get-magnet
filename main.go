package main

import (
	"flag"
	"github.com/nekoimi/get-magnet/engine"
	"github.com/nekoimi/get-magnet/handlers/javdb"
	"github.com/nekoimi/get-magnet/internal/task"
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
	flag.Parse()
	e := engine.New(&cfg)

	// e.Submit(task.NewTask("https://javdb.com/censored?vft=2&vst=2", javdb.ChineseSubtitlesMovieList))
	e.CronSubmit("00 2 * * *", task.NewTask("https://javdb.com/censored?vft=2&vst=2", javdb.ChineseSubtitlesMovieList))

	e.Run()
}
