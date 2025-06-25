package download

import (
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
)

type ClickBypassDownloader struct {
	downloader Downloader
}

func NewClickBypassDownloader() Downloader {
	return &ClickBypassDownloader{
		downloader: Default(),
	}
}

func (s *ClickBypassDownloader) SetCookies(u *url.URL, cookies []*http.Cookie) {
	s.downloader.SetCookies(u, cookies)
}

func (s *ClickBypassDownloader) Download(url string) (selection *goquery.Selection, err error) {
	
	return nil, nil
}
