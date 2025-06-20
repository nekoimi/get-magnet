package test

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/common/task"
	"github.com/nekoimi/get-magnet/core/engine"
	"github.com/nekoimi/get-magnet/providers/javdb"
	"github.com/nekoimi/get-magnet/storage"
	"os"
	"testing"
)

func Test_Engine(t *testing.T) {
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:2080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:2080")

	e := engine.New(&engine.Config{
		WorkerNum: 1,
		DbDsn:     "root:mysql#123456@(10.1.1.100:3306)/get_magnet_dev",
		Jsonrpc:   "",
		Secret:    "",
		Storage:   storage.Console,
	})

	// 立即执行
	e.Submit(task.NewTask(1, "https://javdb.com/actors/kW6?t=c&sort_type=0", javdb.ChineseSubtitlesMovieList))

	e.Run()
}

func Test_PageParse(t *testing.T) {
	file, err := os.Open("test.html")
	if err != nil {
		fmt.Println("打开文件失败:", err)
		return
	}
	defer file.Close()

	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		fmt.Println("NewDocumentFromReader error:", err)
		return
	}

	javdb.ChineseSubtitlesMovieList(&task.Meta{}, doc.Selection)
}
