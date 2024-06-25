package engine

import (
	"context"
	"get-magnet/aria2"
	"get-magnet/scheduler"
	"get-magnet/storage"
	"get-magnet/storage/db_storage"
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const DefaultWorkerNum = 5

type Engine struct {
	// Signal chan
	signalChan chan os.Signal
	// worker 数量
	workerNum int
	// WaitGroup
	wg *sync.WaitGroup
	// ctx
	ctx    context.Context
	cancel context.CancelFunc
	// aria2
	aria2 *aria2.Aria2
	// cron
	cron *cron.Cron
	// 任务调度器
	scheduler *scheduler.Scheduler
	// 结果存储接口
	Storage storage.Storage
}

// Default create default Engine
func Default() *Engine {
	return New(DefaultWorkerNum)
}

// New create new Engine instance
// workerNum: worker num, default value DefaultWorkerNum
func New(workerNum int) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		signalChan: make(chan os.Signal),
		workerNum:  workerNum,
		wg:         &sync.WaitGroup{},
		ctx:        ctx,
		cancel:     cancel,
		aria2:      aria2.New(),
		cron:       cron.New(),
		scheduler:  scheduler.New(workerNum),
		Storage:    db_storage.New(),
	}
}

// engineLoop Loop handle output
// or resubmit task
func (e *Engine) engineLoop() {
	for {
		select {
		case <-e.ctx.Done():
			// Done
			e.wg.Done()
			log.Printf("cancel engineLoop...")
			return
		case out := <-e.scheduler.OutputChan():
			for _, t := range out.Tasks {
				e.scheduler.Submit(t)
			}
			for _, item := range out.Items {
				err := e.Storage.Save(item)
				if err != nil {
					log.Printf("Save item err: %s \n", err.Error())
				}

				// submit the item to aria2 and start downloading
				e.aria2.Submit(item)
			}
		case s := <-e.signalChan:
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
				e.Stop(s)
			default:
				log.Println("Ignore Signal: ", s)
			}
		}
	}
}

// Run start Engine
func (e *Engine) Run() {
	e.wg.Add(1)
	for i := 0; i < e.workerNum; i++ {
		w := scheduler.NewWorker(i, e.scheduler)
		w.Run(e.ctx)
	}

	signal.Notify(e.signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go e.cron.Run()
	go e.aria2.Run(e.ctx)
	go e.scheduler.Dispatch(e.ctx)
	go e.engineLoop()
	e.wg.Wait()
}

// Submit add task to scheduler
func (e *Engine) Submit(task scheduler.Task) {
	e.scheduler.Submit(task)
}

// CronSubmit use cron func submit
func (e *Engine) CronSubmit(cron string, task scheduler.Task) {
	_, err := e.cron.AddFunc(cron, func() {
		e.scheduler.Submit(task)
	})
	if err != nil {
		log.Fatalf("Add cron submit err: %s \n", err.Error())
	}
}

// Stop shutdown engine
func (e *Engine) Stop(s os.Signal) {
	e.cron.Stop()
	e.cancel()
	log.Println("Shutdown...")
}
