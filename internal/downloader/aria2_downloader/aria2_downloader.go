package aria2_downloader

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/downloader"
	"github.com/nekoimi/get-magnet/internal/downloader/aria2_downloader/tracker"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"time"
)

type Aria2Downloader struct {
	// context
	ctx context.Context
	// 配置信息
	cfg *Config
	// aria2 客户端
	client *Client
	// 下载完成回调
	onComplete []downloader.DownloadCallback
	// 下载失败回调
	onError []downloader.DownloadCallback
	// 定时任务调度
	cronScheduler job.CronScheduler
}

func NewAria2DownloadService(ctx context.Context, cfg *Config, cronScheduler job.CronScheduler) downloader.DownloadService {
	d := &Aria2Downloader{
		ctx:           ctx,
		cfg:           cfg,
		onComplete:    make([]downloader.DownloadCallback, 0),
		onError:       make([]downloader.DownloadCallback, 0),
		cronScheduler: cronScheduler,
	}

	d.initializeAria2Client()

	return d
}

func (d *Aria2Downloader) initializeAria2Client() {
	ctx, cancel := context.WithCancel(d.ctx)
	d.client = newAria2Client(ctx, d.cfg)

	// 注册定时更新tracker服务器任务
	d.cronScheduler.Register("10 00 * * *", &job.CronJob{
		Name: "更新Aria2下载tracker服务器",
		Exec: func() {
			btTrackers := tracker.FetchTrackers()
			d.client.UpdateTrackers(btTrackers)
		},
	})

	go func() {
		// 启动aria2连接
		apptools.AutoRestart(ctx, "aria2客户端", d.client.initialize, 10*time.Second)

		for {
			select {
			case <-d.ctx.Done():
				cancel()
				d.client.Close()
				return
			case e := <-d.client.eventCh:
				log.Debugf("接收到aria2事件: %s", e.Type)
				t := downloader.DownloadTask{
					Id:    e.Id(),
					Name:  e.Name(),
					Files: e.Files(),
				}
				switch e.Type {
				case arigo.CompleteEvent, arigo.BTCompleteEvent:
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
	}()
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
