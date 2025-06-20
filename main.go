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
	e.Submit(task.NewTask(1, "https://javdb.com/censored?vft=2&vst=1", javdb.ChineseSubtitlesMovieList))
	// 定时执行
	e.CronSubmit("00 2 * * *", []*task.Task{
		task.NewTask(1, "https://javdb.com/censored?vft=2&vst=1", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/actors/nePY4?t=c&sort_type=0", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/actors/kW6?t=c&sort_type=0", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/actors/0rva?t=c&sort_type=0", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/actors/AOqm?t=c&sort_type=0", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/video_codes/NTR?f=cnsub", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/series/a5b3", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/directors/qG3?f=cnsub", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/makers/OXz?f=cnsub", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/publishers/A60?f=cnsub", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/actors/O2Q30?t=c&sort_type=0", javdb.ChineseSubtitlesMovieList),
		task.NewTask(1, "https://javdb.com/actors/x7wn?t=c&sort_type=0", javdb.ChineseSubtitlesMovieList),
	})

	e.Run()
	//
	//app := iris.New()
	//
	//app.Listen(":8080")
}
