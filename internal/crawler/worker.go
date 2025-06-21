package crawler

import (
	"fmt"
	"log"
	"time"
)

type Worker struct {
	id       uint64
	version  uint64
	callback WorkerCallback
	tasks    chan WorkerTask
	exit     chan struct{}
	running  bool
}

// 创建一个新的任务执行worker
func newWorker(id uint64, version uint64, callback WorkerCallback) *Worker {
	return &Worker{
		id:       id,
		version:  version,
		callback: callback,
		tasks:    make(chan WorkerTask, 1),
		exit:     make(chan struct{}),
		running:  false,
	}
}

// Run 启动任务执行worker，监听任务并执行
func (w *Worker) Run() {
	log.Printf("启动Worker: %s...\n", w)
	for {
		select {
		case <-w.exit:
			return
		case t := <-w.tasks:
			w.do(t)
		}
	}
}

// Work 投递任务
func (w *Worker) Work(t WorkerTask) {
	w.tasks <- t
}

// Release 释放worker
func (w *Worker) Release() {
	w.callback.release(w)
}

// Stop 停止任务执行worker
func (w *Worker) Stop() {
	for w.running {
		log.Printf("等待Worker执行完毕: %s\n", w)
		time.Sleep(3 * time.Second)
	}
	close(w.exit)
	log.Printf("停止Worker: %s\n", w)
}

// do 执行任务
func (w *Worker) do(t WorkerTask) {
	w.running = true
	defer func() {
		w.Release()
		w.running = false
	}()

	handler := t.Handler()
	tasks, outputs, err := handler.Handle(t)
	if err != nil {
		w.callback.error(t, err)
		log.Printf("[%s] handle task (%s) err: %s \n", w, t.RawUrl(), err.Error())
		return
	}
	w.callback.success(tasks, outputs)
	log.Printf("[%s] handle task done: %s \n", w, t.RawUrl())
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d@v%d", w.id, w.version)
}
