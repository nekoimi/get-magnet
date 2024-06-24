package engine

import (
	"get-magnet/scheduler"
	"get-magnet/storage"
	"get-magnet/storage/console_storage"
	"github.com/robfig/cron/v3"
	"log"
	"sync"
)

const DefaultWorkerNum = 5

type Engine struct {
	// worker 数量
	workerNum int
	// WaitGroup
	wg *sync.WaitGroup
	// Cron
	Cron *cron.Cron
	// 任务调度器
	Scheduler *scheduler.Scheduler
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
	return &Engine{
		workerNum: workerNum,
		wg:        &sync.WaitGroup{},
		Cron:      cron.New(),
		Scheduler: scheduler.New(workerNum),
		Storage:   console_storage.New(),
	}
}

// engineLoop Loop handle output
// or resubmit task
func (e *Engine) engineLoop() {
	for out := range e.Scheduler.OutputChan() {
		for _, t := range out.Tasks {
			e.Scheduler.Submit(t)
		}
		for _, item := range out.Items {
			err := e.Storage.Save(item)
			if err != nil {
				log.Printf("Save item err: %s \n", err.Error())
			}
		}
	}

	// Done
	e.wg.Done()
}

// Run start Engine
func (e *Engine) Run() {
	for i := 0; i < e.workerNum; i++ {
		scheduler.StartWorker(e.Scheduler)
		log.Printf("Worker-%d Start...", i)
	}

	go e.Cron.Run()
	go e.Scheduler.Dispatch()
	go e.engineLoop()
}

// RunWait start Engine and Wait
func (e *Engine) RunWait() {
	e.wg.Add(1)
	e.Run()
	e.wg.Wait()
}
