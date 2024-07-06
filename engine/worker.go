package engine

import (
	"context"
	"fmt"
	"github.com/nekoimi/get-magnet/common/task"
	"github.com/nekoimi/get-magnet/pkg/downloader"
	"log"
	"time"
)

type Worker struct {
	id       int64
	version  int64
	callback WorkerCallback

	TaskQueue chan *task.Task
	exit      chan struct{}
}

type WorkerCallback interface {
	Success(w *Worker, out *task.Out)
	Error(w *Worker, t *task.Task, err error)
	Finally(w *Worker)
}

func NewWorker(id int64, version int64, callback WorkerCallback) *Worker {
	return &Worker{
		id:       id,
		version:  version,
		callback: callback,

		TaskQueue: make(chan *task.Task, 16),
		exit:      make(chan struct{}),
	}
}

func (w *Worker) Run() {
	log.Printf("start worker %s ...\n", w)
	for {
		select {
		case <-w.exit:
			return
		case t := <-w.TaskQueue:
			w.callback(t)
		}
	}
}

func (w *Worker) Stop() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for len(w.TaskQueue) > 0 {
			time.Sleep(1 * time.Second)
		}
		close(w.exit)
		log.Printf("stop worker %s \n", w)
		cancel()
	}()

	<-ctx.Done()
}

// doWork run worker callback
// download url raw data & parse html doc
func (w *Worker) doWork(t *task.Task) {
	defer w.callback.Finally(w)

	s, err := downloader.Download(t.Url)
	if err != nil {
		w.callback.Error(w, t, err)
		log.Printf("[%s] Download (%s) err: %s \n", w, t.Url, err.Error())
		return
	}

	// invoke parse callback
	result, err := t.Handle(t.Meta, s)
	if err != nil {
		w.callback.Error(w, t, err)
		log.Printf("[%s] Handle task (%s) err: %s \n", w, t.Url, err.Error())
		return
	}
	w.callback.Success(w, result)
	log.Printf("[%s] Handle task done: %s \n", w, t.Url)
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d@v%d", w.id, w.version)
}
