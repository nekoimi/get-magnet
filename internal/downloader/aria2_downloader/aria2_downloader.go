package aria2_downloader

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/core"
	"github.com/nekoimi/get-magnet/internal/downloader"
	"github.com/nekoimi/get-magnet/internal/downloader/aria2_downloader/tracker"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"time"
)

type Aria2Downloader struct {
	// 配置信息
	cfg *config.Aria2Config
	// aria2 客户端
	client *Client
	// 下载完成回调
	onComplete []downloader.DownloadCallback
	// 下载失败回调
	onError []downloader.DownloadCallback
	// 定时任务调度
	cronScheduler job.CronScheduler
	// cancel
	cancel context.CancelFunc
}

func NewAria2DownloadService() downloader.DownloadService {
	return &Aria2Downloader{
		onComplete: make([]downloader.DownloadCallback, 0),
		onError:    make([]downloader.DownloadCallback, 0),
	}
}

func (d *Aria2Downloader) Name() string {
	return "Aria2Downloader"
}

func (d *Aria2Downloader) Start(parent context.Context) error {
	cfg := core.PtrFromContext[config.Config](parent)
	d.cfg = cfg.Aria2
	d.cronScheduler = core.FromContext[job.CronScheduler](parent)

	var subCtx context.Context
	subCtx, d.cancel = context.WithCancel(parent)
	d.client = newAria2Client(subCtx, d.cfg)

	// 注册定时更新tracker服务器任务
	d.cronScheduler.Register("10 00 * * *", &job.CronJob{
		Name: "更新Aria2下载tracker服务器",
		Exec: func() {
			btTrackers := tracker.FetchTrackers()
			d.client.UpdateTrackers(btTrackers)
		},
	})

	// 启动aria2连接
	apptools.AutoRestart(subCtx, "aria2客户端", d.client.initialize, 10*time.Second)

	for {
		select {
		case <-subCtx.Done():
			return d.client.Close()
		case e := <-d.client.eventCh:
			log.Debugf("接收到aria2事件: %s", e.Type)
			t := downloader.DownloadTask{
				Id:    e.Id(),
				Name:  e.Name(),
				Files: e.Files(),
			}
			switch e.Type {
			case arigo.CompleteEvent, arigo.BTCompleteEvent:
				// 文件下载完成删除不需要的文件
				handleDownloadCompleteDelFile(e.taskStatus)
				// 文件下载完成移动文件
				handleDownloadCompleteMoveFile(e.taskStatus, "JavDB", d.cfg.MoveTo.JavDBDir)

				// 处理其他回调
				for _, callback := range d.onComplete {
					go func(call downloader.DownloadCallback) {
						defer func() {
							if r := recover(); r != nil {
								log.Errorf("处理下载完成回调panic: %v", r)
							}
						}()

						call(t)
					}(callback)
				}
			case arigo.ErrorEvent:
				// 处理内置的名称错误
				d.client.handleFileNameTooLongError(e.taskStatus)

				// 处理其他回调
				for _, callback := range d.onError {
					go func(call downloader.DownloadCallback) {
						defer func() {
							if r := recover(); r != nil {
								log.Errorf("处理下载错误回调panic: %v", r)
							}
						}()

						call(t)
					}(callback)
				}
			}
		}
	}
}

func (d *Aria2Downloader) Stop(ctx context.Context) error {
	d.cancel()
	return d.client.Close()
}

func (d *Aria2Downloader) Download(category string, url string) (string, error) {
	return d.client.Submit(category, url)
}

func (d *Aria2Downloader) OnComplete(callback downloader.DownloadCallback) {
	d.onComplete = append(d.onComplete, callback)
}

func (d *Aria2Downloader) OnError(callback downloader.DownloadCallback) {
	d.onError = append(d.onError, callback)
}
