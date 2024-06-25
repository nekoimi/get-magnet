package scheduler

import (
	"context"
	"log"
)

type Scheduler struct {
	workerNum       int
	workOutQueue    chan TaskOut
	waitTaskChan    chan Task
	readyWorkerChan chan *Worker
}

func New(workerNum int) *Scheduler {
	return &Scheduler{
		workerNum:       workerNum,
		workOutQueue:    make(chan TaskOut, workerNum*10),
		waitTaskChan:    make(chan Task, workerNum*10),
		readyWorkerChan: make(chan *Worker, workerNum),
	}
}

func (s *Scheduler) Submit(task Task) {
	log.Printf("submit task to waitTaskChan: %s \n", task.Url)
	s.waitTaskChan <- task
}

func (s *Scheduler) Done(taskOut TaskOut) {
	s.workOutQueue <- taskOut
}

func (s *Scheduler) OutputChan() chan TaskOut {
	return s.workOutQueue
}

func (s *Scheduler) WorkerReady(w *Worker) {
	s.readyWorkerChan <- w
}

func (s *Scheduler) Dispatch(ctx context.Context) {
	log.Println("scheduler dispatch running...")
	var activeTaskQueue []Task
	var activeWorkerQueue []*Worker
	for {
		var activeTask Task
		var activeWorker *Worker
		if len(activeTaskQueue) > 0 && len(activeWorkerQueue) > 0 {
			activeTask = activeTaskQueue[0]
			activeWorker = activeWorkerQueue[0]
		}

		select {
		case <-ctx.Done():
			log.Println("cancel scheduler...")
			return
		case task := <-s.waitTaskChan:
			log.Printf("read task: %s \n", task.Url)
			activeTaskQueue = append(activeTaskQueue, task)
		case worker := <-s.readyWorkerChan:
			log.Printf("read ready %s \n", worker)
			activeWorkerQueue = append(activeWorkerQueue, worker)
		default:
			if activeWorker != nil {
				activeWorker.taskQueue <- activeTask
				log.Printf("dispatch task (%s) to %s \n", activeTask.Url, activeWorker)
				// 删除第一个元素
				activeTaskQueue = activeTaskQueue[1:]
				activeWorkerQueue = activeWorkerQueue[1:]
			}
		}
	}
}
