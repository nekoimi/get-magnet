package crawler

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/crawler/worker"
	log "github.com/sirupsen/logrus"
	"modernc.org/mathutil"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// 最大worker数
	maxWorkerNum = 512
)

type Scheduler struct {
	ctx context.Context
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
	// worker任务结果处理
	resultHandler worker.ResultHandler
	// 任务队列
	taskCh chan task.Task
}

// newScheduler 获取一个新的调度器实例
func newScheduler(ctx context.Context, resultHandler worker.ResultHandler) *Scheduler {
	s := &Scheduler{
		ctx:               ctx,
		workerLock:        &sync.RWMutex{},
		workerIdNext:      new(atomic.Uint64),
		workerLastVersion: 0,
		workerVersionNext: new(atomic.Uint64),
		workers:           make(map[uint64]*worker.Worker, config.Get().WorkerNum),
		resultHandler:     resultHandler,
		taskCh:            make(chan task.Task),
	}

	bus.Event().Subscribe(bus.SubmitTask.String(), s.Submit)

	return s
}

// Submit 提交一个任务
func (s *Scheduler) Submit(task task.Task) {
	log.Debugf("提交task：%s", task.RawUrl())
	s.taskCh <- task
}

// Start 启动调度器
func (s *Scheduler) Start() {
	log.Debugf("Scheduler启动...")

	// 初始化worker池
	s.initializationWorkerPool(config.Get().WorkerNum)

	showWorkerTicker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-showWorkerTicker.C:
			if log.IsLevelEnabled(log.DebugLevel) {
				s.workerLock.RLock()
				log.Debugf("[WATCH-DEBUG] workers池数量：%d", len(s.workers))
				for _, w := range s.workers {
					log.Debugf("[WATCH-DEBUG] worker：%s", w.String())
				}
				s.workerLock.RUnlock()
			}
		}
	}
}

// 初始化worker池
func (s *Scheduler) initializationWorkerPool(num int) {
	s.workerLock.Lock()
	defer s.workerLock.Unlock()

	s.workerLastVersion = s.workerVersionNext.Add(1)
	for i := 0; i < mathutil.Min(num, maxWorkerNum); i++ {
		workerId := s.workerIdNext.Add(1)

		w := worker.NewWorker(workerId, s.workerLastVersion, s.taskCh, s.resultHandler)
		s.workers[workerId] = w

		// start worker
		go w.Run()
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	var wg sync.WaitGroup
	wg.Add(len(s.workers))
	for _, w := range s.workers {
		go func(w *worker.Worker) {
			w.Stop()
			wg.Done()
		}(w)
	}
	wg.Wait()
}
