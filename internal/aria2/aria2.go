package aria2

import (
	"errors"
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MaxRetryConnNum 最大断线重连次数
const MaxRetryConnNum = 5

type Aria2 struct {
	// aria2 操作锁
	amux *sync.Mutex
	// aria2 jsonrpc 客户端
	_client *arigo.Client
	// 下载任务下载速度缓存
	speedCache *cache.Cache
	// 当前活跃的下载任务GID列表
	activeRepo *ActiveRepo
	// 退出
	exit chan struct{}
	// wait
	exitWG sync.WaitGroup
}

func NewClient() *Aria2 {
	return &Aria2{
		amux:       &sync.Mutex{},
		speedCache: cache.New(LowSpeedTimeout, LowSpeedCleanupInterval),
		activeRepo: newActiveRepo(),
		exit:       make(chan struct{}, 1),
		exitWG:     sync.WaitGroup{},
	}
}

func (a *Aria2) Start() {
	version, err := a.client().GetVersion()
	if err != nil {
		panic(err)
	}
	log.Infof("aria2版本信息: %s\n", version.Version)

	a.client().Subscribe(arigo.StartEvent, a.startEventHandle)
	a.client().Subscribe(arigo.PauseEvent, a.pauseEventHandle)
	a.client().Subscribe(arigo.StopEvent, a.stopEventHandle)
	a.client().Subscribe(arigo.CompleteEvent, a.completeEventHandle)
	a.client().Subscribe(arigo.BTCompleteEvent, a.btCompleteEventHandle)
	a.client().Subscribe(arigo.ErrorEvent, a.errorEventHandle)

	// 初始化获取的任务
	actives, err := a.client().TellActive("gid")
	if err != nil {
		log.Errorf("查询当前活跃的下载任务信息异常: %s \n", err.Error())
		panic(err)
	}

	for _, active := range actives {
		a.activeRepo.put(active.GID)
		log.Debugf("init active task: %s\n", active.GID)
	}

	a.bestFileSelectWork()
}

func (a *Aria2) Submit(downloadUrl string) error {
	return a.BatchSubmit([]string{downloadUrl})
}

func (a *Aria2) BatchSubmit(downloadUrls []string) error {
	ops, err := a.client().GetGlobalOptions()
	if err != nil {
		log.Errorf("查询aria2全局配置异常: %s - %s\n", err.Error(), debug.Stack())
		return err
	}

	saveDir := ops.Dir + "/" + util.NowDate("-")
	if _, err = a.client().AddURI(arigo.URIs(downloadUrls...), &arigo.Options{
		Dir: saveDir,
	}); err != nil {
		log.Errorf("添加aria2下载任务异常: %s \n", err.Error())
		return err
	}
	return nil
}

func (a *Aria2) Stop() {
	a.exit <- struct{}{}

	if a._client != nil {
		err := a.client().Close()
		if err != nil {
			log.Errorf("aria2客户端关闭异常: %s \n", err.Error())
		}
	}

	log.Debugln("停止aria2客户端")
	a.exitWG.Wait()
}

// 下载文件优选
func (a *Aria2) bestFileSelectWork() {
	a.exitWG.Add(1)
	ticker := time.NewTicker(LowSpeedInterval)
	for {
		select {
		case <-a.exit:
			log.Debugln("aria2 best file select exit.")
			a.exitWG.Done()
			return
		case <-ticker.C:
			if a.activeRepo.isEmpty() {
				// 如果当前没有活跃的任务，查询已经停止的任务启动起来
				tasks, err := a.client().TellWaiting(0, 30, "gid", "status")
				if err != nil {
					log.Errorf("查询等待的下载任务信息异常: %s \n", err.Error())
					continue
				}
				for _, task := range tasks {
					if task.Status == arigo.StatusPaused {
						err = a.client().Unpause(task.GID)
						if err != nil {
							log.Errorf("恢复下载任务(%s)异常: %s \n", task.GID, err.Error())
							continue
						}
						log.Infof("恢复下载任务(%s)\n", task.GID)
					}
				}
			} else {
				a.activeRepo.each(func(gid string, createdAt time.Time) {
					status, err := a.client().TellStatus(gid, "status", "files", "downloadSpeed")
					if err != nil {
						log.Errorf("查询下载任务GID#%s状态信息异常: %s \n", gid, err.Error())
						return
					}

					if status.Status != arigo.StatusActive {
						// 下载任务不活跃，不做处理
						return
					}

					// 检查任务的下载速度
					if a.isPauseCheckDownloadSpeed(gid, status.DownloadSpeed) {
						// 检查不通过，需要降低当前任务的优先级
						err = a.client().Pause(gid)
						if err != nil {
							log.Errorf("暂停下载任务(%s)异常: %s \n", gid, err.Error())
							return
						}
						log.Infof("暂停任务：%s 下载速度一直小于 %d 字节/s\n", gid, LowSpeedThreshold)
						return
					}

					// 下载文件优选
					files := status.Files
					if len(files) <= 1 {
						// 只有一个文件，不做处理
						return
					}

					needChangeOps := false
					for _, f := range files {
						// if selected non best, need re-change options
						if f.Selected && !isBestFile(f) {
							needChangeOps = true
							break
						}
					}

					if needChangeOps {
						allowFiles := bestSelectFile(files)
						if len(allowFiles) == 0 {
							err = a.client().Pause(gid)
							if err != nil {
								log.Errorf("暂停下载任务(%s)异常: %s \n", gid, err.Error())
								return
							}
							log.Debugf("下载任务没有优选出符合要求的文件：%s\n", gid)
							return
						}

						var builder strings.Builder
						for _, a := range allowFiles {
							builder.WriteString(strconv.Itoa(a.Index))
							builder.WriteString(",")
						}
						selectIndex := builder.String()
						err = a.client().ChangeOptions(gid, arigo.Options{
							SelectFile: selectIndex,
						})
						if err != nil {
							log.Errorf("下载任务(%s)文件优选异常: %s \n", gid, err.Error())
							return
						}
					}
				})
			}
		}
	}
}

func (a *Aria2) startEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s startEventHandle\n", event.GID)
	a.activeRepo.put(event.GID)
}

func (a *Aria2) pauseEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s pauseEventHandle\n", event.GID)
	a.activeRepo.del(event.GID)
}

func (a *Aria2) stopEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s stopEventHandle\n", event.GID)
	a.activeRepo.del(event.GID)
}

func (a *Aria2) completeEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s completeEventHandle\n", event.GID)
	a.activeRepo.del(event.GID)
}

func (a *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	log.Debugf("GID#%s btCompleteEventHandle\n", event.GID)
	a.activeRepo.del(event.GID)
}

func (a *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	a.activeRepo.del(event.GID)
	status, err := a.client().TellStatus(event.GID, "status", "errorCode", "errorMessage")
	if err != nil {
		log.Errorf("查询下载任务GID#%s状态信息异常: %s \n", event.GID, err.Error())
		return
	}
	log.Errorf("下载任务GID#%s出错：[%s] %s - %s\n", event.GID, status.Status, status.ErrorCode, status.ErrorMessage)
}

func (a *Aria2) client() *arigo.Client {
	a.amux.Lock()
	defer a.amux.Unlock()

	pingErr := a.ping()
	if pingErr != nil {
		reConnNum := 0
		for {
			err := a.connect()
			if err != nil {
				reConnNum++
				if reConnNum > MaxRetryConnNum {
					break
				}
				log.Warnf("检测到aria2客户端异常, 重新连接 %d ... %s\n", reConnNum, err.Error())
				time.Sleep(5 * time.Second)
				continue
			}
			return a._client
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
