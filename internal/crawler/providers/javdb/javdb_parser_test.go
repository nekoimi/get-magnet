package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/ocr"
	"golang.org/x/net/http/httpproxy"
	"os"
	"strings"
	"testing"
)

func TestSeeder_Handle(t *testing.T) {
	os.Setenv("ROD_HEADLESS", "false")
	os.Setenv("ROD_DATA_DIR", "C:\\Users\\nekoimi\\Downloads\\rod-data")
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")

	config.Default()

	go ocr.NewOcrServer("").Run()

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

func TestDetails_Handle(t *testing.T) {
	f, err := os.Open("D:\\Developer\\GoProjects\\go-library\\get-magnet\\deploy\\details_01.html")
	if err != nil {
		t.Fatal(err.Error())
	}
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err.Error())
	}

	root := doc.Selection

	// parse
	s := root.Find("section.section>div.container").First()
	// Title
	var title = s.Find("div.video-detail > h2").Text()
	title = strings.ReplaceAll(title, "\n", "")
	title = strings.ReplaceAll(title, "  ", "")
	title = strings.TrimSpace(title)
	t.Logf("Title: %s", title)
	// Number
	var number = s.Find(".movie-panel-info>div.first-block>span.value").Text()
	t.Logf("Number: %s", number)
	// Actress
	var actress []string
	s.Find("nav.panel.movie-panel-info > div.panel-block").Each(func(i int, sub *goquery.Selection) {
		titleVal := sub.Find("strong").Text()
		if strings.Contains(titleVal, "演員") {
			var (
				currALink         *goquery.Selection
				currALinkNext     *goquery.Selection
				aLinkText         string
				aLinkTextNextText string
				ok                bool
			)
			currALink = sub.Find("span.value").Find("a").First()
			for {
				aLinkText = strings.TrimSpace(currALink.Text())
				if aLinkText == "" {
					break
				}
				currALinkNext = currALink.Next()
				if aLinkTextNextText, ok = currALinkNext.Attr("class"); ok {
					if strings.Contains(aLinkTextNextText, "female") {
						actress = append(actress, aLinkText)
					}
					currALink = currALinkNext.Next()
				} else {
					break
				}
			}
		}
	})
	actress0 := strings.Join(actress, ",")
	t.Logf("Actress0: %s", actress0)
}
