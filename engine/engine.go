package engine

import (
	"github.com/nekoimi/get-magnet/aria2"
	"github.com/nekoimi/get-magnet/common/model"
	"github.com/nekoimi/get-magnet/common/task"
	"github.com/nekoimi/get-magnet/config"
	scheduler2 "github.com/nekoimi/get-magnet/scheduler"
	"github.com/nekoimi/get-magnet/storage"
	"log"
	"sync"
)

type Engine struct {
	workers []*scheduler2.Worker

	// allow to submit
	allowSubmit bool

	// aria2 rpc 客户端
	aria2 *aria2.Aria2
	// 任务调度器
	scheduler *scheduler2.Scheduler
	// 存储
	Storage storage.Storage
}

// New create new Engine instance
// workerNum: worker num
func New(cfg *config.Engine) *Engine {
	e := &Engine{
		workers:     make([]*scheduler2.Worker, 0),
		allowSubmit: true,
		aria2:       aria2.New(cfg.Aria2),
		scheduler:   scheduler2.New(cfg.WorkerNum),
		Storage:     storage.NewStorage(storage.Db),
	}

	for i := 0; i < cfg.WorkerNum; i++ {
		e.workers = append(e.workers, scheduler2.NewWorker(i, e.scheduler))
	}

	return e
}

// Run start Engine
func (e *Engine) Run() {
	go e.aria2.Run()

	e.scheduler.SetOutputHandle(e.taskOutputHandle)
	go e.scheduler.Run()

	for _, worker := range e.workers {
		go worker.Run()
	}
}

// SubmitDownload add item to aria2 and start download
func (e *Engine) SubmitDownload(item *model.Item) {
	e.aria2.Submit(item)
}

// Submit add task to scheduler
func (e *Engine) Submit(task *task.Task) {
	if e.allowSubmit {
		e.scheduler.Submit(task)
		return
	}
	log.Printf("Not allow to submit, ignore task: %s \n", task.Url)
}

func (e *Engine) taskOutputHandle(o *task.Out) {
	for _, t := range o.Tasks {
		e.Submit(t)
	}
	for _, item := range o.Items {
		err := e.Storage.Save(item)
		if err != nil {
			log.Printf("Save item err: %s \n", err.Error())
		}

		// submit the item to aria2 and start downloading
		e.aria2.Submit(item)
	}
}

// Stop shutdown engine
func (e *Engine) Stop() {
	e.allowSubmit = false
	e.scheduler.Stop()

	var wg sync.WaitGroup
	wg.Add(len(e.workers))
	for _, worker := range e.workers {
		go func(w *scheduler2.Worker) {
			w.Stop()
			wg.Done()
		}(worker)
	}
	wg.Wait()

	e.aria2.Stop()

	log.Println("stop engine")
}
