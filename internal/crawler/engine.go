package crawler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/repo/resource_repo"
	"github.com/nekoimi/get-magnet/internal/repo/task_repo"
	log "github.com/sirupsen/logrus"
	"modernc.org/mathutil"
)

const (
	// MaxTaskErrorNum 任务出现错误最多重试次数
	MaxTaskErrorNum = 5
)

type Engine struct {
	// 配置文件
	cfg *config.CrawlerConfig
	// worker操作锁
	workerLock *sync.RWMutex
	// worker池
	workers []*Worker
	// 任务队列
	taskDispatcher TaskDispatcher
	// crawler管理器
	crawlerManager *Manager
	// cancel
	cancel context.CancelFunc
}

type EngineSnapshot struct {
	WorkerCount int              `json:"worker_count"`
	Running     int              `json:"running"`
	QueueLength int              `json:"queue_length"`
	Workers     []WorkerSnapshot `json:"workers"`
}

func NewCrawlerEngine() *Engine {
	return &Engine{
		workerLock:     &sync.RWMutex{},
		workers:        make([]*Worker, 0),
		taskDispatcher: NewCrawlerTaskQueue(512),
	}
}
func (e *Engine) Name() string {
	return "CrawlerEngine"
}

func (e *Engine) Start(parent context.Context) error {
	cfg := bean.PtrFromContext[config.Config](parent)
	e.cfg = cfg.Crawler
	e.crawlerManager = bean.PtrFromContext[Manager](parent)
	if recovered, err := task_repo.RecoverExpiredLeases(); err != nil {
		log.Warnf("恢复过期采集任务租约失败：%s", err.Error())
	} else if recovered > 0 {
		log.Infof("恢复过期采集任务租约：%d", recovered)
	}

	var subCtx context.Context
	subCtx, e.cancel = context.WithCancel(parent)

	e.workerLock.Lock()
	defer e.workerLock.Unlock()

	for i := 0; i < mathutil.Max(1, e.cfg.WorkerNum); i++ {
		w := NewWorker(subCtx, i, e.taskDispatcher, e)

		e.workers = append(e.workers, w)

		go w.Run()
	}

	if e.cfg.ExecOnStartup {
		time.AfterFunc(30*time.Second, func() {
			e.crawlerManager.RunAll()
		})
	}

	bus.Event().Subscribe(bus.SubmitTask.Topic(), e.Submit)

	e.crawlerManager.ScheduleAll()

	return nil
}

func (e *Engine) Submit(t CrawlerTask) {
	e.persistTask(t, 0, 0)
	log.Debugf("提交task：%s", t.RawUrl())
	e.taskDispatcher.Submit(t)
}

func (e *Engine) Success(w *Worker, tasks []CrawlerTask, outputs []MagnetEntry) {
	parentID, runID := int64(0), int64(0)
	var parent *TaskEntry
	if current, ok := w.CurrentTask().(*TaskEntry); ok {
		parent = current
		parentID, runID = parent.taskID, parent.runID
	}
	for _, t := range tasks {
		if parent != nil && parent.taskID > 0 && parent.attemptID > 0 {
			if err := task_repo.CheckAttemptID(parent.taskID, parent.attemptID); err != nil {
				return
			}
		}
		e.persistTask(t, runID, parentID)
		e.Submit(t)
	}

	for _, output := range outputs {
		if parent != nil && parent.taskID > 0 && parent.attemptID > 0 {
			if err := task_repo.CheckAttemptID(parent.taskID, parent.attemptID); err != nil {
				return
			}
		}
		var resource *table.Resource
		write := func() error {
			var err error
			resource, err = resource_repo.SaveCollected(output.Origin, output.Title, output.Number, output.Actress0,
				output.RawURLHost, output.RawURLPath, output.Links, output.OptimalLink)
			return err
		}
		var err error
		if parent != nil && parent.taskID > 0 && parent.attemptID > 0 {
			err = task_repo.WithActiveAttemptID(parent.taskID, parent.attemptID, write)
		} else {
			err = write()
		}
		if err != nil {
			log.Errorf("保存采集资源异常：%s -> %s: %s", output.Origin, output.OptimalLink, err.Error())
			continue
		}
		log.Debugf("保存采集资源：%s -> %s (id=%d, status=%s)", output.Origin, output.OptimalLink, resource.Id, resource.Status)
	}
	if parent != nil && parent.taskID > 0 && parent.attemptID > 0 {
		_ = task_repo.Complete(parent.taskID, parent.attemptID, fmt.Sprintf(`{"outputs":%d,"children":%d}`, len(outputs), len(tasks)))
	}
}

func (e *Engine) Error(w *Worker, t CrawlerTask, err error) {
	if entry, ok := t.(*TaskEntry); ok && entry.taskID > 0 && entry.attemptID > 0 {
		retry, persistErr := task_repo.Fail(entry.taskID, entry.attemptID, err, entry.ErrorNum() < MaxTaskErrorNum)
		if persistErr != nil {
			log.Errorf("持久化任务失败状态异常：%s", persistErr.Error())
		}
		if !retry {
			log.Errorf("任务进入死信：%s - %s", t.RawUrl(), err.Error())
			return
		}
		t.IncrErrorNum()
		time.AfterFunc(taskRetryDelay(t.ErrorNum()), func() { e.Submit(t) })
		return
	}
	if t.ErrorNum() >= MaxTaskErrorNum {
		log.Errorf("任务出错次数太多: %s - %s", t.RawUrl(), err.Error())
		return
	}

	t.IncrErrorNum()
	log.Errorf("任务处理异常：%s - %s", t.RawUrl(), err.Error())

	e.Submit(t)
}

func (e *Engine) persistTask(t CrawlerTask, runID, parentID int64) {
	entry, ok := t.(*TaskEntry)
	if !ok || entry.taskID > 0 || task_repo.DatabaseUnavailable() {
		return
	}
	if runID <= 0 {
		run, err := task_repo.CreateRun(entry.Origin, entry.Origin, "event", task_repo.TaskInput(entry.RawURL, entry.Origin))
		if err != nil {
			log.Warnf("创建持久化运行记录失败：%s", err.Error())
			return
		}
		runID = run.Id
	}
	task, err := task_repo.CreateTask(runID, parentID, task_repo.TaskStep(entry.RawURL), "browser", task_repo.TaskInput(entry.RawURL, entry.Origin), MaxTaskErrorNum)
	if err != nil {
		log.Warnf("创建持久化采集任务失败：%s", err.Error())
		return
	}
	entry.SetPersistence(runID, task.Id, parentID)
}

func taskRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<attempt) * time.Second
}

func (e *Engine) Stop(ctx context.Context) error {
	var wait sync.WaitGroup
	wait.Add(len(e.workers))
	for _, w := range e.workers {
		go func(w *Worker) {
			w.Close()
			wait.Done()
		}(w)
	}
	e.cancel()
	wait.Wait()
	log.Infoln("stop engine...")
	return nil
}

func (e *Engine) Snapshot() EngineSnapshot {
	e.workerLock.RLock()
	defer e.workerLock.RUnlock()
	result := EngineSnapshot{
		WorkerCount: len(e.workers),
		QueueLength: e.taskDispatcher.Len(),
		Workers:     make([]WorkerSnapshot, 0, len(e.workers)),
	}
	for _, worker := range e.workers {
		item := worker.Snapshot()
		if item.Running {
			result.Running++
		}
		result.Workers = append(result.Workers, item)
	}
	return result
}
