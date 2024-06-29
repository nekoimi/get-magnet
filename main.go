package main

import (
	"github.com/nekoimi/get-magnet/engine"
	"github.com/nekoimi/get-magnet/handlers/javdb"
	"github.com/nekoimi/get-magnet/internal/task"
	"github.com/nekoimi/get-magnet/storage"
	"log"
	"os"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	// TODO Set temporary environment variables
	_ = os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:2080")
	_ = os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:2080")
}

func main() {
	e := engine.New(&engine.Config{
		WorkerNum: 1,
		DbDsn:     "root:mysql#123456@(10.1.1.100:3306)/get_magnet_db",
		AriaRpc:   "wss://aria2.sakuraio.com/jsonrpc",
		AriaToken: "nekoimi",
		Storage:   storage.Db,
	})

	e.Submit(task.NewTask("https://javdb.com/censored?vft=2&vst=2", javdb.ChineseSubtitlesMovieList))
	// e.CronSubmit("00 2 * * *", task.NewTask("https://javdb.com/censored?vft=2&vst=1", javdb.MovieDetails))

	// e.Submit(task.NewTask("https://movie.douban.com/top250", douban.Top250List))

	//go func() {
	//	time.Sleep(10 * time.Second)
	//	e.SubmitDownload(&model.Item{
	//		OptimalLink: "magnet:?xt=urn:btih:E1A47C5A4B172768EBA93B9C8CBE3120DDFC4699",
	//		ResHost:     "https://javdb.com",
	//	})
	//}()

	e.Run()
}
