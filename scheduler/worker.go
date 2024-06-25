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
	w.scheduler.WorkerReady(w.taskQueue)
	go w.workerLoop()
}

// workerLoop 工作循环
// Loop listen work queue
func (w *Worker) workerLoop() {
	for task := range w.taskQueue {
		w.handle(task)
	}
}

// handle run worker handle
// download url raw data & parse html doc
func (w *Worker) handle(task Task) {
	s, err := downloader.Download(task.Url)
	if err != nil {
		// again
		w.scheduler.Submit(task)

		log.Printf("[%s] Download (%s) err: %s \n", w, task.Url, err.Error())
		return
	}

	// invoke parse handle
	result, err := task.Handle(task.Meta, s)
	if err != nil {
		log.Printf("[%s] Handle task (%s) err: %s \n", w, task.Url, err.Error())
		return
	}
	w.scheduler.Done(result)
	w.scheduler.WorkerReady(w.taskQueue)
	log.Printf("[%s] Handle task done: %s \n", w, task.Url)
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d", w.Id)
}
