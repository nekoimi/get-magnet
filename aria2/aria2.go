package aria2

import (
	"context"
	"errors"
	"github.com/cenkalti/rpc2"
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/common/model"
	"github.com/nekoimi/get-magnet/config"
	"github.com/nekoimi/get-magnet/pkg/aria2_ext"
	"github.com/nekoimi/get-magnet/pkg/util"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Aria2 struct {
	cfg *config.Aria2

	cmux    *sync.Mutex
	_client *arigo.Client

	magnetChan chan *model.Item

	mmux      *sync.Mutex
	magnetMap map[string]struct{}

	exit chan struct{}
}

func New(cfg *config.Aria2) *Aria2 {
	aria := &Aria2{
		cfg:        cfg,
		magnetChan: make(chan *model.Item),
		mmux:       &sync.Mutex{},
		magnetMap:  make(map[string]struct{}),
		exit:       make(chan struct{}),
	}

	aria.client().Subscribe(arigo.StartEvent, aria.startEventHandle)
	aria.client().Subscribe(arigo.PauseEvent, aria.pauseEventHandle)
	aria.client().Subscribe(arigo.StopEvent, aria.stopEventHandle)
	aria.client().Subscribe(arigo.CompleteEvent, aria.completeEventHandle)
	aria.client().Subscribe(arigo.BTCompleteEvent, aria.btCompleteEventHandle)
	aria.client().Subscribe(arigo.ErrorEvent, aria.errorEventHandle)

	return aria
}

func (a *Aria2) Submit(item *model.Item) {
	a.magnetChan <- item
}

func (a *Aria2) Run() {
	go a.bestFileSelectWork()

	for {
		select {
		case item := <-a.magnetChan:
			a.createDownload(item)
		}
	}
}

func (a *Aria2) Stop() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		close(a.exit)
		for len(a.magnetChan) > 0 {
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

func (a *Aria2) createDownload(item *model.Item) {
	magnetLink := item.OptimalLink
	log.Printf("add url to aria2: %s \n", magnetLink)
	ops, err := a.client().GetGlobalOptions()
	if err != nil {
		panic(err)
	}

	host := strings.ReplaceAll(strings.ReplaceAll(util.CleanHost(item.ResHost), ":", "_"), ".", "_")
	saveDir := ops.Dir + "/" + util.NowDate("-") + "/" + host
	g, err := a.client().AddURI(arigo.URIs(magnetLink), &arigo.Options{
		Dir: saveDir,
	})
	if err != nil {
		log.Printf("add uri (%s) to aria2 err: %s \n", magnetLink, err.Error())
		return
	}

	a.mmux.Lock()
	defer a.mmux.Unlock()
	a.magnetMap[g.GID] = struct{}{}
}

func (a *Aria2) bestFileSelectWork() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-a.exit:
			return
		default:
		}

		a.mmux.Lock()
		for magnetId := range a.magnetMap {
			status, err := a.client().TellStatus(magnetId, "status", "errorCode", "errorMessage", "dir", "files")
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
				if f.Selected && !aria2_ext.IsBestFile(f) {
					needChangeOps = true
					break
				}
			}

			if needChangeOps {
				allowFiles := aria2_ext.BestSelectFile(files)
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
		a.mmux.Unlock()
	}
}

func (a *Aria2) startEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s startEventHandle\n", event.GID)

	a.mmux.Lock()
	defer a.mmux.Unlock()
	a.magnetMap[event.GID] = struct{}{}
}

func (a *Aria2) pauseEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s pauseEventHandle\n", event.GID)
}

func (a *Aria2) stopEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s stopEventHandle\n", event.GID)
}

func (a *Aria2) completeEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s completeEventHandle\n", event.GID)

	a.mmux.Lock()
	defer a.mmux.Unlock()
	delete(a.magnetMap, event.GID)
}

func (a *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s btCompleteEventHandle\n", event.GID)
}

func (a *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	log.Printf("GID#%s errorEventHandle\n", event.GID)
}

func (a *Aria2) client() *arigo.Client {
	a.cmux.Lock()
	defer a.cmux.Unlock()

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
	client, err := arigo.Dial(a.cfg.JsonRpc, a.cfg.Secret)
	if err != nil {
		return err
	}
	a._client = client
	return nil
}
