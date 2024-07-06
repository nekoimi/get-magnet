package worker

import (
	"context"
	"fmt"
	"github.com/nekoimi/get-magnet/contract"
	"log"
	"time"
)

type Worker struct {
	id       int64
	version  int64
	callback Callback
	task     chan contract.Task
	exit     chan struct{}
	running  bool
}

type Callback interface {
	Success(w *Worker, tasks []contract.Task, outputs ...any)
	Error(w *Worker, t contract.Task, err error)
	Finally(w *Worker)
}

// NewWorker 创建一个新的任务执行worker
func NewWorker(id int64, version int64, callback Callback) *Worker {
	return &Worker{
		id:       id,
		version:  version,
		callback: callback,
		task:     make(chan contract.Task, 1),
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
		case t := <-w.task:
			w.do(t)
		}
	}
}

// Id 获取worker id
func (w *Worker) Id() int64 {
	return w.id
}

// Version 获取worker版本
func (w *Worker) Version() int64 {
	return w.version
}

// Deliver 投递任务
func (w *Worker) Deliver(t contract.Task) {
	w.task <- t
}

// Release 释放worker
func (w *Worker) Release() {
	w.callback.Finally(w)
}

// Stop 停止任务执行worker
func (w *Worker) Stop() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for w.running {
			log.Printf("等待Worker执行完毕: %s\n", w)
			time.Sleep(3 * time.Second)
		}
		close(w.exit)
		log.Printf("停止Worker: %s\n", w)
		cancel()
	}()

	<-ctx.Done()
}

// do 执行任务
func (w *Worker) do(t contract.Task) {
	w.running = true
	defer func() {
		w.callback.Finally(w)
		w.running = false
	}()

	handler := t.GetHandler()
	switch handler.(type) {
	case contract.SimpleTaskHandler:
		break
	case contract.HTMLQueryParseHandler:
		//s, err := downloader.Download(t.GetUrl())
		//if err != nil {
		//	w.callback.Error(w, t, err)
		//	log.Printf("[%s] Download (%s) err: %s \n", w, t.GetUrl(), err.Error())
		//	return
		//}
		//parseHandler := handler.(engine.HTMLQueryParseHandler)
		//result, err := parseHandler.Handle(s)
		//if err != nil {
		//	w.callback.Error(w, t, err)
		//	log.Printf("[%s] Handle task (%s) err: %s \n", w, t.GetUrl(), err.Error())
		//	return
		//}
		//w.callback.Success(w, result)
		//log.Printf("[%s] Handle task done: %s \n", w, t.GetUrl())
		break
	}
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d@v%d", w.id, w.version)
}
