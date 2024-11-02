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

type Aria2 struct {
	client *aria2_client.SafeClient

	magnetChan chan *model.Item

	mmu       *sync.Mutex
	magnetMap map[string]struct{}

	exit chan struct{}
}

func New(jsonrpc string, secret string) *Aria2 {
	client := aria2_client.New(jsonrpc, secret)

	aria := &Aria2{
		client:     client,
		magnetChan: make(chan *model.Item),
		mmu:        &sync.Mutex{},
		magnetMap:  make(map[string]struct{}),
		exit:       make(chan struct{}),
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

	aria.mmu.Lock()
	defer aria.mmu.Unlock()
	aria.magnetMap[g.GID] = struct{}{}
}

func (aria *Aria2) bestFileSelectWork() {
	for {
		time.Sleep(30 * time.Second)
		select {
		case <-aria.exit:
			return
		default:
		}

		for magnetId := range aria.magnetMap {
			status, err := aria.client.Client().TellStatus(magnetId, "status", "errorCode", "errorMessage", "dir", "files")
			if err != nil {
				log.Printf("fetch GID#%s download status err: %s \n", magnetId, err.Error())
				continue
			}

			if status.Status != arigo.StatusActive {
				log.Printf("GID#%s Status not active: %s \n", magnetId, status.Status)
				continue
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
				err = aria.client.Client().ChangeOptions(magnetId, arigo.Options{
					SelectFile: selectIndex,
				})
				if err != nil {
					log.Printf("change GID#%s options (select-file=%s) err: %s \n", magnetId, selectIndex, err.Error())
					return
				}

				log.Println("SELECT-Files: ", selectIndex)
			}
		}
	}
}

func (aria *Aria2) startEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s startEventHandle\n", event.GID)

	aria.mmu.Lock()
	defer aria.mmu.Unlock()
	aria.magnetMap[event.GID] = struct{}{}
}

func (aria *Aria2) pauseEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s pauseEventHandle\n", event.GID)
}

func (aria *Aria2) stopEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s stopEventHandle\n", event.GID)
}

func (aria *Aria2) completeEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s completeEventHandle\n", event.GID)

	aria.mmu.Lock()
	defer aria.mmu.Unlock()
	delete(aria.magnetMap, event.GID)
}

func (aria *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s btCompleteEventHandle\n", event.GID)
}

func (aria *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s errorEventHandle\n", event.GID)
}
