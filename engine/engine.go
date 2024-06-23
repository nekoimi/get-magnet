package engine

import "get-magnet/storage"

const DefaultWorkerNum = 10

type Engine struct {
	Scheduler Scheduler
	WorkerNum int
}

type Scheduler interface {
	Submit(task Task)
}

type SimpleScheduler struct {
	requestChan chan string

	// 存储接口
	storage *storage.Storage
}

// Default create default Engine
func Default() *Engine {
	return New(DefaultWorkerNum)
}

// New create new Engine instance
// workerNum: worker num, default 10
func New(workerNum int) *Engine {
	return &Engine{
		Scheduler: nil,
		WorkerNum: workerNum,
	}
}

// Run start Engine
func (e *Engine) Run() {
	for {

	}
}
