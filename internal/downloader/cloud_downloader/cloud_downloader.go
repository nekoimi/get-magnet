package cloud_downloader

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/downloader"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	log "github.com/sirupsen/logrus"
)

const pendingPostProcessLimit = 100

type CloudDownloader struct {
	cfg           *config.CloudDriverConfig
	client        *cloudClient
	onComplete    []downloader.DownloadCallback
	onError       []downloader.DownloadCallback
	cronScheduler job.CronScheduler
	cancel        context.CancelFunc
}

func NewCloudDownloadService() downloader.DownloadService {
	return &CloudDownloader{
		onComplete: make([]downloader.DownloadCallback, 0),
		onError:    make([]downloader.DownloadCallback, 0),
	}
}

func (d *CloudDownloader) Name() string {
	return "CloudDownloader"
}

func (d *CloudDownloader) Start(parent context.Context) error {
	cfg := bean.PtrFromContext[config.Config](parent)
	d.cfg = cfg.CloudDriver
	if d.cfg == nil {
		d.cfg = &config.CloudDriverConfig{}
	}
	d.cronScheduler = bean.FromContext[job.CronScheduler](parent)
	d.client = newCloudClient(d.cfg)

	var subCtx context.Context
	subCtx, d.cancel = context.WithCancel(parent)

	if err := d.client.health(subCtx); err != nil {
		log.Warnf("网盘中间服务健康检查失败，后续将通过定时任务重试: %s", err.Error())
	} else {
		log.Infof("网盘中间服务健康检查成功")
	}

	pollCron := d.cfg.PollCron
	if pollCron == "" {
		pollCron = "*/10 * * * *"
	}
	d.cronScheduler.Register(pollCron, &job.CronJob{
		Name: "轮询网盘离线下载任务",
		Exec: func() {
			d.pollPendingTasks(subCtx)
		},
	})

	time.AfterFunc(10*time.Second, func() {
		d.pollPendingTasks(subCtx)
	})

	return nil
}

func (d *CloudDownloader) Stop(ctx context.Context) error {
	if d.cancel != nil {
		d.cancel()
	}
	return nil
}

func (d *CloudDownloader) Download(category string, rawURL string) (string, error) {
	savePath := path.Join(d.cfg.SaveRoot, category, util.NowDate("-"))
	resp, err := d.client.addOfflineTask(context.Background(), addOfflineTaskRequest{
		URL:      rawURL,
		Category: category,
		SavePath: savePath,
		Metadata: map[string]string{
			"origin": category,
		},
	})
	if err != nil {
		return "", err
	}
	log.Infof("提交网盘离线下载任务成功: %s -> %s", rawURL, resp.TaskID)
	return resp.TaskID, nil
}

func (d *CloudDownloader) OnComplete(callback downloader.DownloadCallback) {
	d.onComplete = append(d.onComplete, callback)
}

func (d *CloudDownloader) OnError(callback downloader.DownloadCallback) {
	d.onError = append(d.onError, callback)
}

func (d *CloudDownloader) pollPendingTasks(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	list, err := magnet_repo.ListPendingPostProcess(pendingPostProcessLimit)
	if err != nil {
		return
	}

	for _, m := range list {
		select {
		case <-ctx.Done():
			return
		default:
		}

		taskID := m.FollowedBy
		task, err := d.client.getOfflineTask(ctx, taskID)
		if err != nil {
			log.Errorf("查询网盘离线下载任务异常: %s - %s", taskID, err.Error())
			continue
		}

		log.Debugf("网盘离线下载任务状态: %s - %s", taskID, task.Status)
		switch strings.ToLower(task.Status) {
		case "completed":
			d.handleComplete(task)
		case "failed", "canceled":
			d.handleError(task)
		}
	}
}

func (d *CloudDownloader) handleComplete(task offlineTask) {
	if err := magnet_repo.MarkPostProcessDoneByFollowed(task.TaskID); err != nil {
		log.Errorf("网盘离线下载任务完成后处理失败: %s - %s", task.TaskID, err.Error())
		return
	}

	t := downloader.DownloadTask{
		Id:    task.TaskID,
		Name:  task.Name,
		Files: taskFilePaths(task),
	}
	d.emitComplete(t)
}

func (d *CloudDownloader) handleError(task offlineTask) {
	t := downloader.DownloadTask{
		Id:    task.TaskID,
		Name:  task.Name,
		Files: taskFilePaths(task),
	}
	if task.ErrorMessage != "" {
		log.Errorf("网盘离线下载任务失败: %s - %s", task.TaskID, task.ErrorMessage)
	} else {
		log.Errorf("网盘离线下载任务失败: %s - %s", task.TaskID, task.Status)
	}
	d.emitError(t)
}

func (d *CloudDownloader) emitComplete(task downloader.DownloadTask) {
	for _, callback := range d.onComplete {
		go safeCallback("网盘下载完成", callback, task)
	}
}

func (d *CloudDownloader) emitError(task downloader.DownloadTask) {
	for _, callback := range d.onError {
		go safeCallback("网盘下载异常", callback, task)
	}
}

func safeCallback(name string, callback downloader.DownloadCallback, task downloader.DownloadTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("处理%s回调panic: %v", name, r)
		}
	}()
	callback(task)
}

func taskFilePaths(task offlineTask) []string {
	files := make([]string, 0, len(task.Files))
	for _, file := range task.Files {
		if file.Path != "" {
			files = append(files, file.Path)
			continue
		}
		if file.Name != "" {
			files = append(files, file.Name)
		}
	}
	return files
}
