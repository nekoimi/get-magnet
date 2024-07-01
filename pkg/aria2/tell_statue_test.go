package aria2

import (
	"github.com/nekoimi/arigo"
	"testing"
)

func TestTellStatue(t *testing.T) {
	// url := "ws://127.0.0.1:6800/jsonrpc"
	authToken := "nekoimi"

	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", authToken)
	if err != nil {
		panic(err)
	}

	g, err := client.AddURI(arigo.URIs("magnet:?xt=urn:btih:f696b5773fdf1dfd1dca3afe8ac77d97e6490372&tr=http://open.acgtracker.com:1096/announce"), nil)
	if err != nil {
		panic(err)
	}

	t.Log("GID: ", g.GID)

	for {
		s, err := client.TellStatus(g.GID, "gid", "status", "totalLength", "completedLength", "uploadLength", "downloadSpeed", "uploadSpeed", "errorCode", "errorMessage", "dir", "files")
		if err != nil {
			panic(err)
		}

		t.Log(s)
	}
}
