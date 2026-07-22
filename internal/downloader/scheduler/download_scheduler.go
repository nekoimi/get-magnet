package scheduler

import (
	"context"
	"sync/atomic"

	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/downloader"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	log "github.com/sirupsen/logrus"
)

const (
	defaultSubmitCron = "*/5 * * * *"
	defaultBatchSize  = 20
	defaultMaxRetry   = 5
)

type DownloadScheduler struct {
	cfg             *config.DownloadConfig
	cronScheduler   job.CronScheduler
	downloadService downloader.DownloadService
	running         atomic.Bool
	cancel          context.CancelFunc
}

func NewDownloadScheduler() *DownloadScheduler {
	return &DownloadScheduler{}
}

func (s *DownloadScheduler) Name() string {
	return "DownloadScheduler"
}

func (s *DownloadScheduler) Start(parent context.Context) error {
	cfg := bean.PtrFromContext[config.Config](parent)
	s.cfg = cfg.Download
	if s.cfg == nil {
		s.cfg = &config.DownloadConfig{}
	}
	if !s.cfg.Enabled {
		log.Infoln("下载调度器已禁用，跳过定时任务注册")
		return nil
	}

	s.cronScheduler = bean.FromContext[job.CronScheduler](parent)
	s.downloadService = bean.FromContext[downloader.DownloadService](parent)

	subCtx, cancel := context.WithCancel(parent)
	s.cancel = cancel

	submitCron := s.cfg.SubmitCron
	if submitCron == "" {
		submitCron = defaultSubmitCron
	}
	s.cronScheduler.Register(submitCron, &job.CronJob{
		Name: "提交待下载磁力任务",
		Exec: func() {
			s.RunOnce(subCtx)
		},
	})

	return nil
}

func (s *DownloadScheduler) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *DownloadScheduler) RunOnce(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if !s.running.CompareAndSwap(false, true) {
		log.Debugln("下载调度任务仍在执行，跳过本轮")
		return
	}
	defer s.running.Store(false)

	batchSize := s.batchSize()
	maxRetry := s.maxRetry()
	list, err := magnet_repo.ListPendingDownload(batchSize, maxRetry)
	if err != nil {
		return
	}
	if len(list) == 0 {
		log.Debugln("没有待提交下载的磁力任务")
		return
	}

	for _, m := range list {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ok, err := magnet_repo.MarkDownloadSubmitting(m.Id, maxRetry)
		if err != nil {
			continue
		}
		if !ok {
			log.Debugf("磁力任务已被其他调度领取，跳过：%d", m.Id)
			continue
		}

		taskID, err := s.downloadService.Download(m.Origin, m.OptimalLink)
		if err != nil {
			log.Errorf("提交磁力下载失败：%d - %s - %s", m.Id, m.OptimalLink, err.Error())
			_ = magnet_repo.MarkDownloadSubmitFailed(m.Id, err)
			continue
		}

		if err := magnet_repo.MarkDownloadSubmitted(m.Id, taskID); err != nil {
			log.Errorf("提交磁力下载后更新状态失败：%d - %s - %s", m.Id, taskID, err.Error())
			continue
		}
		log.Infof("提交磁力下载成功：%d - %s -> %s", m.Id, m.OptimalLink, taskID)
	}
}

func (s *DownloadScheduler) batchSize() int {
	if s.cfg == nil || s.cfg.BatchSize <= 0 {
		return defaultBatchSize
	}
	return s.cfg.BatchSize
}

func (s *DownloadScheduler) maxRetry() int {
	if s.cfg == nil || s.cfg.MaxRetry <= 0 {
		return defaultMaxRetry
	}
	return s.cfg.MaxRetry
}
