package crawler

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/aria2"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/crawler/worker"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/ocr"
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	log "github.com/sirupsen/logrus"
	"runtime/debug"
	"strings"
	"time"
)

const (
	// 任务出现错误最多重试次数
	taskErrorMax = 5
)

type Engine struct {
	ctx    context.Context
	cancel context.CancelFunc
	// aria2rpc 客户端
	aria2 *aria2.Aria2
	// ocr 服务
	ocr *ocr.Server
	// 任务调度器
	scheduler *Scheduler
}

// New create new Engine instance
func New() *Engine {
	e := &Engine{
		aria2: aria2.NewClient(),
		ocr:   ocr.NewServer(),
	}

	bus.Event().Subscribe(bus.Download.String(), e.submitDownload)

	return e
}

// Start Engine
func (e *Engine) Start() {
	ctx, cancelFunc := context.WithCancel(context.Background())

	e.ctx = ctx
	e.cancel = cancelFunc
	e.scheduler = newScheduler(ctx, e)

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("engine运行异常: %v, %s\n", r, string(debug.Stack()))
		}
	}()

	// 启动aria2连接
	apptools.AutoRestart(ctx, "aria2客户端", e.aria2.Start, 10*time.Second)
	// 启动OCR服务
	apptools.AutoRestart(ctx, "OCR服务", e.ocr.Start, 10*time.Second)
	// 启动任务生成
	apptools.DelayStart("任务生成", startTaskSeeders, 10*time.Second)
	// 启动任务调度器
	e.scheduler.Start()
}

// 添加下载任务
func (e *Engine) submitDownload(origin string, downloadUrl string) {
	if err := e.aria2.Submit(origin, downloadUrl); err != nil {
		log.Errorf("提交下载任务异常：%s - %s\n", downloadUrl, err.Error())
	} else {
		log.Infof("提交下载任务：%s\n", downloadUrl)
	}
}

func (e *Engine) Success(w *worker.Worker, tasks []task.Task, outputs []task.MagnetEntry) {
	for _, t := range tasks {
		e.scheduler.Submit(t)
	}

	for _, output := range outputs {
		_, err := db.Instance().InsertOne(&table.Magnets{
			Origin:      output.Origin,
			Title:       output.Title,
			Number:      strings.ToUpper(output.Number),
			OptimalLink: output.OptimalLink,
			Links:       output.Links,
			RawURLHost:  output.RawURLHost,
			RawURLPath:  output.RawURLPath,
			Status:      0,
		})
		if err != nil {
			log.Errorf("保存数据异常: %s \n", err.Error())
		}

		// 提交下载
		log.Debugf("提交下载：%s -> %s", output.Origin, output.OptimalLink)
		e.submitDownload(output.Origin, output.OptimalLink)
	}
}

func (e *Engine) Error(w *worker.Worker, t task.Task, err error) {
	if t.ErrorNum() >= taskErrorMax {
		log.Errorf("任务出错次数太多: %s - %s\n", t.RawUrl(), err.Error())
		return
	}

	log.Errorf("任务处理异常：%s - %s\n", t.RawUrl(), err.Error())

	e.scheduler.Submit(t)
}

// Stop shutdown engine
func (e *Engine) Stop() {
	e.cancel()
	e.scheduler.Stop()
	e.aria2.Stop()
	e.ocr.Stop()
	log.Debugf("stop engine")
}
