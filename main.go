package main

import (
	"get-magnet/engine"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	//// TODO Set temporary environment variables
	//_ = os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:2080")
	//_ = os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:2080")
}

func main() {
	e := engine.Default()

	//e.Submit(task.NewTask("https://javdb.com/censored?vft=2&vst=2", javdb.ParseMovieList))
	//e.CronSubmit("00 3 */3 * *", task.NewTask("https://javdb.com/censored?vft=2&vst=2", javdb.ParseMovieList))

	// e.Submit(task.NewTask("https://movie.douban.com/top250", douban.Top250List))

	e.Run()
}
