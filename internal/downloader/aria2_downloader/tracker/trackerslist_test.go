package tracker

import (
	"github.com/siku2/arigo"
	"os"
	"strings"
	"testing"
)

func TestAria2_DownloadLatestTrackers(t *testing.T) {
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")

	trackers, err := downloadLatestTrackers()
	if err != nil {
		t.Errorf("异常：%s", err.Error())
		return
	}

	t.Log(trackers)

	trackerStr := strings.Join(trackers, ",")

	t.Log(trackerStr)

	// 更新aria2配置
	client, err := arigo.Dial("", "")
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
