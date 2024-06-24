package scheduler

import (
	"fmt"
	"get-magnet/pkg/downloader"
	"log"
)

type Worker struct {
	Id        int
	taskQueue chan Task
	scheduler *Scheduler
}

func NewWorker(id int, s *Scheduler) *Worker {
	return &Worker{
		Id:        id,
		taskQueue: make(chan Task, s.workerNum*10),
		scheduler: s,
	}
}

func (w *Worker) Run() {
	log.Printf("Start %s \n", w)
	go workerLoop(w.scheduler, w.taskQueue)
}

// workerLoop 工作循环
// Loop listen work queue
func workerLoop(s *Scheduler, taskQueue chan Task) {
	s.WorkerReady(taskQueue)
	for task := range taskQueue {
		handle(s, task)
		s.WorkerReady(taskQueue)
	}
}

// handle run worker handle
// download url raw data & parse html doc
func handle(scheduler *Scheduler, task Task) {
	s, err := downloader.Download(task.Url)
	if err != nil {
		// again
		// TODO debug not submit
		// scheduler.Submit(task)

		log.Printf("Download (%s) err: %s \n", task.Url, err.Error())
		return
	}

	// invoke parse handle
	result, err := task.Handle(task.Meta, s)
	if err != nil {
		log.Printf("Handle task (%s) err: %s \n", task.Url, err.Error())
		return
	}
	scheduler.Done(result)
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d", w.Id)
}
