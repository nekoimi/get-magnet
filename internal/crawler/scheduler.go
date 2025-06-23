package crawler

import (
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/crawler/worker"
	"github.com/nekoimi/get-magnet/internal/pkg/queue"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Scheduler struct {
	workerQueue *queue.Queue[*worker.Worker]
	taskQueue   *queue.Queue[task.Task]
	exit        chan struct{}
	exitWG      sync.WaitGroup
}

// NewScheduler 获取一个新的调度器实例
func NewScheduler() *Scheduler {
	return &Scheduler{
		workerQueue: queue.New[*worker.Worker]("worker-queue"),
		taskQueue:   queue.New[task.Task]("task-queue"),
		exit:        make(chan struct{}),
		exitWG:      sync.WaitGroup{},
	}
}

// Submit 提交一个任务
func (s *Scheduler) Submit(task task.Task) {
	s.taskQueue.Add(task)
}

// Ready 提交一个就绪等待任务执行的worker
func (s *Scheduler) Ready(w *worker.Worker) {
	s.workerQueue.Add(w)
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.exitWG.Add(1)
	for {
		var (
			activeWorker *worker.Worker
			activeTask   task.Task
		)

		select {
		case <-s.exit:
			log.Debugf("scheduler exit")
			s.exitWG.Done()
			return
		default:
			if activeWorker == nil {
				if w, ok := s.workerQueue.PollWaitTimeout(3 * time.Second); ok {
					activeWorker = w
				}
			}

			if activeTask == nil {
				if t, ok := s.taskQueue.PollWaitTimeout(3 * time.Second); ok {
					activeTask = t
				}
			}

			if activeWorker == nil || activeTask == nil {
				continue
			}
			log.Debugf("调度任务(%s)到%s\n", activeTask.RawUrl(), activeWorker.String())
			activeWorker.Work(activeTask)
		}
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.exit)
	s.exitWG.Wait()
}
