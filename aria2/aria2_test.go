package aria2

import (
	"fmt"
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/internal/model"
	"github.com/nekoimi/get-magnet/pkg/util"
	"log"
	"sync"
	"testing"
	"time"
)

func TestSubmit(t *testing.T) {
	a := New()
	go a.Run()

	time.Sleep(1 * time.Second)

	// magnet:?xt=urn:btih:ce260805da416121016bb0943e4c24119a6876b5&dn=[javdb.com]JUQ-727-C.torrent
	a.Submit(&model.Item{
		OptimalLink: "magnet:?xt=urn:btih:d1185a812f7252b99225af665a69bfa06d2e09a7&tr=http://open.acgtracker.com:1096/announce",
		ResHost:     "https://javdb.com",
	})

	for {
	}
}

func showState(client *arigo.Client, gid string, logPrefix string) {
	s, err := client.TellStatus(gid, "gid", "status", "totalLength", "completedLength", "uploadLength", "downloadSpeed", "uploadSpeed", "errorCode", "errorMessage", "dir", "files")
	if err != nil {
		log.Printf("TellStatus err: %s \n", err.Error())
		panic(err)
		return
	}

	fmt.Printf("[%s] GID#%s-STATE: \n %s \n", logPrefix, gid, util.ToJson(s))
}

func TestCall(t *testing.T) {
	var maxSelectNum = 3
	var wg sync.WaitGroup

	wg.Add(1)

	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	if err != nil {
		panic(err)
	}

	version, err := client.GetVersion()
	if err != nil {
		panic(err)
	}

	t.Log(version)

	g, err := client.AddURI(arigo.URIs("magnet:?xt=urn:btih:f696b5773fdf1dfd1dca3afe8ac77d97e6490372&tr=http://open.acgtracker.com:1096/announce"), nil)
	if err != nil {
		panic(err)
	}

	g.Subscribe(arigo.StartEvent, func(event *arigo.DownloadEvent) {
		t.Log("startEventHandle: ", event.String())
		showState(client, event.GID, "startEventHandle")

		if maxSelectNum > 0 {
			// Select File
			files, err := client.GetFiles(g.GID)
			if err != nil {
				panic(err)
			}

			Sort(files, func(a, b *arigo.File) bool {
				return a.Length > b.Length
			})

			// 优先选择下载最大的文件
			selectFile := files[0]
			t.Log("selectFile: ", selectFile)

			err = client.ChangeOptions(g.GID, arigo.Options{
				SelectFile: fmt.Sprintf("%d", selectFile.Index),
			})
			if err != nil {
				panic(err)
			}

			// maxSelectNum--
		}
	})

	g.Subscribe(arigo.PauseEvent, func(event *arigo.DownloadEvent) {
		t.Log("pauseEventHandle: ", event.String())
		showState(client, event.GID, "pauseEventHandle")
	})

	g.Subscribe(arigo.StopEvent, func(event *arigo.DownloadEvent) {
		t.Log("stopEventHandle: ", event.String())
		showState(client, event.GID, "stopEventHandle")
	})

	g.Subscribe(arigo.CompleteEvent, func(event *arigo.DownloadEvent) {
		t.Log("completeEventHandle: ", event.String())
		showState(client, event.GID, "completeEventHandle")
	})

	g.Subscribe(arigo.BTCompleteEvent, func(event *arigo.DownloadEvent) {
		t.Log("btCompleteEventHandle: ", event.String())
		showState(client, event.GID, "btCompleteEventHandle")
	})

	g.Subscribe(arigo.ErrorEvent, func(event *arigo.DownloadEvent) {
		t.Log("errorEventHandle: ", event.String())
		showState(client, event.GID, "errorEventHandle")
	})

	t.Log("AddURI: ", g.GID)

	wg.Wait()
}
