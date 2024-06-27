package main

import (
	"get-magnet/engine"
	"get-magnet/handlers/javdb"
	"get-magnet/internal/task"
	"get-magnet/storage"
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
	e := engine.New(1, storage.Db)

	e.Submit(task.NewTask("https://javdb.com/censored?vft=2&vst=2", javdb.ChineseSubtitlesMovieList))
	//e.CronSubmit("00 3 */3 * *", task.NewTask("https://javdb.com/censored?vft=2&vst=2", javdb.MovieDetails))

	// e.Submit(task.NewTask("https://movie.douban.com/top250", douban.Top250List))

	e.Run()
}
