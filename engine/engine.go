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
	"time"
)

const DefaultWorkerNum = 5

type Engine struct {
	signalChan chan os.Signal
	workerNum  int

	workers []*scheduler.Worker

	// allow to submit
	allowSubmit bool

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
		signalChan:  make(chan os.Signal),
		workerNum:   workerNum,
		workers:     make([]*scheduler.Worker, 0),
		allowSubmit: true,
		aria2:       aria2.New(),
		cron:        cron.New(),
		scheduler:   scheduler.New(workerNum),
		Storage:     storage.NewStorage(st),
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

	exit := make(chan struct{})
	go func() {
		for {
			select {
			case <-exit:
				return
			case out := <-e.scheduler.OutputQueue:
				log.Printf("scheduler.OutputQueue: %v \n", out)
				for _, t := range out.Tasks {
					e.Submit(t)
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
	}()

	for {
		select {
		case s := <-e.signalChan:
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
				e.Stop(s, exit)
				return
			default:
				log.Println("Ignore Signal: ", s)
			}
		}
	}
}

// Submit add task to scheduler
func (e *Engine) Submit(task *task.Task) {
	if e.allowSubmit {
		e.scheduler.Submit(task)
		return
	}
	log.Printf("Not allow to submit, ignore task: %s \n", task.Url)
}

// CronSubmit use cron func submit
func (e *Engine) CronSubmit(cron string, task *task.Task) {
	_, err := e.cron.AddFunc(cron, func() {
		e.Submit(task)
	})
	if err != nil {
		log.Fatalf("Add cron submit err: %s \n", err.Error())
	}
}

// Stop shutdown engine
func (e *Engine) Stop(s os.Signal, exit chan struct{}) {
	e.allowSubmit = false

	c := e.cron.Stop()
	<-c.Done()

	e.scheduler.Stop()

	var wg sync.WaitGroup
	wg.Add(len(e.workers))
	for _, worker := range e.workers {
		go func(w *scheduler.Worker) {
			w.Stop()
			wg.Done()
		}(worker)
	}
	wg.Wait()

	for len(e.scheduler.OutputQueue) > 0 {
		time.Sleep(1 * time.Second)
	}

	close(exit)

	e.aria2.Stop()

	log.Println("stop engine")
}
