package main

import (
	"get-magnet/engine"
	"get-magnet/scheduler"
	"get-magnet/test"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	//// tmp env
	//_ = os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:2080")
	//_ = os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:2080")
}

func main() {
	e := engine.Default()

	//e.Submit(scheduler.Task{
	//	Url:    "https://javdb.com/censored?vft=2&vst=2",
	//	Handle: javdb.ParseMovieList,
	//	Meta: &scheduler.TaskMeta{
	//		Host:    "https://javdb.com",
	//		UrlPath: "/censored?vft=2&vst=2",
	//	},
	//})
	//
	//e.CronSubmit("00 3 */3 * *", scheduler.Task{
	//	Url:    "https://javdb.com/censored?vft=2&vst=2",
	//	Handle: javdb.ParseMovieList,
	//	Meta: &scheduler.TaskMeta{
	//		Host:    "https://javdb.com",
	//		UrlPath: "/censored?vft=2&vst=2",
	//	},
	//})

	e.Submit(scheduler.Task{
		Url:    "https://movie.douban.com/top250",
		Handle: test.DouBanTop250List,
		Meta: &scheduler.TaskMeta{
			Host:    "https://movie.douban.com",
			UrlPath: "/top250",
		},
	})

	e.Run()
}
