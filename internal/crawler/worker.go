package crawler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nekoimi/get-magnet/internal/repo/task_repo"
	log "github.com/sirupsen/logrus"
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
	// 任务队列
	taskDispatcher TaskDispatcher
	// 是否正在运行
	running     atomic.Bool
	current     atomic.Value
	taskMu      sync.RWMutex
	currentTask CrawlerTask
}

// NewWorker 创建一个新的任务执行worker
func NewWorker(ctx context.Context, id int, taskDispatcher TaskDispatcher, resultHandler ResultHandler) *Worker {
	w := &Worker{
		ctx:            ctx,
		id:             id,
		resultHandler:  resultHandler,
		taskDispatcher: taskDispatcher,
	}
	w.current.Store("")
	return w
}

func (w *Worker) Id() int {
	return w.id
}

// Run 启动任务执行worker，监听任务并执行
func (w *Worker) Run() {
	log.Debugf("启动Worker: %s ...", w)
	for {
		select {
		case <-w.ctx.Done():
			return
		case t := <-w.taskDispatcher.Chan():
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
	w.running.Store(true)
	w.current.Store(t.RawUrl())
	defer func() {
		w.running.Store(false)
		w.current.Store("")
		w.taskMu.Lock()
		w.currentTask = nil
		w.taskMu.Unlock()
	}()
	w.taskMu.Lock()
	w.currentTask = t
	w.taskMu.Unlock()
	if entry, ok := t.(*TaskEntry); ok && entry.taskID > 0 {
		if attempt, err := task_repo.StartAttempt(entry.taskID, w.String(), task_repo.TaskInput(entry.RawURL, entry.Origin)); err == nil {
			entry.SetAttempt(attempt.Id)
			_, stopLease := task_repo.MaintainLease(w.ctx, entry.taskID, *attempt, 5*time.Minute)
			defer stopLease()
		} else {
			log.Warnf("创建任务尝试记录失败：%s", err.Error())
			return
		}
	}

	handler := t.Handler()
	tasks, outputs, err := handler(t)
	if entry, ok := t.(*TaskEntry); ok && entry.taskID > 0 && entry.attemptID > 0 {
		if checkErr := task_repo.CheckAttemptID(entry.taskID, entry.attemptID); checkErr != nil {
			log.Warnf("丢弃失去租约的采集结果: task=%d error=%s", entry.taskID, checkErr)
			return
		}
	}
	if err != nil {
		w.resultHandler.Error(w, t, err)
		log.Errorf("[%s] handle task (%s) err: %s", w, t.RawUrl(), err.Error())
		return
	}
	w.resultHandler.Success(w, tasks, outputs)
	log.Debugf("[%s] handle task done: %s", w, t.RawUrl())
}

func (w *Worker) CurrentTask() CrawlerTask {
	w.taskMu.RLock()
	defer w.taskMu.RUnlock()
	return w.currentTask
}

// Close 停止任务执行worker
func (w *Worker) Close() error {
	for w.running.Load() {
		log.Debugf("等待Worker执行完毕: %s", w)
		time.Sleep(3 * time.Second)
	}
	log.Debugf("停止Worker: %s", w)
	return nil
}

type WorkerSnapshot struct {
	ID         int    `json:"id"`
	Running    bool   `json:"running"`
	CurrentURL string `json:"current_url,omitempty"`
}

func (w *Worker) Snapshot() WorkerSnapshot {
	current, _ := w.current.Load().(string)
	return WorkerSnapshot{ID: w.id, Running: w.running.Load(), CurrentURL: current}
}

func (w *Worker) String() string {
	return fmt.Sprintf("worker-%d", w.Id())
}
