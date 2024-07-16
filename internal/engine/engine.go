package engine

import (
	"github.com/nekoimi/get-magnet/internal/aria2"
	contract2 "github.com/nekoimi/get-magnet/internal/contract"
	"github.com/nekoimi/get-magnet/internal/engine/scheduler"
	"github.com/nekoimi/get-magnet/internal/engine/worker"
	"github.com/nekoimi/get-magnet/internal/event"
	"github.com/nekoimi/get-magnet/internal/pkg/sample_downloader"
	"log"
	"modernc.org/mathutil"
	"sync"
	"sync/atomic"
)

const (
	// 默认启动16个worker
	defaultWorkerNum = 16
	// 最大worker数
	maxWorkerNum = 512
)

type Engine struct {
	// worker操作锁
	workerLock sync.Mutex
	// workerId生成
	workerIdNext atomic.Int64
	// worker最新版本
	workerLastVersion int64
	// worker版本生成
	workerVersionNext atomic.Int64
	// worker池
	workers map[int64]*worker.Worker
	// aria2rpc 客户端
	aria2 *aria2.Aria2
	// 任务调度器
	scheduler *scheduler.Scheduler
	// 下载器
	downloader contract2.Downloader

	// allow to submit
	allowSubmit bool
}

// New create new Engine instance
func New() *Engine {
	e := &Engine{
		workers:     make(map[int64]*worker.Worker, 0),
		allowSubmit: true,
		scheduler:   scheduler.NewScheduler(),
		downloader:  sample_downloader.New(),
	}

	event.GetBus().Subscribe(event.ScaleWorker.String(), e.ScaleWorker)
	event.GetBus().Subscribe(event.Download.String(), e.Download)
	event.GetBus().Subscribe(event.Aria2Test.String(), func() {})
	event.GetBus().Subscribe(event.Aria2LinkUp.String(), func() {})
	event.GetBus().Subscribe(event.Aria2LinkDown.String(), func() {})

	return e
}

// Run start Engine
func (e *Engine) Run() {
	e.ScaleWorker(defaultWorkerNum)
	e.scheduler.Start()
}

// Download 添加下载任务
func (e *Engine) Download(item contract2.DownloadTask) {
	// TODO 需要判断aria2的对接状态
	e.aria2.Submit(item)
}

// Submit 添加任务到调度器
func (e *Engine) Submit(task contract2.WorkerTask) {
	if e.allowSubmit {
		e.scheduler.Submit(task)
		return
	}
	log.Printf("Not allow to submit, ignore task: %s \n", task.Url())
}

// ScaleWorker 更改worker池规模
func (e *Engine) ScaleWorker(num int) {
	e.workerLock.Lock()
	defer e.workerLock.Unlock()

	e.workerLastVersion = e.workerVersionNext.Add(1)
	for i := 0; i < mathutil.Min(num, maxWorkerNum); i++ {
		workerId := e.workerIdNext.Add(1)

		w := worker.NewWorker(workerId, e.workerLastVersion, e.downloader, e)
		e.workers[workerId] = w

		e.scheduler.Ready(w)

		go w.Run()
	}
}

func (e *Engine) Success(w *worker.Worker, tasks []contract2.WorkerTask, outputs ...any) {
	// TODO 任务结果处理
	for _, t := range tasks {
		e.Submit(t)
	}
	//for _, item := range outputs {
	//err := e.Storage.Save(item)
	//if err != nil {
	//	log.Printf("Save item err: %s \n", err.Error())
	//}
	//
	//// submit the item to aria2 and start downloading
	//e.aria2.Submit(item)
	//}
}

func (e *Engine) Error(w *worker.Worker, t contract2.WorkerTask, err error) {
	// TODO 错误记录
	// TODO 任务重试
}

func (e *Engine) Finally(w *worker.Worker) {
	// 判断worker版本, 是否和最新版本一致
	// 版本一致: 保留该worker实例，继续执行后续任务
	if e.workerLastVersion == w.Version() {
		e.scheduler.Ready(w)
	} else {
		// 不一致: 直接释放掉
		e.workerLock.Lock()
		defer e.workerLock.Unlock()

		delete(e.workers, w.Id())
	}
}

// Stop shutdown engine
func (e *Engine) Stop() {
	e.allowSubmit = false
	e.scheduler.Stop()

	var wg sync.WaitGroup
	wg.Add(len(e.workers))
	for _, w := range e.workers {
		go func(w *worker.Worker) {
			w.Stop()
			wg.Done()
		}(w)
	}
	wg.Wait()

	e.aria2.Stop()

	log.Println("stop engine")
}