package scheduler

import (
	"context"
	"fmt"
	"get-magnet/internal/task"
	"get-magnet/pkg/downloader"
	"log"
	"time"
)

type Worker struct {
	Id        int
	taskQueue chan *task.Task
	scheduler *Scheduler
	exit      chan struct{}
}

func NewWorker(id int, s *Scheduler) *Worker {
	return &Worker{
		Id:        id,
		taskQueue: make(chan *task.Task, s.workerNum*10),
		scheduler: s,
		exit:      make(chan struct{}),
	}
}

func (w *Worker) Run() {
	log.Printf("start worker %s ...\n", w)
	w.scheduler.ReadyWorker(w)

	for {
		select {
		case <-w.exit:
			return
		case t := <-w.taskQueue:
			w.handle(t)
		}
	}
}

func (w *Worker) Stop() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for len(w.taskQueue) > 0 {
			time.Sleep(1 * time.Second)
		}
		close(w.exit)
		log.Printf("stop worker %s \n", w)
		cancel()
	}()

	<-ctx.Done()
}

// handle run worker handle
// download url raw data & parse html doc
func (w *Worker) handle(t *task.Task) {
	defer w.scheduler.ReadyWorker(w)

	s, err := downloader.Download(t.Url)
	if err != nil {
		// again
		t.SetErrorMessage(err.Error())
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
	log.Printf("[%s] Handle task done: %s \n", w, t.Url)
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d", w.Id)
}
