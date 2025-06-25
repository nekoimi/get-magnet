package aria2

import (
	"github.com/siku2/arigo"
	"runtime/debug"
	"testing"
)

func TestAria2_Client(t *testing.T) {
	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	if err != nil {
		t.Fatal(err.Error())
	}

	// 获取全局属性
	options, err := client.GetGlobalOptions()
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Log(options)
	t.Log(options.MaxConcurrentDownloads)

	// 获取下载任务状态
	tasks, err := client.TellActive("gid", "status", "downloadSpeed")
	if err != nil {
		t.Log(err)
		t.Log(string(debug.Stack()))
	}

	for _, task := range tasks {
		t.Log(task)
	}
}
