package engine

import (
	"get-magnet/scheduler"
	"get-magnet/storage"
	"get-magnet/storage/console_storage"
	"log"
)

const DefaultWorkerNum = 5

type Engine struct {
	// worker 数量
	workerNum int
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
		Scheduler: scheduler.New(workerNum),
		Storage:   console_storage.New(),
	}
}

// Run start Engine
func (e *Engine) Run() {
	for i := 0; i < e.workerNum; i++ {
		scheduler.StartWorker(e.Scheduler)
		log.Printf("Worker-%d Start...", i)
	}

	go e.Scheduler.Dispatch()

	for {
		select {
		case out := <-e.Scheduler.OutputChan():
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
	}
}
