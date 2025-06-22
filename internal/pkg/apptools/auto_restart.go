package apptools

import (
	"log"
	"runtime/debug"
	"time"
)

func AutoRestart(name string, runFunc func(), delay time.Duration) {
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[%s] panic: %v\n%s", name, r, debug.Stack())
					}
				}()

				log.Printf("[%s] 启动服务...", name)
				runFunc()
			}()

			log.Printf("[%s] %v 后将尝试重新启动...", name, delay)
			time.Sleep(delay)
		}
	}()
}

func DelayStart(name string, runFunc func(), delay time.Duration) {
	timer := time.NewTimer(delay)
	<-timer.C

	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("延迟执行[%s] panic: %v\n%s", name, r, debug.Stack())
			}
		}()

		log.Printf("延迟执行[%s]...\n", name)
		runFunc()
	}()
}
