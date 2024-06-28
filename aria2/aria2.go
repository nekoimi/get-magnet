package aria2

import (
	"bytes"
	"context"
	"fmt"
	"get-magnet/internal/model"
	"get-magnet/pkg/file"
	"get-magnet/pkg/util"
	"github.com/nekoimi/arigo"
	"log"
	"strings"
)

const MinVideoSize = 100_000_000

type Aria2 struct {
	client     *arigo.Client
	magnetChan chan *model.MagnetItem
}

func New() *Aria2 {
	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	if err != nil {
		panic(err)
	}

	aria := &Aria2{
		client:     client,
		magnetChan: make(chan *model.MagnetItem),
	}

	aria.client.Subscribe(arigo.StartEvent, aria.StartEvent)
	aria.client.Subscribe(arigo.PauseEvent, aria.PauseEvent)
	aria.client.Subscribe(arigo.StopEvent, aria.StopEvent)
	aria.client.Subscribe(arigo.CompleteEvent, aria.CompleteEvent)
	aria.client.Subscribe(arigo.BTCompleteEvent, aria.BTCompleteEvent)
	aria.client.Subscribe(arigo.ErrorEvent, aria.ErrorEvent)

	return aria
}

func (aria *Aria2) Submit(item *model.MagnetItem) {
	aria.magnetChan <- item
}

func (aria *Aria2) Run() {
	for {
		select {
		case item := <-aria.magnetChan:
			aria.download(item)
		}
	}
}

func (aria *Aria2) download(item *model.MagnetItem) {
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

func (aria *Aria2) Stop() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := aria.client.Close()
		if err != nil {
			log.Printf("aria2 client close err: %s \n", err.Error())
		}
		log.Println("stop aria2 client")
		cancel()
	}()
	<-ctx.Done()
}

func (aria *Aria2) StartEvent(event *arigo.DownloadEvent) {
	log.Printf("GID#%s StartEvent\n", event.GID)

	status, err := aria.client.TellStatus(event.GID, "status", "errorCode", "errorMessage", "dir", "files")
	if err != nil {
		log.Printf("fetch GID#%s download status err: %s \n", event.GID, err.Error())
		return
	}
	if status.Status == arigo.StatusError {
		log.Printf("GID#%s StatusError \n", event.GID)
		return
	}

	files := status.Files
	if len(files) <= 1 {
		log.Printf("GID#%s file length only one: %s \n", event.GID, util.ToJson(files[0]))
		return
	}

	// 允许下载的文件列表
	var allowFiles []arigo.File
	for _, f := range files {
		if file.IsVideo(f.Path) && f.Length > MinVideoSize {
			allowFiles = append(allowFiles, f)
			log.Printf("GID#%s video file [%s] length: %d \n", event.GID, f.Path, f.Length)
		}
	}

	bufs := bytes.NewBufferString("")
	for _, a := range allowFiles {
		bufs.WriteString(fmt.Sprintf("%d", a.Index))
		bufs.WriteString(",")
	}
	selectFile, _ := strings.CutSuffix(bufs.String(), ",")
	err = aria.client.ChangeOptions(event.GID, arigo.Options{
		SelectFile: selectFile,
	})
	if err != nil {
		log.Printf("change GID#%s options (select-file=%s) err: %s \n", event.GID, selectFile, err.Error())
		return
	}

	log.Println("SELECT-Files: ", selectFile)
}

func (aria *Aria2) PauseEvent(event *arigo.DownloadEvent) {
	log.Printf("GID#%s PauseEvent\n", event.GID)
}

func (aria *Aria2) StopEvent(event *arigo.DownloadEvent) {
	log.Printf("GID#%s StopEvent\n", event.GID)
}

func (aria *Aria2) CompleteEvent(event *arigo.DownloadEvent) {
	log.Printf("GID#%s CompleteEvent\n", event.GID)
}

func (aria *Aria2) BTCompleteEvent(event *arigo.DownloadEvent) {
	log.Printf("GID#%s BTCompleteEvent\n", event.GID)
}

func (aria *Aria2) ErrorEvent(event *arigo.DownloadEvent) {
	log.Printf("GID#%s ErrorEvent\n", event.GID)
}
