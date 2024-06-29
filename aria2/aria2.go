package aria2

import (
	"context"
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/internal/model"
	"github.com/nekoimi/get-magnet/pkg/util"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Aria2 struct {
	mu         *sync.Mutex
	client     *arigo.Client
	magnetChan chan *model.Item
	magnetMap  map[string]struct{}
	exit       chan struct{}
}

func New(rpc string, token string) *Aria2 {
	client, err := arigo.Dial(rpc, token)
	if err != nil {
		panic(err)
	}

	aria := &Aria2{
		mu:         &sync.Mutex{},
		client:     client,
		magnetChan: make(chan *model.Item),
		magnetMap:  make(map[string]struct{}),
		exit:       make(chan struct{}),
	}

	aria.client.Subscribe(arigo.StartEvent, aria.startEventHandle)
	aria.client.Subscribe(arigo.PauseEvent, aria.pauseEventHandle)
	aria.client.Subscribe(arigo.StopEvent, aria.stopEventHandle)
	aria.client.Subscribe(arigo.CompleteEvent, aria.completeEventHandle)
	aria.client.Subscribe(arigo.BTCompleteEvent, aria.btCompleteEventHandle)
	aria.client.Subscribe(arigo.ErrorEvent, aria.errorEventHandle)

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
		err := aria.client.Close()
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
	ops, err := aria.client.GetGlobalOptions()
	if err != nil {
		panic(err)
	}

	host := strings.ReplaceAll(strings.ReplaceAll(util.CleanHost(item.ResHost), ":", "_"), ".", "_")
	saveDir := ops.Dir + "/" + util.NowDate("-") + "/" + host
	g, err := aria.client.AddURI(arigo.URIs(magnetLink), &arigo.Options{
		Dir: saveDir,
	})
	if err != nil {
		log.Printf("add uri (%s) to aria2 err: %s \n", magnetLink, err.Error())
		return
	}

	aria.mu.Lock()
	defer aria.mu.Unlock()
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

		for magnetId, _ := range aria.magnetMap {
			status, err := aria.client.TellStatus(magnetId, "status", "errorCode", "errorMessage", "dir", "files")
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
				err = aria.client.ChangeOptions(magnetId, arigo.Options{
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

	aria.mu.Lock()
	defer aria.mu.Unlock()
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

	aria.mu.Lock()
	defer aria.mu.Unlock()
	delete(aria.magnetMap, event.GID)
}

func (aria *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s btCompleteEventHandle\n", event.GID)
}

func (aria *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s errorEventHandle\n", event.GID)
}
