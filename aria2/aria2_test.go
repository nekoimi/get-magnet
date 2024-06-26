package aria2

import (
	"fmt"
	"get-magnet/pkg/util"
	"github.com/siku2/arigo"
	"log"
	"sync"
	"testing"
)

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
		t.Log("StartEvent: ", event.String())
		showState(client, event.GID, "StartEvent")

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
		t.Log("PauseEvent: ", event.String())
		showState(client, event.GID, "PauseEvent")
	})

	g.Subscribe(arigo.StopEvent, func(event *arigo.DownloadEvent) {
		t.Log("StopEvent: ", event.String())
		showState(client, event.GID, "StopEvent")
	})

	g.Subscribe(arigo.CompleteEvent, func(event *arigo.DownloadEvent) {
		t.Log("CompleteEvent: ", event.String())
		showState(client, event.GID, "CompleteEvent")
	})

	g.Subscribe(arigo.BTCompleteEvent, func(event *arigo.DownloadEvent) {
		t.Log("BTCompleteEvent: ", event.String())
		showState(client, event.GID, "BTCompleteEvent")
	})

	g.Subscribe(arigo.ErrorEvent, func(event *arigo.DownloadEvent) {
		t.Log("ErrorEvent: ", event.String())
		showState(client, event.GID, "ErrorEvent")
	})

	t.Log("AddURI: ", g.GID)

	wg.Wait()
}
