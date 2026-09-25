package javdb

import (
	"context"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/scrapio/internal/crawler/download"
	"github.com/nekoimi/scrapio/internal/drission_rod"
	log "github.com/sirupsen/logrus"
)

type drissionRodDownloader struct {
	downloader download.Downloader
}

func newDrissionRodDownloader(ctx context.Context) download.Downloader {
	return &drissionRodDownloader{
		downloader: drission_rod.NewHTMLDownloader(ctx, "javdb"),
	}
}

func (s *drissionRodDownloader) SetCookies(u *url.URL, cookies []*http.Cookie) {
}

func (s *drissionRodDownloader) Download(rawUrl string) (selection *goquery.Selection, err error) {
	log.Debugf("等待页面 %s 加载...", rawUrl)
	return s.downloader.Download(rawUrl)
}
