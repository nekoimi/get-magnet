package scheduler

import (
	"get-magnet/pkg/downloader"
	"log"
)

type WorkerTaskQueue chan Task

// StartWorker start worker
func StartWorker(scheduler *Scheduler) {
	workerTaskQueue := scheduler.NewTaskQueue()
	go workerLoop(scheduler, workerTaskQueue)
}

// workerLoop 工作循环
// Loop listen work queue
func workerLoop(scheduler *Scheduler, workerTaskQueue WorkerTaskQueue) {
	scheduler.WorkerReady(workerTaskQueue)
	for {
		select {
		case task := <-workerTaskQueue:
			handle(scheduler, task)
			scheduler.WorkerReady(workerTaskQueue)
		}
	}
}

// handle run worker handle
// download url raw data & parse html doc
func handle(scheduler *Scheduler, task Task) {
	s, err := downloader.Download(task.Url)
	if err != nil {
		// again
		scheduler.Submit(task)

		log.Printf("Download (%s) err: %s \n", task.Url, err.Error())
		return
	}

	// invoke parse handle
	result, err := task.Handle(s)
	if err != nil {
		log.Printf("Handle task (%s) err: %s \n", task.Url, err.Error())
		return
	}
	scheduler.Done(result)
}
