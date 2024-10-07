package main

import (
	"flag"
	"github.com/nekoimi/get-magnet/common/task"
	"github.com/nekoimi/get-magnet/core/engine"
	"github.com/nekoimi/get-magnet/providers/javdb"
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

	// 立即执行
	e.Submit(task.NewTask(1, "https://javdb.com/censored?vft=2&vst=2", javdb.ChineseSubtitlesMovieList))
	// 定时执行
	e.CronSubmit("00 2 * * *", task.NewTask(1, "https://javdb.com/censored?vft=2&vst=2", javdb.ChineseSubtitlesMovieList))

	e.Run()
	//
	//app := iris.New()
	//
	//app.Listen(":8080")
}
