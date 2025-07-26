package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"os"
	"strings"
	"testing"
)

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
