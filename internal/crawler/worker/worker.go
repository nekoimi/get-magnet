package worker

import (
	"errors"
	"fmt"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"log"
	"time"
)

// ResultHandler 结果处理器
type ResultHandler interface {
	// Success worker处理任务成功
	Success(w *Worker, tasks []task.Task, outputs []task.MagnetEntry)
	// Error worker处理任务异常
	Error(w *Worker, t task.Task, err error)
}

type Worker struct {
	id            uint64
	version       uint64
	resultHandler ResultHandler
	tasks         chan task.Task
	exit          chan struct{}
	running       bool
}

// NewWorker 创建一个新的任务执行worker
func NewWorker(id uint64, version uint64, resultHandler ResultHandler) *Worker {
	return &Worker{
		id:            id,
		version:       version,
		resultHandler: resultHandler,
		tasks:         make(chan task.Task, 1),
		exit:          make(chan struct{}),
		running:       false,
	}
}

func (w *Worker) Id() uint64 {
	return w.id
}

func (w *Worker) Version() uint64 {
	return w.version
}

// Run 启动任务执行worker，监听任务并执行
func (w *Worker) Run() {
	log.Printf("启动Worker: %s...\n", w)
	for {
		select {
		case <-w.exit:
			return
		case t := <-w.tasks:
			func() {
				defer func() {
					if r := recover(); r != nil {
						w.resultHandler.Error(w, t, errors.New(fmt.Sprintf("panic: %v\n", r)))
						log.Printf("worker(%s)处理任务(%s)panic: %v\n", w, t.RawUrl(), r)
					}
				}()

				w.do(t)
			}()
		}
	}
}

// Work 投递任务
func (w *Worker) Work(t task.Task) {
	w.tasks <- t
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
func (w *Worker) do(t task.Task) {
	w.running = true
	defer func() {
		w.running = false
	}()

	handler := t.Handler()
	tasks, outputs, err := handler.Handle(t)
	if err != nil {
		t.IncrErrorNum()
		w.resultHandler.Error(w, t, err)
		log.Printf("[%s] handle task (%s) err: %s \n", w, t.RawUrl(), err.Error())
		return
	}
	w.resultHandler.Success(w, tasks, outputs)
	log.Printf("[%s] handle task done: %s \n", w, t.RawUrl())
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d@v%d", w.Id(), w.Version())
}
