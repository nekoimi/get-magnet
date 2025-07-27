package crawler

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

// ResultHandler 结果处理器
type ResultHandler interface {
	// Success worker处理任务成功
	Success(w *Worker, tasks []CrawlerTask, outputs []MagnetEntry)
	// Error worker处理任务异常
	Error(w *Worker, t CrawlerTask, err error)
}

type Worker struct {
	// context
	ctx context.Context
	// id
	id int
	// 结果处理器
	resultHandler ResultHandler
	// 任务chan
	taskCh chan CrawlerTask
	// 是否正在运行
	running bool
}

// NewWorker 创建一个新的任务执行worker
func NewWorker(ctx context.Context, id int, taskCh chan CrawlerTask, resultHandler ResultHandler) *Worker {
	return &Worker{
		ctx:           ctx,
		id:            id,
		resultHandler: resultHandler,
		taskCh:        taskCh,
		running:       false,
	}
}

func (w *Worker) Id() int {
	return w.id
}

// Run 启动任务执行worker，监听任务并执行
func (w *Worker) Run() {
	log.Debugf("启动Worker: %s - [%v]...", w, w.taskCh)
	for {
		select {
		case <-w.ctx.Done():
			return
		case t := <-w.taskCh:
			func() {
				defer func() {
					if r := recover(); r != nil {
						w.resultHandler.Error(w, t, errors.New(fmt.Sprintf("panic: %v", r)))
						log.Errorf("worker (%s) 处理任务 (%s) panic: %v", w, t.RawUrl(), r)
					}
				}()

				w.do(t)
			}()
		}
	}
}

// do 执行任务
func (w *Worker) do(t CrawlerTask) {
	w.running = true
	defer func() {
		w.running = false
	}()

	handler := t.Handler()
	tasks, outputs, err := handler(t)
	if err != nil {
		w.resultHandler.Error(w, t, err)
		log.Errorf("[%s] handle task (%s) err: %s", w, t.RawUrl(), err.Error())
		return
	}
	w.resultHandler.Success(w, tasks, outputs)
	log.Debugf("[%s] handle task done: %s", w, t.RawUrl())
}

// Close 停止任务执行worker
func (w *Worker) Close() error {
	for w.running {
		log.Debugf("等待Worker执行完毕: %s", w)
		time.Sleep(3 * time.Second)
	}
	log.Debugf("停止Worker: %s", w)
	return nil
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d", w.Id())
}
