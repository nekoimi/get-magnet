package download

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"time"
)

// RodBrowserDownloader 浏览器下载
type RodBrowserDownloader struct {
}

func NewRodBrowserDownloader() Downloader {
	return &RodBrowserDownloader{}
}

func (s *RodBrowserDownloader) SetCookies(u *url.URL, cookies []*http.Cookie) {
}

func (s *RodBrowserDownloader) Download(rawUrl string) (selection *goquery.Selection, err error) {
	page, closeFunc := rod_browser.NewTabPage()
	defer closeFunc()

	page.MustNavigate(rawUrl)
	// 等待页面加载
	log.Debugf("等待页面 %s 加载...", rawUrl)
	page.Timeout(10 * time.Second).MustWaitStable()

	html, err := page.HTML()
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(html))
	if err != nil {
		return nil, err
	}

	return doc.Selection, nil
}
