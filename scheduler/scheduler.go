package scheduler

import (
	"context"
	"fmt"
	"get-magnet/internal/task"
	"log"
	"time"
)

type Scheduler struct {
	workerNum       int
	exit            chan struct{}
	readyTaskChan   chan *task.Task
	readyWorkerChan chan *Worker

	activeTaskQueue   []*task.Task
	activeWorkerQueue []*Worker

	OutputQueue chan *task.Out
}

func New(workerNum int) *Scheduler {
	return &Scheduler{
		workerNum: workerNum,

		exit: make(chan struct{}),

		readyTaskChan:   make(chan *task.Task, workerNum*10),
		readyWorkerChan: make(chan *Worker, workerNum),

		activeTaskQueue:   make([]*task.Task, 0),
		activeWorkerQueue: make([]*Worker, 0),

		OutputQueue: make(chan *task.Out, workerNum*10),
	}
}

func (s *Scheduler) Submit(task *task.Task) {
	log.Printf("submit task to readyTaskChan: %s \n", task.Url)
	s.readyTaskChan <- task
}

func (s *Scheduler) ReadyWorker(w *Worker) {
	s.readyWorkerChan <- w
}

func (s *Scheduler) Done(taskOut *task.Out) {
	s.OutputQueue <- taskOut
}

func (s *Scheduler) Run() {
	log.Println("scheduler dispatch running...")
	for {
		var activeTask *task.Task
		var activeWorker *Worker
		if len(s.activeTaskQueue) > 0 && len(s.activeWorkerQueue) > 0 {
			activeTask = s.activeTaskQueue[0]
			activeWorker = s.activeWorkerQueue[0]
		}

		select {
		case <-s.exit:
			return
		case t := <-s.readyTaskChan:
			log.Printf("read task: %s \n", t.Url)
			s.activeTaskQueue = append(s.activeTaskQueue, t)
		case w := <-s.readyWorkerChan:
			log.Printf("read worker: %s \n", w)
			s.activeWorkerQueue = append(s.activeWorkerQueue, w)
		default:
			if activeWorker == nil || activeTask == nil {
				continue
			}
			activeWorker.taskQueue <- activeTask
			log.Printf("dispatch task (%s) to %s \n", activeTask.Url, activeWorker)
			s.activeTaskQueue = s.activeTaskQueue[1:]
			s.activeWorkerQueue = s.activeWorkerQueue[1:]
		}
	}
}

func (s *Scheduler) Stop() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for len(s.readyTaskChan) > 0 || len(s.activeTaskQueue) > 0 || len(s.activeWorkerQueue) < s.workerNum {
			log.Printf("wait task process, %s \n", s.debug())
			time.Sleep(1 * time.Second)
		}
		close(s.exit)
		log.Println("stop scheduler")
		cancel()
	}()

	return ctx
}

func (s *Scheduler) debug() string {
	return fmt.Sprintf("ready-task: %d, ready-worker: %d, task-queue: %d, worker-queue: %d",
		len(s.readyTaskChan), len(s.readyWorkerChan),
		len(s.activeTaskQueue), len(s.activeWorkerQueue))
}
