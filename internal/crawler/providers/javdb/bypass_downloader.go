package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	log "github.com/sirupsen/logrus"
)

var bypassDownloaderSingleton = singleton.New[download.Downloader](func() download.Downloader {
	return buildBypassDownloader()
})

func GetBypassDownloader() download.Downloader {
	return bypassDownloaderSingleton.Get()
}

func buildBypassDownloader() download.Downloader {
	return download.NewClickBypassDownloader(
		func(root *goquery.Selection) bool {
			return root.Find("body > div.modal.is-active.over18-modal").Size() > 0
		},
		func(page *rod.Page) error {
			btn := page.MustElementByJS(`() => document.querySelector("body > div.modal.is-active.over18-modal > div.modal-card > footer > a.button.is-success.is-large")`)
			text, err := btn.Text()
			if err != nil {
				return err
			}
			log.Debugf("点击访问按钮: %s", text)
			btn.MustClick()
			return nil
		},
	)
}
