package contract

import "github.com/PuerkitoBio/goquery"

type Downloader interface {
	Download(url string) (*goquery.Selection, error)
}
