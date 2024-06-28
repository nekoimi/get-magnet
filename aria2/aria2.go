package aria2

import (
	"context"
	"get-magnet/internal/model"
	"get-magnet/pkg/util"
	"github.com/nekoimi/arigo"
	"log"
	"strconv"
	"strings"
	"time"
)

type Aria2 struct {
	client     *arigo.Client
	magnetChan chan *model.MagnetItem
	magnetIds  []string
	exit       chan struct{}
}

func New() *Aria2 {
	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	if err != nil {
		panic(err)
	}

	aria := &Aria2{
		client:     client,
		magnetChan: make(chan *model.MagnetItem),
		magnetIds:  make([]string, 0),
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

func (aria *Aria2) Submit(item *model.MagnetItem) {
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

func (aria *Aria2) createDownload(item *model.MagnetItem) {
	magnetLink := item.OptimalLink
	log.Printf("add url to aria2: %s \n", magnetLink)
	ops, err := aria.client.GetGlobalOptions()
	if err != nil {
		panic(err)
	}

	host := strings.ReplaceAll(strings.ReplaceAll(util.CleanHost(item.ResHost), ":", "_"), ".", "_")
	saveDir := ops.Dir + "/" + util.NowDate("-") + "/" + host
	_, err = aria.client.AddURI(arigo.URIs(magnetLink), &arigo.Options{
		Dir: saveDir,
	})
	if err != nil {
		log.Printf("add uri (%s) to aria2 err: %s \n", magnetLink, err.Error())
		return
	}
}

func (aria *Aria2) bestFileSelectWork() {
	for {
		time.Sleep(30 * time.Second)
		select {
		case <-aria.exit:
			return
		default:
		}

		for _, magnetId := range aria.magnetIds {
			status, err := aria.client.TellStatus(magnetId, "status", "errorCode", "errorMessage", "dir", "files")
			if err != nil {
				log.Printf("fetch GID#%s download status err: %s \n", magnetId, err.Error())
				continue
			}

			files := status.Files
			if status.Status == arigo.StatusError {
				for _, f := range files {
					if f.Selected {
						log.Printf("GID#%s StatusError: %s \n", magnetId, f.Path)
					}
				}
				continue
			}

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
}

func (aria *Aria2) pauseEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s pauseEventHandle\n", event.GID)
}

func (aria *Aria2) stopEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s stopEventHandle\n", event.GID)
}

func (aria *Aria2) completeEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s completeEventHandle\n", event.GID)
}

func (aria *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s btCompleteEventHandle\n", event.GID)
}

func (aria *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s errorEventHandle\n", event.GID)
}
