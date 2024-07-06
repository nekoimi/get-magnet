package scheduler

import (
	"context"
	"github.com/nekoimi/get-magnet/contract"
	"github.com/nekoimi/get-magnet/engine/worker"
	"github.com/nekoimi/get-magnet/pkg/queue"
	"log"
)

const TaskErrorMax = 5

type Scheduler struct {
	taskQueue   *queue.Queue[contract.Task]
	workerQueue *queue.Queue[*worker.Worker]

	workerNum int
	exit      chan struct{}
}

// NewScheduler 获取一个新的调度器实例
func NewScheduler() *Scheduler {
	return &Scheduler{
		taskQueue:   queue.New[contract.Task](),
		workerQueue: queue.New[*worker.Worker](),

		workerNum: 16,

		exit: make(chan struct{}),
	}
}

// Submit 提交一个任务
func (s *Scheduler) Submit(task contract.Task) {
	//if task.ErrorCount >= TaskErrorMax {
	//	log.Printf("Too many task errors, ignore task: %s \n", task.GetUrl())
	//	return
	//}
	s.taskQueue.Add(task)
}

// Ready 提交一个就绪等待任务执行的worker
func (s *Scheduler) Ready(w *worker.Worker) {
	s.workerQueue.Add(w)
}

// Start 启动调度器
func (s *Scheduler) Start() {
	for {
		select {
		case <-s.exit:
			return
		default:
			// 阻塞等待就绪的worker
			waitWorker := s.workerQueue.PollWait()
			if t, exists := s.taskQueue.Poll(); exists {
				log.Printf("dispatch task (%s) to %s \n", t.GetUrl(), waitWorker.String())
				waitWorker.Deliver(t)
			} else {
				// 有等待执行任务的worker
				// 但是没有要执行的任务
				waitWorker.Release()
			}
		}
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		close(s.exit)
		cancel()
	}()

	<-ctx.Done()
}
