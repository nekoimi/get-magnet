package sehuatang

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	log "github.com/sirupsen/logrus"
)

func newBypassDownloader(browser *rod_browser.Browser, cloudflarePassApi string) download.Downloader {
	return download.NewClickBypassDownloader(
		browser,
		cloudflarePassApi,
		func(root *goquery.Selection) bool {
			return root.Find("#hd").Size() == 0
		},
		func(page *rod.Page) error {
			btn := page.MustElementByJS(`() => document.querySelector("body > a:nth-child(5)")`)
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
