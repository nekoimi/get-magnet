package aria2

import (
	"context"
	"errors"
	"github.com/cenkalti/rpc2"
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/internal/contract"
	"github.com/nekoimi/get-magnet/internal/pkg/queue"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Aria2 struct {
	cMux            *sync.Mutex
	_client         *arigo.Client
	downloadChan    chan contract.DownloadTask
	bestSelectQueue *queue.Queue[string]
	exit            chan struct{}
}

func New() *Aria2 {
	a := &Aria2{
		downloadChan:    make(chan contract.DownloadTask),
		bestSelectQueue: queue.New[string](),
		exit:            make(chan struct{}, 1),
	}

	a.client().Subscribe(arigo.StartEvent, a.startEventHandle)
	a.client().Subscribe(arigo.PauseEvent, a.pauseEventHandle)
	a.client().Subscribe(arigo.StopEvent, a.stopEventHandle)
	a.client().Subscribe(arigo.CompleteEvent, a.completeEventHandle)
	a.client().Subscribe(arigo.BTCompleteEvent, a.btCompleteEventHandle)
	a.client().Subscribe(arigo.ErrorEvent, a.errorEventHandle)

	return a
}

func (a *Aria2) Submit(t contract.DownloadTask) {
	a.downloadChan <- t
}

func (a *Aria2) Start() {
	go a.bestFileSelectWork()

	for {
		select {
		case t := <-a.downloadChan:
			a.createDownload(t)
		}
	}
}

func (a *Aria2) Stop() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		close(a.exit)
		for len(a.downloadChan) > 0 {
			time.Sleep(1 * time.Second)
		}
		err := a.client().Close()
		if err != nil {
			log.Printf("aria2 client close err: %s \n", err.Error())
		}
		log.Println("stop aria2 client")
		cancel()
	}()
	<-ctx.Done()
}

func (a *Aria2) createDownload(t contract.DownloadTask) {
	magnetUrl := t.Url()
	log.Printf("add url to aria2: %s \n", magnetUrl)
	ops, err := a.client().GetGlobalOptions()
	if err != nil {
		panic(err)
	}

	saveDir := ops.Dir + "/" + util.NowDate("-") + "/" + t.Category()
	g, err := a.client().AddURI(arigo.URIs(magnetUrl), &arigo.Options{
		Dir: saveDir,
	})
	if err != nil {
		log.Printf("add uri (%s) to aria2 err: %s \n", magnetUrl, err.Error())
		return
	}

	a.bestSelectQueue.Add(g.GID)
}

// 下载文件优选
func (a *Aria2) bestFileSelectWork() {
	for {
		select {
		case <-a.exit:
			return
		default:
			magnetId := a.bestSelectQueue.PollWait()
			status, err := a.client().TellStatus(magnetId, "status", "errorCode", "errorMessage", "dir", "files")
			if err != nil {
				log.Printf("fetch GID#%s download status err: %s \n", magnetId, err.Error())
				a.bestSelectQueue.Add(magnetId)
				continue
			}

			if status.Status == arigo.StatusCompleted {
				continue
			}

			if status.Status != arigo.StatusActive {
				log.Printf("GID#%s Status not active: %s \n", magnetId, status.Status)
				a.bestSelectQueue.Add(magnetId)
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
				err = a.client().ChangeOptions(magnetId, arigo.Options{
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

func (a *Aria2) startEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s startEventHandle\n", event.GID)
	a.bestSelectQueue.Add(event.GID)
}

func (a *Aria2) pauseEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s pauseEventHandle\n", event.GID)
}

func (a *Aria2) stopEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s stopEventHandle\n", event.GID)
}

func (a *Aria2) completeEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s completeEventHandle\n", event.GID)
}

func (a *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s btCompleteEventHandle\n", event.GID)
}

func (a *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s errorEventHandle\n", event.GID)
}

func (a *Aria2) client() *arigo.Client {
	a.cMux.Lock()
	defer a.cMux.Unlock()

	err := a.ping()
	if err != nil {
		if errors.Is(err, rpc2.ErrShutdown) {
			for {
				err := a.connect()
				if err != nil {
					log.Printf("Check the rpc connection is closed, reconnect... %s\n", err.Error())
					time.Sleep(5 * time.Second)
					continue
				}

				break
			}
		}
	}

	return a._client
}

func (a *Aria2) ping() error {
	_, err := a._client.GetVersion()
	return err
}

func (a *Aria2) connect() error {
	client, err := arigo.Dial("a.cfg.JsonRpc", "a.cfg.Secret")
	if err != nil {
		return err
	}
	a._client = client
	return nil
}
