package apptools

import (
	"context"
	log "github.com/sirupsen/logrus"
	"runtime/debug"
	"time"
)

func AutoRestart(ctx context.Context, name string, runFunc func(ctx context.Context), delay time.Duration) {
	go func() {
		exit := false
		for {
			select {
			case <-ctx.Done():
				exit = true
				return
			default:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Errorf("启动服务[%s] panic: %v", name, r)
							log.Debugf("[%s] %v 后将尝试重新启动...", name, delay)
							select {
							case <-ctx.Done():
								log.Debugf("[%s] 已取消，不再重启", name)
								exit = true
								return
							case <-time.After(delay):
							}
						}

					}()

					log.Debugf("[%s] 启动服务...", name)
					runFunc(ctx)
				}()
			}

			if exit {
				break
			}
		}
	}()
}

func DelayStart(name string, runFunc func(), delay time.Duration) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("延迟执行[%s] panic: %v\n%s", name, r, debug.Stack())
			}
		}()

		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			log.Debugf("延迟执行[%s]...\n", name)
			runFunc()
		}
	}()
}
