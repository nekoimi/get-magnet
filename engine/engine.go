package engine

import (
	"get-magnet/aria2"
	"get-magnet/internal/task"
	"get-magnet/scheduler"
	"get-magnet/storage"
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const DefaultWorkerNum = 5

type Engine struct {
	signalChan chan os.Signal
	workerNum  int

	workers []*scheduler.Worker

	// aria2 rpc 客户端
	aria2 *aria2.Aria2
	// 定时任务调度
	cron *cron.Cron
	// 任务调度器
	scheduler *scheduler.Scheduler
	// 存储
	Storage storage.Storage
}

// Default create default Engine
func Default() *Engine {
	return New(DefaultWorkerNum, storage.Console)
}

// New create new Engine instance
// workerNum: worker num, default value DefaultWorkerNum
func New(workerNum int, st storage.Type) *Engine {
	e := &Engine{
		signalChan: make(chan os.Signal),
		workerNum:  workerNum,
		workers:    make([]*scheduler.Worker, 0),
		aria2:      aria2.New(),
		cron:       cron.New(),
		scheduler:  scheduler.New(workerNum),
		Storage:    storage.NewStorage(st),
	}

	signal.Notify(e.signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for i := 0; i < e.workerNum; i++ {
		e.workers = append(e.workers, scheduler.NewWorker(i, e.scheduler))
	}

	return e
}

// Run start Engine
func (e *Engine) Run() {
	go e.cron.Run()
	go e.aria2.Run()
	go e.scheduler.Run()

	for _, worker := range e.workers {
		go worker.Run()
	}

	for {
		select {
		case s := <-e.signalChan:
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
				e.Stop(s)
				return
			default:
				log.Println("Ignore Signal: ", s)
			}
		case out := <-e.scheduler.OutputQueue:
			log.Printf("scheduler.OutputQueue: %v \n", out)
			for _, t := range out.Tasks {
				e.scheduler.Submit(t)
			}
			for _, item := range out.Items {
				err := e.Storage.Save(item)
				if err != nil {
					log.Printf("Save item err: %s \n", err.Error())
				}

				// TODO submit the item to aria2 and start downloading
				e.aria2.Submit(item)
			}
		}
	}
}

// Submit add task to scheduler
func (e *Engine) Submit(task *task.Task) {
	e.scheduler.Submit(task)
}

// CronSubmit use cron func submit
func (e *Engine) CronSubmit(cron string, task *task.Task) {
	_, err := e.cron.AddFunc(cron, func() {
		e.scheduler.Submit(task)
	})
	if err != nil {
		log.Fatalf("Add cron submit err: %s \n", err.Error())
	}
}

// Stop shutdown engine
func (e *Engine) Stop(s os.Signal) {
	c2 := e.cron.Stop()
	<-c2.Done()

	c1 := e.scheduler.Stop()
	<-c1.Done()

	var wg sync.WaitGroup
	wg.Add(len(e.workers))
	for _, worker := range e.workers {
		go func(w *scheduler.Worker) {
			<-w.Stop().Done()
			wg.Done()
		}(worker)
	}
	wg.Wait()

	c3 := e.aria2.Stop()
	<-c3.Done()

	log.Println("stop engine")
}
