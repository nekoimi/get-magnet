package aria2

import (
	"github.com/siku2/arigo"
	"testing"
)

func TestCall(t *testing.T) {
	//client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	if err != nil {
		panic(err)
	}

	version, err := client.GetVersion()
	if err != nil {
		panic(err)
	}

	t.Log(version)

	//g, err := client.AddURI(arigo.URIs("magnet:?xt=urn:btih:00a7a42d5526cb01639a71b0cac74d707851acd8&dn=[javdb.com]ALDN-324-C.torrent"), nil)
	//if err != nil {
	//	panic(err)
	//}

	client.Subscribe(arigo.StartEvent, func(event *arigo.DownloadEvent) {
		t.Log("StartEvent: ", event.String())
	})

	client.Subscribe(arigo.StopEvent, func(event *arigo.DownloadEvent) {
		t.Log("StopEvent: ", event.String())
	})

	client.Subscribe(arigo.PauseEvent, func(event *arigo.DownloadEvent) {
		t.Log("PauseEvent: ", event.String())
	})

	client.Subscribe(arigo.ErrorEvent, func(event *arigo.DownloadEvent) {
		t.Log("ErrorEvent: ", event.String())
	})

	//t.Log("AddURI: ", g.GID)

	//go func() {
	//	time.Sleep(10 * time.Second)
	//	//client.PauseAll()
	//}()

	for {
	}
}
