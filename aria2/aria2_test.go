package aria2

import (
	"github.com/siku2/arigo"
	"testing"
	"time"
)

func TestCall(t *testing.T) {
	//client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	client, err := arigo.Dial("ws://10.1.1.100:6800/jsonrpc", "nekoimi")
	if err != nil {
		panic(err)
	}

	version, err := client.GetVersion()
	if err != nil {
		panic(err)
	}

	t.Log(version)

	g, err := client.AddURI(arigo.URIs("magnet:?xt=urn:btih:00a7a42d5526cb01639a71b0cac74d707851acd8&dn=[javdb.com]ALDN-324-C.torrent"), nil)
	if err != nil {
		panic(err)
	}

	g.Subscribe(arigo.StartEvent, func(event *arigo.DownloadEvent) {
		t.Log("StartEvent: ", event.String())
	})

	g.Subscribe(arigo.StopEvent, func(event *arigo.DownloadEvent) {
		t.Log("StopEvent: ", event.String())
	})

	g.Subscribe(arigo.PauseEvent, func(event *arigo.DownloadEvent) {
		t.Log("PauseEvent: ", event.String())
	})

	g.Subscribe(arigo.ErrorEvent, func(event *arigo.DownloadEvent) {
		t.Log("ErrorEvent: ", event.String())
	})

	t.Log("AddURI: ", g.GID)

	go func() {
		time.Sleep(10 * time.Second)
		client.PauseAll()
	}()

	for {
	}
}
