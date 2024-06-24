package scheduler

import "log"

type Scheduler struct {
	workerNum       int
	workOutQueue    chan TaskOut
	waitTaskChan    chan Task
	readyWorkerChan chan chan Task
}

func New(workerNum int) *Scheduler {
	return &Scheduler{
		workerNum:       workerNum,
		workOutQueue:    make(chan TaskOut, workerNum*10),
		waitTaskChan:    make(chan Task, workerNum*10),
		readyWorkerChan: make(chan chan Task, workerNum),
	}
}

func (s *Scheduler) Submit(task Task) {
	log.Printf("submit task to waitTaskChan: %s \n", task.Url)
	s.waitTaskChan <- task
}

func (s *Scheduler) Done(taskOut TaskOut) {
	log.Println("done task to workOutQueue")
	s.workOutQueue <- taskOut
}

func (s *Scheduler) OutputChan() chan TaskOut {
	return s.workOutQueue
}

func (s *Scheduler) WorkerReady(taskQueue chan Task) {
	log.Println("ready workerTaskQueue")
	s.readyWorkerChan <- taskQueue
}

func (s *Scheduler) Dispatch() {
	log.Println("QueueScheduler dispatch...")
	var activeTaskQueue []Task
	var activeWorkerQueue []chan Task
	for {
		var activeTask Task
		var activeWorker chan Task
		if len(activeTaskQueue) > 0 && len(activeWorkerQueue) > 0 {
			activeTask = activeTaskQueue[0]
			activeWorker = activeWorkerQueue[0]
		}

		select {
		case task := <-s.waitTaskChan:
			log.Printf("Read task: %s \n", task.Url)
			activeTaskQueue = append(activeTaskQueue, task)
		case worker_ := <-s.readyWorkerChan:
			log.Printf("Read ready worker\n")
			activeWorkerQueue = append(activeWorkerQueue, worker_)
		case activeWorker <- activeTask:
			log.Printf("Dispatch task (%s) to worker \n", activeTask.Url)
			// 删除第一个元素
			activeTaskQueue = activeTaskQueue[1:]
			activeWorkerQueue = activeWorkerQueue[1:]
		}
	}
}
