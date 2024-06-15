package test

import (
	"github.com/PuerkitoBio/goquery"
	"net/url"
	"os"
	"testing"
)

func TestGetList(t *testing.T) {
	fr, err := os.OpenFile("html/list.html", os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer fr.Close()

	doc, err := goquery.NewDocumentFromReader(fr)
	if err != nil {
		panic(err)
	}

	var next = true

	doc.Find(".movie-list>div>a.box").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		fullLink, err := url.JoinPath(JavdbRootDomain, href)
		if err != nil {
			panic(err)
		}

		t.Log("idx: ", i, fullLink)
	})

	if next {
		t.Log("execute next...")
	}
}
