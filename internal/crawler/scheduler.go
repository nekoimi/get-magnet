package crawler

import (
	"github.com/nekoimi/get-magnet/internal/pkg/queue"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

type Scheduler struct {
	workerQueue  *queue.Queue[*Worker]
	taskQueue    *queue.Queue[WorkerTask]
	exitWg       *sync.WaitGroup
	exit         chan struct{}
	workerChExit chan struct{}
	taskChExit   chan struct{}
}

// NewScheduler 获取一个新的调度器实例
func NewScheduler() *Scheduler {
	return &Scheduler{
		workerQueue:  queue.New[*Worker]("worker-queue"),
		taskQueue:    queue.New[WorkerTask]("task-queue"),
		exitWg:       &sync.WaitGroup{},
		exit:         make(chan struct{}),
		workerChExit: make(chan struct{}),
		taskChExit:   make(chan struct{}),
	}
}

// Submit 提交一个任务
func (s *Scheduler) Submit(task WorkerTask) {
	s.taskQueue.Add(task)
}

// Ready 提交一个就绪等待任务执行的worker
func (s *Scheduler) Ready(w *Worker) {
	s.workerQueue.Add(w)
}

// Start 启动调度器
func (s *Scheduler) Start() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("scheduler运行异常: %s\n", debug.Stack())
		}
	}()

	var (
		workerCh     = make(chan *Worker)
		taskCh       = make(chan WorkerTask)
		activeWorker *Worker
		activeTask   WorkerTask
	)

	go func() {
		s.exitWg.Add(1)
		for {
			select {
			case <-s.workerChExit:
				log.Println("---workerCh-exit---")
				s.exitWg.Done()
				return
			default:
				if w, ok := s.workerQueue.PollWaitTimeout(5 * time.Second); ok {
					workerCh <- w
				}
				log.Println("---workerCh---")
			}
		}
	}()

	go func() {
		s.exitWg.Add(1)
		for {
			select {
			case <-s.taskChExit:
				log.Println("---taskCh-exit---")
				s.exitWg.Done()
				return
			default:
				if t, ok := s.taskQueue.PollWaitTimeout(5 * time.Second); ok {
					taskCh <- t
				}
				log.Println("---taskCh---")
			}
		}
	}()

	s.exitWg.Add(1)
	for {
		log.Println("AAAAAAAAA")
		select {
		case <-s.exit:
			log.Println("scheduler exit")
			s.exitWg.Done()
			return
		default:
			if activeWorker == nil {
				if w, ok := s.workerQueue.Poll(); ok {
					activeWorker = w
				}
			}

			if activeTask == nil {
				if t, ok := s.taskQueue.Poll(); ok {
					activeTask = t
				}
			}

			if activeWorker == nil || activeTask == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			log.Printf("调度任务(%s)到%s \n", activeTask.RawUrl(), activeWorker.String())
			activeWorker.Work(activeTask)
		}
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.workerChExit)
	close(s.taskChExit)
	close(s.exit)
	s.exitWg.Wait()
}
