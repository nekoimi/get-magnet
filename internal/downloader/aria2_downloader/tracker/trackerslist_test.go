package tracker

import (
	"os"
	"strings"
	"testing"

	"github.com/siku2/arigo"
)

func TestAria2_DownloadLatestTrackers(t *testing.T) {
	if os.Getenv("RUN_NETWORK_TESTS") != "1" {
		t.Skip("set RUN_NETWORK_TESTS=1 to run external tracker and aria2 integration test")
	}

	trackers, err := downloadLatestTrackers()
	if err != nil {
		t.Errorf("异常：%s", err.Error())
		return
	}

	t.Log(trackers)

	trackerStr := strings.Join(trackers, ",")

	t.Log(trackerStr)

	// 更新aria2配置
	jsonRPC := os.Getenv("ARIA2_JSONRPC")
	if jsonRPC == "" {
		t.Skip("ARIA2_JSONRPC is required")
	}
	client, err := arigo.Dial(jsonRPC, os.Getenv("ARIA2_SECRET"))
	if err != nil {
		t.Errorf("异常：%s", err.Error())
		return
	}

	if err = client.ChangeGlobalOptions(arigo.Options{
		BTTracker: strings.Join(trackers, ","),
	}); err != nil {
		t.Errorf("更新aria2最新tracker服务器信息异常：%s", err.Error())
	}

	t.Log("更新成功")
}
