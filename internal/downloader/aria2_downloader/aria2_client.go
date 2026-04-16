package aria2_downloader

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/downloader/aria2_downloader/speed"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"modernc.org/mathutil"
)

const (
	// MaxRetryConnNum 最大断线重连次数
	MaxRetryConnNum = 5

	// ChannelBufferSize 事件通道缓冲区大小
	EventChannelBufferSize = 128

	// FileSelectChannelBufferSize 文件选择通道缓冲区大小
	FileSelectChannelBufferSize = 32

	// DownloadSpeedChannelBufferSize 下载速度通道缓冲区大小
	DownloadSpeedChannelBufferSize = 32

	// FetchBatchSize 每次获取任务数量
	FetchBatchSize = 20

	// InitialRetryDelay 初始重连延迟
	InitialRetryDelay = 1 * time.Second

	// MaxRetryDelay 最大重连延迟
	MaxRetryDelay = 30 * time.Second
)

var (
	// QueryArgs json rpc查询参数
	queryArgs = []string{
		"gid", "status", "files", "downloadSpeed", "infoHash", "bittorrent", "followedBy", "errorCode", "errorMessage",
	}
)

type Client struct {
	// context
	ctx context.Context
	// 配置信息
	cfg *config.Aria2Config
	// aria2 json rpc 客户端
	arigoClient *arigo.Client
	// client once
	closeOnce sync.Once
	// aria2 客户端操作锁
	clientMux *sync.Mutex
	// 事件处理chan
	eventCh chan Event
	// 文件优选chan
	fileSelectCh chan arigo.Status
	// 下载速度检测chan
	downloadSpeedCh chan arigo.Status
	// 下载速度管理器
	downloadSpeedManager *speed.Manager
}

func newAria2Client(ctx context.Context, cfg *config.Aria2Config) *Client {
	sm := speed.NewSpeedManager()

	return &Client{
		ctx:                  ctx,
		cfg:                  cfg,
		closeOnce:            sync.Once{},
		clientMux:            &sync.Mutex{},
		eventCh:              make(chan Event, EventChannelBufferSize),
		fileSelectCh:         make(chan arigo.Status, FileSelectChannelBufferSize),
		downloadSpeedCh:      make(chan arigo.Status, DownloadSpeedChannelBufferSize),
		downloadSpeedManager: sm,
	}
}

func (c *Client) initialize() {
	version, err := c.client().GetVersion()
	if err != nil {
		panic(err)
	}
	log.Infof("aria2版本信息: %s", version.Version)

	// 初始化事件订阅
	c.initializeEventSubscriptions()

	// 处理已停止的任务
	c.processStoppedTasks()

	c.Loop()
}

// initializeEventSubscriptions 订阅所有 aria2 事件
func (c *Client) initializeEventSubscriptions() {
	eventTypes := []struct {
		t       arigo.EventType
		name    string
		handler func(arigo.EventType, *arigo.DownloadEvent)
	}{
		{arigo.StartEvent, "Start", c.handleEvent},
		{arigo.PauseEvent, "Pause", c.handleEvent},
		{arigo.StopEvent, "Stop", c.handleEvent},
		{arigo.CompleteEvent, "Complete", c.handleEvent},
		{arigo.BTCompleteEvent, "BTComplete", c.handleEvent},
		{arigo.ErrorEvent, "Error", c.handleEvent},
	}

	for _, e := range eventTypes {
		// 使用局部变量避免闭包捕获循环变量问题
		eventType := e.t
		eventName := e.name
		eventHandler := e.handler

		c.client().Subscribe(eventType, func(event *arigo.DownloadEvent) {
			log.Debugf("收到aria2事件: %s", eventName)
			eventHandler(eventType, event)
		})
	}
}

// processStoppedTasks 处理已停止的任务（错误或完成）
func (c *Client) processStoppedTasks() {
	offset := 0
	for {
		stops := c.FetchStopped(offset, uint(FetchBatchSize))
		if len(stops) == 0 {
			break
		}

		for _, stop := range stops {
			log.Debugf("启动初始化任务：%s - %s", friendly(stop), stop.Status)
			if stop.Status == arigo.StatusError {
				c.eventCh <- Event{
					Type:       arigo.ErrorEvent,
					taskStatus: stop,
				}
			}

			if stop.Status == arigo.StatusCompleted && c.shouldEmitCompleteEvent(stop.GID) {
				c.eventCh <- Event{
					Type:       arigo.CompleteEvent,
					taskStatus: stop,
				}
			}
		}

		offset = offset + FetchBatchSize
	}
}

func (c *Client) shouldEmitCompleteEvent(gid string) bool {
	m, exists := magnet_repo.GetByFollowed(gid)
	if !exists {
		return true
	}
	return !m.PostProcessDone
}

func (c *Client) Loop() {
	checkRunning := false
	ticker := time.NewTicker(speed.LowSpeedInterval)
	for {
		select {
		case <-c.ctx.Done():
			log.Debugln("aria2 check loop exit...")
			return
		case s := <-c.fileSelectCh:
			// 下载文件优选
			c.handleFileBestSelect(s)
		case s := <-c.downloadSpeedCh:
			// 下载速度检查
			c.checkDownloadSpeed(s)
		case <-ticker.C:
			if checkRunning {
				log.Debugln("正在检测中，跳过执行...")
				continue
			}

			func() {
				checkRunning = true
				defer func() {
					checkRunning = false

					if r := recover(); r != nil {
						log.Errorf("检查下载状态 panic: %v", r)
					}
				}()

				c.processActiveDownloads()
			}()
		}
	}
}

// checkDownloadSpeed 检查下载速度，如果过慢则暂停任务
func (c *Client) checkDownloadSpeed(status arigo.Status) {
	gid := status.GID
	if c.downloadSpeedManager.LowSpeedDownloadCheck(status) {
		log.Debugf("[下载速度] 下载任务(%s)低速下载，将暂停...", friendly(status))
		// 检查不通过，需要降低当前任务的优先级
		if err := c.client().Pause(gid); err != nil {
			log.Errorf("[下载速度] 暂停下载任务(%s)异常: %s", friendly(status), err.Error())
		} else {
			// 清除当前任务的下载速度缓存
			c.downloadSpeedManager.Clean(gid)
			log.Infof("[下载速度] 暂停任务：(%s) 下载速度一直小于 %d 字节/s", friendly(status), speed.LowSpeedThreshold)
		}
	}
}

// processActiveDownloads 处理活跃的下载任务
func (c *Client) processActiveDownloads() {
	ops, err := c.globalOptions()
	if err != nil {
		panic(err)
	}

	maxDownloadNum := mathutil.Max(int(ops.MaxConcurrentDownloads), 1)
	actives := c.FetchActive()

	if len(actives) == 0 {
		// ignore
		return
	}

	if len(actives) < maxDownloadNum {
		// 如果当前没有活跃的任务，查询已经停止的任务启动起来
		if err = c.client().UnpauseAll(); err != nil {
			log.Errorf("恢复下载任务信息异常: %s", err.Error())
		}
	}

	// 处理文件选择
	c.processFileSelection(actives)

	// 检查是否存在等待中的任务，如果有则检查速度
	waits := c.FetchWaiting(0, 1)
	if len(waits) > 0 {
		c.processSpeedCheck(actives)
	}
}

// processFileSelection 处理文件选择
func (c *Client) processFileSelection(actives []arigo.Status) {
	log.Debugf("下载文件优选：size-%d", len(actives))
	for _, active := range actives {
		if active.Status != arigo.StatusActive {
			// 下载任务不活跃，不做处理
			continue
		}

		c.fileSelectCh <- active
	}
}

// processSpeedCheck 处理速度检查
func (c *Client) processSpeedCheck(actives []arigo.Status) {
	log.Debugf("检查下载任务：size-%d", len(actives))
	for _, active := range actives {
		if active.Status != arigo.StatusActive {
			// 下载任务不活跃，不做处理
			continue
		}

		if active.Status == arigo.StatusCompleted {
			// 已经完成的不做处理
			continue
		}

		c.downloadSpeedCh <- active
	}
}

func (c *Client) GetStatus(gid string) (arigo.Status, bool) {
	status, err := c.client().TellStatus(gid, queryArgs...)
	if err != nil {
		log.Errorf("查询下载任务信息异常: %s -> %s", gid, err.Error())
		return arigo.Status{}, false
	}
	return status, true
}

func (c *Client) FetchStopped(offset int, num uint) (result []arigo.Status) {
	status, err := c.client().TellStopped(offset, num, queryArgs...)
	if err != nil {
		log.Errorf("查询当前停止的下载任务信息列表异常: %s", err.Error())
		return result
	}
	return status
}

func (c *Client) FindStoppedByGID(gid string) (arigo.Status, bool) {
	offset := 0
	for {
		stops := c.FetchStopped(offset, uint(FetchBatchSize))
		if len(stops) == 0 {
			return arigo.Status{}, false
		}

		for _, stop := range stops {
			if stop.GID == gid {
				return stop, true
			}
		}

		offset += FetchBatchSize
	}
}

func (c *Client) FetchActive() (result []arigo.Status) {
	status, err := c.client().TellActive(queryArgs...)
	if err != nil {
		log.Errorf("查询当前活动的下载任务信息列表异常: %s", err.Error())
		return result
	}
	return status
}

func (c *Client) FetchWaiting(offset int, num uint) (result []arigo.Status) {
	status, err := c.client().TellWaiting(offset, num, queryArgs...)
	if err != nil {
		log.Errorf("查询当前等待的下载任务信息列表异常: %s", err.Error())
		return result
	}
	return status
}

func (c *Client) Submit(category string, downloadUrl string) (string, error) {
	ops, err := c.globalOptions()
	if err != nil {
		return "", err
	}

	saveDir := filepath.Join(ops.Dir, category, util.NowDate("-"))
	if gid, err := c.client().AddURI(arigo.URIs(downloadUrl), &arigo.Options{
		Dir: saveDir,
	}); err != nil {
		return "", fmt.Errorf("添加aria2下载任务异常: %s", err.Error())
	} else {
		return gid.GID, nil
	}
}

func (c *Client) UpdateTrackers(btTrackers string) {
	if err := c.client().ChangeGlobalOptions(arigo.Options{
		BTTracker: btTrackers,
	}); err != nil {
		log.Errorf("更新aria2最新tracker服务器信息异常：%s", err.Error())
	}
}

func (c *Client) client() *arigo.Client {
	c.clientMux.Lock()
	defer c.clientMux.Unlock()

	err := c.ping()
	if err != nil {
		if err = c.reconnect(); err != nil {
			panic(err)
		}
	}

	return c.arigoClient
}

// reconnect 使用指数退避策略重新连接 aria
func (c *Client) reconnect() error {
	delay := InitialRetryDelay
	hadClient := c.arigoClient != nil

	for reConnNum := 0; reConnNum < MaxRetryConnNum; reConnNum++ {
		select {
		case <-c.ctx.Done():
			log.Debugf("aria2取消重连...")
			return c.ctx.Err()
		default:
		}

		err := c.connect()
		if err == nil {
			if hadClient {
				c.initializeEventSubscriptions()
				c.processStoppedTasks()
			}
			log.Infof("aria2重连成功")
			return nil
		}

		if reConnNum < MaxRetryConnNum-1 {
			log.Warnf("aria2重连失败(%d/%d), %.1f秒后重试: %s",
				reConnNum+1, MaxRetryConnNum, delay.Seconds(), err.Error())
			select {
			case <-c.ctx.Done():
				log.Debugf("aria2取消重连...")
				return c.ctx.Err()
			case <-time.After(delay):
			}

			// 指数退避：每次延迟增加 50%，最大不超过 MaxRetryDelay
			delay = time.Duration(float64(delay) * 1.5)
			if delay > MaxRetryDelay {
				delay = MaxRetryDelay
			}
		}
	}

	return errors.New("aria2重连失败，超过最大重试次数")
}

func (c *Client) ping() error {
	if c.arigoClient == nil {
		return errors.New("aria2客户端未初始化连接")
	}

	_, err := c.arigoClient.GetVersion()
	return err
}

func (c *Client) connect() error {
	client, err := arigo.Dial(c.cfg.JsonRpc, c.cfg.Secret)
	if err != nil {
		return err
	}
	c.arigoClient = client
	return nil
}

func (c *Client) globalOptions() (arigo.Options, error) {
	ops, err := c.client().GetGlobalOptions()
	if err != nil {
		log.Errorf("查询aria2全局配置异常: %s - %s", err.Error(), debug.Stack())
		return arigo.Options{}, err
	}
	return ops, nil
}

func (c *Client) Close() error {
	if c.arigoClient != nil {
		c.closeOnce.Do(func() {
			if err := c.client().Close(); err != nil {
				log.Errorf("aria2客户端关闭异常: %s", err.Error())
			}
		})
	}
	return nil
}
