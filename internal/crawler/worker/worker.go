package worker

import (
	"errors"
	"fmt"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	log "github.com/sirupsen/logrus"
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
	taskCh        chan task.Task
	exit          chan struct{}
	running       bool
}

// NewWorker 创建一个新的任务执行worker
func NewWorker(id uint64, version uint64, taskCh chan task.Task, resultHandler ResultHandler) *Worker {
	return &Worker{
		id:            id,
		version:       version,
		resultHandler: resultHandler,
		taskCh:        taskCh,
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
	log.Debugf("启动Worker: %s - [%v]...\n", w, w.taskCh)
	for {
		select {
		case <-w.exit:
			return
		case t := <-w.taskCh:
			func() {
				defer func() {
					if r := recover(); r != nil {
						w.resultHandler.Error(w, t, errors.New(fmt.Sprintf("panic: %v\n", r)))
						log.Errorf("worker (%s) 处理任务 (%s) panic: %v\n", w, t.RawUrl(), r)
					}
				}()

				w.do(t)
			}()
		}
	}
}

// Stop 停止任务执行worker
func (w *Worker) Stop() {
	for w.running {
		log.Debugf("等待Worker执行完毕: %s\n", w)
		time.Sleep(3 * time.Second)
	}
	close(w.exit)
	log.Debugf("停止Worker: %s\n", w)
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
		log.Errorf("[%s] handle task (%s) err: %s \n", w, t.RawUrl(), err.Error())
		return
	}
	w.resultHandler.Success(w, tasks, outputs)
	log.Debugf("[%s] handle task done: %s \n", w, t.RawUrl())
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d@v%d", w.Id(), w.Version())
}
