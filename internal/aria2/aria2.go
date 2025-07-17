package aria2

import (
	"context"
	"errors"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/patrickmn/go-cache"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"modernc.org/mathutil"
	"runtime/debug"
	"sync"
	"time"
)

// MaxRetryConnNum 最大断线重连次数
const MaxRetryConnNum = 5

type Aria2 struct {
	ctx context.Context
	// aria2 操作锁
	amux *sync.Mutex
	// aria2 jsonrpc 客户端
	_client *arigo.Client
	// 下载任务下载速度缓存
	speedCache *cache.Cache
	// 退出
	exit chan struct{}
	// wait
	exitWG sync.WaitGroup
}

func NewClient() *Aria2 {
	return &Aria2{
		amux:       &sync.Mutex{},
		speedCache: cache.New(LowSpeedTimeout, LowSpeedCleanupInterval),
		exit:       make(chan struct{}, 1),
		exitWG:     sync.WaitGroup{},
	}
}

func (a *Aria2) Start(ctx context.Context) {
	a.ctx = ctx

	version, err := a.client().GetVersion()
	if err != nil {
		panic(err)
	}
	log.Infof("aria2版本信息: %s", version.Version)

	a.client().Subscribe(arigo.StartEvent, a.startEventHandle)
	a.client().Subscribe(arigo.PauseEvent, a.pauseEventHandle)
	a.client().Subscribe(arigo.StopEvent, a.stopEventHandle)
	a.client().Subscribe(arigo.CompleteEvent, a.completeEventHandle)
	a.client().Subscribe(arigo.BTCompleteEvent, a.btCompleteEventHandle)
	a.client().Subscribe(arigo.ErrorEvent, a.errorEventHandle)

	// 初始化处理异常任务
	offset := 0
	fetchNum := 20
	for {
		stops, err := a.client().TellStopped(offset, uint(fetchNum), "gid", "status", "infoHash", "files", "bittorrent", "errorCode", "errorMessage")
		if err != nil {
			log.Errorf("查询当前停止的下载任务信息异常: %s", err.Error())
			panic(err)
		}

		if len(stops) == 0 {
			break
		}

		for _, stop := range stops {
			log.Debugf("启动初始化任务：%s - %s", display(stop), stop.Status)
			if stop.Status == arigo.StatusError {
				// 处理异常任务
				a.onErrorFileNameTooLong(stop)
			}

			if stop.Status == arigo.StatusCompleted {
				// 检查完成的任务下载文件是否最优
				a.handleFileBestSelect(stop)
			}
		}

		offset = offset + fetchNum
	}

	// 添加更新tracker服务器job
	job.Register("10 00 * * *", &job.Job{
		Name: "更新Aria2下载tracker服务器",
		Cmd: func() {
			upgradeTrackers(a)
		},
	})

	a.runStatusLoop()
}

func (a *Aria2) Submit(origin string, downloadUrl string) error {
	ops, err := a.globalOptions()
	if err != nil {
		return err
	}

	saveDir := ops.Dir + "/" + origin + "/" + util.NowDate("-")
	if _, err = a.client().AddURI(arigo.URIs(downloadUrl), &arigo.Options{
		Dir: saveDir,
	}); err != nil {
		log.Errorf("添加aria2下载任务异常: %s", err.Error())
		return err
	}
	return nil
}

func (a *Aria2) Stop() {
	a.exit <- struct{}{}

	if a._client != nil {
		if err := a.client().Close(); err != nil {
			log.Errorf("aria2客户端关闭异常: %s", err.Error())
		}
	}

	log.Debugln("停止aria2客户端")
	a.exitWG.Wait()
}

// 任务状态检测
func (a *Aria2) runStatusLoop() {
	checkRunning := false
	a.exitWG.Add(1)
	ticker := time.NewTicker(LowSpeedInterval)
	for {
		select {
		case <-a.exit:
			log.Debugln("aria2 best file select exit.")
			a.exitWG.Done()
			return
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

				ops, err := a.globalOptions()
				if err != nil {
					panic(err)
				}

				maxDownloadNum := mathutil.Max(int(ops.MaxConcurrentDownloads), 1)
				actives, err := a.client().TellActive("gid", "status", "files", "downloadSpeed")
				if err != nil {
					log.Errorf("查询当前活跃的下载任务信息异常: %s", err.Error())
					return
				}

				if len(actives) < maxDownloadNum {
					//num := maxDownloadNum - len(tasks)
					//log.Debugf("检测到当前下载任务数量小于最大下载数量，尝试启动暂停的下载任务: size-%d ...", num)
					// 如果当前没有活跃的任务，查询已经停止的任务启动起来
					if err = a.client().UnpauseAll(); err != nil {
						log.Errorf("恢复下载任务信息异常: %s", err.Error())
					}
				}

				log.Debugf("下载文件优选：size-%d", len(actives))
				for _, act := range actives {
					if act.Status != arigo.StatusActive {
						// 下载任务不活跃，不做处理
						log.Debugf("[文件优选] 下载任务(%s)状态不活跃，不做处理：%s", display(act), act.Status)
						continue
					}
					// 下载文件优选
					a.handleFileBestSelect(act)
				}

				// 检查是否存在等待中的任务
				waits, err := a.client().TellWaiting(0, 1, "gid", "status")
				if err != nil {
					log.Errorf("查询当前等待的下载任务信息异常: %s", err.Error())
					return
				}
				if len(waits) > 0 {
					log.Debugf("检查下载任务：size-%d", len(actives))
					for _, act := range actives {
						if act.Status != arigo.StatusActive {
							// 下载任务不活跃，不做处理
							log.Debugf("[下载速度] 下载任务(%s)状态不活跃，不做处理：%s", display(act), act.Status)
							continue
						}

						if act.Status == arigo.StatusCompleted {
							// 已经完成的不做处理
							log.Debugf("[下载速度] 下载任务(%s)状态已经完成，不做处理：%s", display(act), act.Status)
							continue
						}

						gid := act.GID
						// 检查任务的下载速度
						if a.isPauseCheckDownloadSpeed(gid, act.DownloadSpeed) {
							log.Debugf("[下载速度] 下载任务(%s)低速下载，将暂停...", display(act))
							// 检查不通过，需要降低当前任务的优先级
							if err = a.client().Pause(gid); err != nil {
								log.Errorf("[下载速度] 暂停下载任务(%s)异常: %s", display(act), err.Error())
							} else {
								// 清除当前任务的下载速度缓存
								a.speedCache.Delete(gid)
								log.Infof("[下载速度] 暂停任务：(%s) 下载速度一直小于 %d 字节/s", display(act), LowSpeedThreshold)
							}
							time.Sleep(300 * time.Microsecond)
						}
					}
				}
			}()
		}
	}
}

func (a *Aria2) startEventHandle(event *arigo.DownloadEvent) {
	gid := event.GID
	log.Debugf("GID#%s startEventHandle", gid)

	// 清除下载速度缓存
	a.speedCache.Delete(event.GID)

	// 获取下载任务信息
	task, err := a.client().TellStatus(gid, "gid", "status", "files", "downloadSpeed")
	if err != nil {
		log.Errorf("查询当前(%s)下载任务信息异常: %s", gid, err.Error())
		return
	}

	// 下载文件优选
	a.handleFileBestSelect(task)
}

func (a *Aria2) pauseEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s pauseEventHandle", event.GID)

	// 清除下载速度缓存
	a.speedCache.Delete(event.GID)
}

func (a *Aria2) stopEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s stopEventHandle", event.GID)

	// 清除下载速度缓存
	a.speedCache.Delete(event.GID)
}

func (a *Aria2) completeEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s completeEventHandle", event.GID)

	// 清除下载速度缓存
	a.speedCache.Delete(event.GID)
}

func (a *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s btCompleteEventHandle", event.GID)

	// 清除下载速度缓存
	a.speedCache.Delete(event.GID)
}

func (a *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	// 清除下载速度缓存
	a.speedCache.Delete(event.GID)

	status, err := a.client().TellStatus(event.GID, "gid", "status", "infoHash", "files", "bittorrent", "errorCode", "errorMessage")
	if err != nil {
		log.Errorf("查询下载任务GID#%s状态信息异常: %s", event.GID, err.Error())
		return
	}
	log.Errorf("下载任务(%s)出错：[%s] %s - %s", display(status), status.Status, status.ErrorCode, status.ErrorMessage)

	// 处理文件出错的情况
	a.onErrorFileNameTooLong(status)
}

func (a *Aria2) client() *arigo.Client {
	a.amux.Lock()
	defer a.amux.Unlock()

	pingErr := a.ping()
	if pingErr != nil {
		exit := false
		reConnNum := 0
		for {
			select {
			case <-a.ctx.Done():
				exit = true
				log.Debugf("aria2取消重连...")
				break
			default:
				err := a.connect()
				if err != nil {
					reConnNum++
					if reConnNum > MaxRetryConnNum {
						exit = true
						break
					}
					log.Warnf("检测到aria2客户端异常, 重新连接 %d ... %s", reConnNum, err.Error())
					time.Sleep(3 * time.Second)
					continue
				}
				return a._client
			}

			if exit {
				break
			}
		}
		panic(pingErr)
	}

	return a._client
}

func (a *Aria2) ping() error {
	if a._client == nil {
		return errors.New("aria2客户端未初始化连接")
	}

	_, err := a._client.GetVersion()
	return err
}

func (a *Aria2) connect() error {
	cfg := config.Get()
	client, err := arigo.Dial(cfg.Aria2Ops.JsonRpc, cfg.Aria2Ops.Secret)
	if err != nil {
		return err
	}
	a._client = client
	return nil
}

func (a *Aria2) globalOptions() (arigo.Options, error) {
	ops, err := a.client().GetGlobalOptions()
	if err != nil {
		log.Errorf("查询aria2全局配置异常: %s - %s", err.Error(), debug.Stack())
		return arigo.Options{}, err
	}
	return ops, nil
}
