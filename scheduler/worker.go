package scheduler

import (
	"fmt"
	"get-magnet/internal/task"
	"get-magnet/pkg/downloader"
	"log"
)

type Worker struct {
	Id        int
	taskQueue chan *task.Task
	scheduler *Scheduler
}

func NewWorker(id int, s *Scheduler) *Worker {
	return &Worker{
		Id:        id,
		taskQueue: make(chan *task.Task, s.workerNum*10),
		scheduler: s,
	}
}

func (w *Worker) Run() {
	log.Printf("start %s \n", w)
	w.scheduler.ReadyWorker(w)
	go w.workerLoop()
}

// workerLoop 工作循环
// Loop listen work queue
func (w *Worker) workerLoop() {
	for t := range w.taskQueue {
		w.handle(t)
	}
}

// handle run worker handle
// download url raw data & parse html doc
func (w *Worker) handle(t *task.Task) {
	s, err := downloader.Download(t.Url)
	if err != nil {
		// again
		w.scheduler.Submit(t)

		log.Printf("[%s] Download (%s) err: %s \n", w, t.Url, err.Error())
		return
	}

	// invoke parse handle
	result, err := t.Handle(t.Meta, s)
	if err != nil {
		log.Printf("[%s] Handle task (%s) err: %s \n", w, t.Url, err.Error())
		return
	}
	w.scheduler.Done(result)
	w.scheduler.ReadyWorker(w)
	log.Printf("[%s] Handle task done: %s \n", w, t.Url)
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d", w.Id)
}
