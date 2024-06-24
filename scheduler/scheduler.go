package scheduler

import "log"

type Scheduler struct {
	workerNum       int
	workOutQueue    chan TaskOut
	waitTaskChan    chan Task
	readyWorkerChan chan WorkerTaskQueue
}

func New(workerNum int) *Scheduler {
	return &Scheduler{
		workerNum:       workerNum,
		workOutQueue:    make(chan TaskOut, workerNum*10),
		waitTaskChan:    make(chan Task, workerNum*10),
		readyWorkerChan: make(chan WorkerTaskQueue, workerNum),
	}
}

func (s *Scheduler) NewTaskQueue() WorkerTaskQueue {
	return make(WorkerTaskQueue, s.workerNum*10)
}

func (s *Scheduler) Submit(task Task) {
	s.waitTaskChan <- task
}

func (s *Scheduler) Done(taskOut TaskOut) {
	s.workOutQueue <- taskOut
}

func (s *Scheduler) OutputChan() chan TaskOut {
	return s.workOutQueue
}

func (s *Scheduler) WorkerReady(workerTaskQueue WorkerTaskQueue) {
	s.readyWorkerChan <- workerTaskQueue
}

func (s *Scheduler) Dispatch() {
	log.Println("QueueScheduler dispatch...")
	var activeTaskQueue []Task
	var activeWorkerQueue []WorkerTaskQueue
	for {
		var activeTask Task
		var activeWorker WorkerTaskQueue
		if len(activeTaskQueue) > 0 && len(activeWorkerQueue) > 0 {
			activeTask = activeTaskQueue[0]
			activeWorker = activeWorkerQueue[0]
		}

		select {
		case task := <-s.waitTaskChan:
			activeTaskQueue = append(activeTaskQueue, task)
		case worker_ := <-s.readyWorkerChan:
			activeWorkerQueue = append(activeWorkerQueue, worker_)
		case activeWorker <- activeTask:
			// 删除第一个元素
			activeTaskQueue = activeTaskQueue[1:]
			activeWorkerQueue = activeWorkerQueue[1:]
		}
	}
}
