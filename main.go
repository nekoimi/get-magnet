package main

import (
	"get-magnet/engine"
	"get-magnet/handlers/javdb"
	"get-magnet/scheduler"
	"log"
	"os"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	// tmp env
	_ = os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:2080")
	_ = os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:2080")
}

func main() {
	e := engine.Default()

	e.Cron.AddFunc("*/1 * * * *", func() {
		e.Scheduler.Submit(scheduler.Task{
			Url:    "https://javdb.com/censored?vft=2&vst=2",
			Handle: javdb.ParseMovieList,
		})
	})

	e.RunWait()
}
