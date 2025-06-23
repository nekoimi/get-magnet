package crawler

import (
	"github.com/nekoimi/get-magnet/internal/aria2"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/crawler/worker"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	"log"
	"modernc.org/mathutil"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// 默认启动16个worker
	defaultWorkerNum = 4
	// 最大worker数
	maxWorkerNum = 512
	// 任务出现错误最多重试次数
	taskErrorMax = 5
)

type Engine struct {
	// worker操作锁
	workerLock *sync.RWMutex
	// workerId生成
	workerIdNext *atomic.Uint64
	// worker最新版本
	workerLastVersion uint64
	// worker版本生成
	workerVersionNext *atomic.Uint64
	// worker池
	workers map[uint64]*worker.Worker
	// aria2rpc 客户端
	aria2 *aria2.Aria2
	// 任务调度器
	scheduler *Scheduler
}

// New create new Engine instance
func New() *Engine {
	e := &Engine{
		workerLock:        &sync.RWMutex{},
		workerIdNext:      new(atomic.Uint64),
		workerLastVersion: 0,
		workerVersionNext: new(atomic.Uint64),
		workers:           make(map[uint64]*worker.Worker, defaultWorkerNum),
		aria2:             aria2.NewClient(),
		scheduler:         NewScheduler(),
	}

	bus.Event().Subscribe(bus.Download.String(), e.createDownload)
	bus.Event().Subscribe(bus.SubmitTask.String(), e.scheduler.Submit)
	bus.Event().Subscribe(bus.Aria2Test.String(), func() {})
	bus.Event().Subscribe(bus.Aria2LinkUp.String(), func() {})
	bus.Event().Subscribe(bus.Aria2LinkDown.String(), func() {})

	return e
}

// Run start Engine
func (e *Engine) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("engine运行异常: %v, %s\n", r, string(debug.Stack()))
		}
	}()

	//go func() {
	//	ticker := time.NewTicker(60 * time.Second)
	//	for {
	//		select {
	//		case <-ticker.C:
	//			e.workerLock.RLock()
	//			log.Printf("[WATCH-DEBUG] workers池数量：%d\n", len(e.workers))
	//			for _, w := range e.workers {
	//				log.Printf("[WATCH-DEBUG] worker：%s\n", w.String())
	//			}
	//			e.workerLock.RUnlock()
	//		}
	//	}
	//}()

	// 初始化worker池
	e.initWorkerPool(defaultWorkerNum)
	// 启动aria2连接
	apptools.AutoRestart("aria2客户端", e.aria2.Start, 10*time.Second)
	// 启动任务生成
	apptools.DelayStart("任务生成", startTaskSeeders, 10*time.Second)
	// 启动任务调度器
	e.scheduler.Start()
}

// 添加下载任务
func (e *Engine) createDownload(downloadUrl string) {
	if err := e.aria2.Submit(downloadUrl); err != nil {
		log.Println(err.Error())
	}
}

// 初始化worker池
func (e *Engine) initWorkerPool(num int) {
	e.workerLock.Lock()
	defer e.workerLock.Unlock()

	e.workerLastVersion = e.workerVersionNext.Add(1)
	for i := 0; i < mathutil.Min(num, maxWorkerNum); i++ {
		workerId := e.workerIdNext.Add(1)

		w := worker.NewWorker(workerId, e.workerLastVersion, e)
		e.workers[workerId] = w

		// start worker
		go w.Run()

		e.scheduler.Ready(w)
	}
}

func (e *Engine) Success(w *worker.Worker, tasks []task.Task, outputs []task.MagnetEntry) {
	for _, t := range tasks {
		e.scheduler.Submit(t)
	}

	for _, output := range outputs {
		_, err := db.Instance().InsertOne(&table.Magnets{
			Title:       output.Title,
			Number:      output.Number,
			OptimalLink: output.OptimalLink,
			Links:       output.Links,
			ResHost:     output.ResHost,
			ResPath:     output.ResPath,
			Status:      0,
		})
		if err != nil {
			log.Printf("保存数据异常: %s \n", err.Error())
		}

		// 提交下载
		e.createDownload(output.OptimalLink)
	}

	e.release(w)
}

func (e *Engine) Error(w *worker.Worker, t task.Task, err error) {
	if t.ErrorNum() >= taskErrorMax {
		log.Printf("任务出错次数太多: %s - %s\n", t.RawUrl(), err.Error())
		return
	}

	log.Printf("任务处理异常：%s - %s\n", t.RawUrl(), err.Error())

	e.scheduler.Submit(t)

	e.release(w)
}

func (e *Engine) release(w *worker.Worker) {
	// 判断worker版本, 是否和最新版本一致
	if e.workerLastVersion == w.Version() {
		// 版本一致: 保留该worker实例，继续执行后续任务
		e.scheduler.Ready(w)
	} else {
		// 版本不一致: 直接释放掉
		e.workerLock.Lock()
		defer e.workerLock.Unlock()

		delete(e.workers, w.Id())
	}
}

// Stop shutdown engine
func (e *Engine) Stop() {
	stopTaskSeeders()
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
