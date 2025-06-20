package aria2

import (
	"context"
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/common/model"
	"github.com/nekoimi/get-magnet/pkg/aria2_client"
	"github.com/nekoimi/get-magnet/pkg/util"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ActiveRepo struct {
	mmu       *sync.Mutex
	magnetMap map[string]struct{}
}

func NewActiveRepo() *ActiveRepo {
	return &ActiveRepo{
		mmu:       &sync.Mutex{},
		magnetMap: make(map[string]struct{}),
	}
}

func (ar *ActiveRepo) Put(gid string) {
	ar.mmu.Lock()
	defer ar.mmu.Unlock()
	ar.magnetMap[gid] = struct{}{}
}

func (ar *ActiveRepo) Del(gid string) {
	ar.mmu.Lock()
	defer ar.mmu.Unlock()
	delete(ar.magnetMap, gid)
}

func (ar *ActiveRepo) Each(callback func(gid string)) {
	ar.mmu.Lock()
	defer ar.mmu.Unlock()
	for gid := range ar.magnetMap {
		callback(gid)
	}
}

type Aria2 struct {
	client           *aria2_client.SafeClient
	magnetChan       chan *model.Item
	ar               *ActiveRepo
	zeroSpeedCounter map[string]int8
	exit             chan struct{}
}

func New(jsonrpc string, secret string) *Aria2 {
	client := aria2_client.New(jsonrpc, secret)

	aria := &Aria2{
		client:           client,
		magnetChan:       make(chan *model.Item),
		ar:               NewActiveRepo(),
		zeroSpeedCounter: make(map[string]int8),
		exit:             make(chan struct{}),
	}

	aria.client.Client().Subscribe(arigo.StartEvent, aria.startEventHandle)
	aria.client.Client().Subscribe(arigo.PauseEvent, aria.pauseEventHandle)
	aria.client.Client().Subscribe(arigo.StopEvent, aria.stopEventHandle)
	aria.client.Client().Subscribe(arigo.CompleteEvent, aria.completeEventHandle)
	aria.client.Client().Subscribe(arigo.BTCompleteEvent, aria.btCompleteEventHandle)
	aria.client.Client().Subscribe(arigo.ErrorEvent, aria.errorEventHandle)

	return aria
}

func (aria *Aria2) Submit(item *model.Item) {
	aria.magnetChan <- item
}

func (aria *Aria2) Run() {
	go aria.bestFileSelectWork()

	for {
		select {
		case item := <-aria.magnetChan:
			aria.createDownload(item)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (aria *Aria2) Stop() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		close(aria.exit)
		for len(aria.magnetChan) > 0 {
			time.Sleep(1 * time.Second)
		}
		err := aria.client.Client().Close()
		if err != nil {
			log.Printf("aria2 client close err: %s \n", err.Error())
		}
		log.Println("stop aria2 client")
		cancel()
	}()
	<-ctx.Done()
}

func (aria *Aria2) createDownload(item *model.Item) {
	magnetLink := item.OptimalLink
	log.Printf("add url to aria2: %s \n", magnetLink)
	ops, err := aria.client.Client().GetGlobalOptions()
	if err != nil {
		panic(err)
	}

	host := strings.ReplaceAll(strings.ReplaceAll(util.CleanHost(item.ResHost), ":", "_"), ".", "_")
	saveDir := ops.Dir + "/" + util.NowDate("-") + "/" + host
	g, err := aria.client.Client().AddURI(arigo.URIs(magnetLink), &arigo.Options{
		Dir: saveDir,
	})
	if err != nil {
		log.Printf("add uri (%s) to aria2 err: %s \n", magnetLink, err.Error())
		return
	}

	aria.ar.Put(g.GID)
}

func (aria *Aria2) bestFileSelectWork() {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			// Each active items
			activeStatus, err := aria.client.Client().TellActive("gid", "status", "errorCode", "errorMessage", "dir", "files", "downloadSpeed")
			if err != nil {
				log.Printf("fetch download status err: %s \n", err.Error())
				continue
			}

			for _, status := range activeStatus {
				gid := status.GID
				if status.Status != arigo.StatusActive {
					log.Printf("GID#%s Status not active: %s \n", gid, status.Status)
					continue
				}

				// 检查任务的下载速度
				currDownloadSpeed := status.DownloadSpeed
				var zeroNum int8
				if currDownloadSpeed < LowSpeedThreshold {
					if num, ok := aria.zeroSpeedCounter[gid]; !ok {
						zeroNum = 1
					} else {
						zeroNum = num + 1
					}

					if zeroNum > ZeroSpeedThreshold {
						delete(aria.zeroSpeedCounter, gid)
						// 下载速度一直为0，直接暂停该任务
						err = aria.client.Client().Pause(gid)
						if err != nil {
							log.Printf("Pause %s download status err: %s \n", gid, err.Error())
							continue
						}
						log.Printf("暂停任务：%s 下载速度一直小于 %d 字节/s\n", gid, LowSpeedThreshold)
						continue
					}

					aria.zeroSpeedCounter[gid] = zeroNum
				} else {
					if num, ok := aria.zeroSpeedCounter[gid]; ok {
						if num > 0 {
							aria.zeroSpeedCounter[gid] = num - 1
						} else {
							delete(aria.zeroSpeedCounter, gid)
						}
					}
				}

				files := status.Files
				if len(files) <= 1 {
					continue
				}

				needChangeOps := false
				for _, f := range files {
					// if selected non best, need re-change options
					if f.Selected && !IsBestFile(f) {
						needChangeOps = true
						break
					}
				}

				if needChangeOps {
					allowFiles := BestSelectFile(files)
					var builder strings.Builder
					for _, a := range allowFiles {
						builder.WriteString(strconv.Itoa(a.Index))
						builder.WriteString(",")
					}
					// selectFile, _ := strings.CutSuffix(builder.String(), ",")
					selectIndex := builder.String()
					err = aria.client.Client().ChangeOptions(gid, arigo.Options{
						SelectFile: selectIndex,
					})
					if err != nil {
						log.Printf("change GID#%s options (select-file=%s) err: %s \n", gid, selectIndex, err.Error())
						continue
					}

					log.Println("SELECT-Files: ", selectIndex)
				}
			}
		}
	}
}

func (aria *Aria2) startEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s startEventHandle\n", event.GID)
	aria.ar.Put(event.GID)
}

func (aria *Aria2) pauseEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s pauseEventHandle\n", event.GID)
	aria.ar.Del(event.GID)
}

func (aria *Aria2) stopEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s stopEventHandle\n", event.GID)
	aria.ar.Del(event.GID)
}

func (aria *Aria2) completeEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s completeEventHandle\n", event.GID)
	aria.ar.Del(event.GID)
}

func (aria *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s btCompleteEventHandle\n", event.GID)
	aria.ar.Del(event.GID)
}

func (aria *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s errorEventHandle\n", event.GID)
	aria.ar.Del(event.GID)
}
