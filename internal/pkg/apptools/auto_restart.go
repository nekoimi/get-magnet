package apptools

import (
	log "github.com/sirupsen/logrus"
	"runtime/debug"
	"time"
)

func AutoRestart(name string, runFunc func(), delay time.Duration) {
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("启动服务[%s] panic: %v\n%s", name, r, debug.Stack())
					}
				}()

				log.Debugf("[%s] 启动服务...", name)
				runFunc()
			}()

			log.Debugf("[%s] %v 后将尝试重新启动...", name, delay)
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
				log.Errorf("延迟执行[%s] panic: %v\n%s", name, r, debug.Stack())
			}
		}()

		log.Debugf("延迟执行[%s]...\n", name)
		runFunc()
	}()
}
