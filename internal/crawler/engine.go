package crawler

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/downloader"
	"github.com/nekoimi/get-magnet/internal/ocr"
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	log "github.com/sirupsen/logrus"
	"modernc.org/mathutil"
	"strings"
	"sync"
	"time"
)

const (
	// MaxTaskErrorNum 任务出现错误最多重试次数
	MaxTaskErrorNum = 5
)

type Engine struct {
	// 配置文件
	cfg *Config
	// worker操作锁
	workerLock *sync.RWMutex
	// worker池
	workers []*Worker
	// 任务队列
	taskCh chan CrawlerTask
	// ocr服务
	ocrServer *ocr.Server
	// 下载器
	downloadService downloader.DownloadService
	// crawler管理器
	crawlerManager *Manager
}

func NewCrawlerEngine(cfg *Config, downloadService downloader.DownloadService, crawlerManager *Manager) *Engine {
	ocrServer := ocr.NewOcrServer(cfg.OcrBin)

	return &Engine{
		cfg:             cfg,
		workerLock:      &sync.RWMutex{},
		workers:         make([]*Worker, 0),
		taskCh:          make(chan CrawlerTask),
		ocrServer:       ocrServer,
		downloadService: downloadService,
		crawlerManager:  crawlerManager,
	}
}
func (e *Engine) Name() string {
	return "CrawlerEngine"
}

func (e *Engine) Start(ctx context.Context) error {
	// 启动OCR服务
	apptools.AutoRestart(ctx, "OCR服务", e.ocrServer.Run, 10*time.Second)

	e.workerLock.Lock()
	defer e.workerLock.Unlock()

	for i := 0; i < mathutil.Max(1, e.cfg.WorkerNum); i++ {
		w := NewWorker(ctx, i, e.taskCh, e)

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
	log.Debugf("提交task：%s", t.RawUrl())
	e.taskCh <- t
}

func (e *Engine) Success(w *Worker, tasks []CrawlerTask, outputs []MagnetEntry) {
	for _, t := range tasks {
		e.Submit(t)
	}

	for _, output := range outputs {
		m := &table.Magnets{
			Origin:      output.Origin,
			Title:       output.Title,
			Number:      strings.ToUpper(output.Number),
			OptimalLink: output.OptimalLink,
			Links:       output.Links,
			RawURLHost:  output.RawURLHost,
			RawURLPath:  output.RawURLPath,
			Status:      1,
			Actress0:    output.Actress0,
			FollowedBy:  "unknow",
		}

		// 提交下载
		log.Debugf("提交下载：%s -> %s", output.Origin, output.OptimalLink)
		id, err := e.downloadService.Download(output.Origin, output.OptimalLink)
		if err != nil {
			log.Errorf("提交下载任务异常: %s", err.Error())
			magnet_repo.Save(m)
		} else {
			m.Status = 0
			m.FollowedBy = id
			magnet_repo.Save(m)
		}
	}
}

func (e *Engine) Error(w *Worker, t CrawlerTask, err error) {
	if t.ErrorNum() >= MaxTaskErrorNum {
		log.Errorf("任务出错次数太多: %s - %s", t.RawUrl(), err.Error())
		return
	}

	t.IncrErrorNum()
	log.Errorf("任务处理异常：%s - %s", t.RawUrl(), err.Error())

	e.Submit(t)
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
	wait.Wait()
	log.Infoln("stop engine...")
	return nil
}
