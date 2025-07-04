package javdb

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/ocr"
	"golang.org/x/net/http/httpproxy"
	"os"
	"testing"
)

func TestSeeder_Handle(t *testing.T) {
	os.Setenv("ROD_HEADLESS", "false")
	os.Setenv("ROD_DATA_DIR", "C:\\Users\\nekoimi\\Downloads\\rod-data")
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")

	config.Default()

	ctx := context.Background()
	go ocr.NewServer().Start(ctx)

	testUrl := "https://javdb.com/censored?vft=2&vst=1"
	//testUrl := "https://javdb.com/login"

	proxyEnv := httpproxy.FromEnvironment()
	t.Log(proxyEnv.HTTPProxy)
	t.Log(proxyEnv.HTTPSProxy)
	t.Log(proxyEnv.NoProxy)

	downloader := GetBypassDownloader()
	s1, err := downloader.Download(testUrl)
	if err != nil {
		panic(err)
	}
	html, err := s1.Html()
	if err != nil {
		panic(err)
	}

	t.Log(html)

	err = os.WriteFile("D:\\Developer\\GoProjects\\go-library\\get-magnet\\deploy\\html_javdb_2.html", []byte(html), 0666)
	if err != nil {
		panic(err)
	}

	select {}
}

func TestSeeder_Handle2(t *testing.T) {
	f, err := os.Open("D:\\Developer\\GoProjects\\go-library\\get-magnet\\deploy\\html_javdb_2.html")
	if err != nil {
		panic(err)
	}
	root, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		panic(err)
	}

	find := root.Find("body > div.modal.is-active.over18-modal")
	t.Log(find.Text())
}
