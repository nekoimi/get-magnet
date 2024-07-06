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
	"sync/atomic"
)

// 默认启动16个worker
const defaultWorkerNum = 16

type Engine struct {
	// worker操作锁
	wmux sync.Mutex
	// workerId 生成
	workerIdNext atomic.Int64
	// worker最新版本
	workerLastVersion int64
	workerVersionNext atomic.Int64
	// worker池
	workers map[int64]*Worker

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
		workers:     make(map[int64]*Worker, 0),
		allowSubmit: true,
		aria2:       aria2.New(cfg.Aria2),
		scheduler:   scheduler2.New(),
		Storage:     storage.NewStorage(storage.Db),
	}

	e.ScaleWorker(defaultWorkerNum)

	return e
}

// Run start Engine
func (e *Engine) Run() {
	go e.aria2.Run()

	for _, worker := range e.workers {
		e.scheduler.Ready(worker)
		go worker.Run()
	}

	e.scheduler.Run()
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

// ScaleWorker 更改worker池规模
func (e *Engine) ScaleWorker(num int) {
	e.wmux.Lock()
	e.workerLastVersion = e.workerVersionNext.Add(1)
	for i := 0; i < num; i++ {
		workerId := e.workerIdNext.Add(1)
		e.workers[workerId] = NewWorker(workerId, e.workerLastVersion, e)
	}
	e.wmux.Unlock()
}

func (e *Engine) Success(w *Worker, o *task.Out) {
	// TODO 任务结果处理
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

func (e *Engine) Error(w *Worker, t *task.Task, err error) {
	// TODO 错误记录
	// TODO 任务重试
}

func (e *Engine) Finally(w *Worker) {
	// 判断worker版本, 是否和最新版本一致
	// 版本一致: 保留该worker实例，继续执行后续任务
	if e.workerLastVersion == w.version {
		e.scheduler.Ready(w)
	} else {
		// 不一致: 直接释放掉
		delete(e.workers, w.id)
	}
}

// Stop shutdown engine
func (e *Engine) Stop() {
	e.allowSubmit = false
	e.scheduler.Stop()

	var wg sync.WaitGroup
	wg.Add(len(e.workers))
	for _, worker := range e.workers {
		go func(w *Worker) {
			w.Stop()
			wg.Done()
		}(worker)
	}
	wg.Wait()

	e.aria2.Stop()

	log.Println("stop engine")
}
