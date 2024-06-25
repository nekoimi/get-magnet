package test

import (
	"get-magnet/scheduler"
	"github.com/PuerkitoBio/goquery"
	"log"
	"strings"
)

func DouBanTop250List(meta *scheduler.TaskMeta, selection *goquery.Selection) (scheduler.TaskOut, error) {
	var keepTasks []scheduler.Task

	selection.Find("div.article>ol.grid_view>li>div.item").Each(func(i int, s *goquery.Selection) {
		title := s.Find("div.info>div.hd>a>span").First().Text()
		log.Printf("Title: %s \n", title)
		detailsUrl := s.Find("div.pic>a").AttrOr("href", "")
		log.Printf("DetailsUrl: %s \n", detailsUrl)
		imgUrl := s.Find("div.pic>a>img").AttrOr("src", "")
		log.Printf("ImgUrl: %s \n", imgUrl)

		keepTasks = append(keepTasks, scheduler.Task{
			Url:    detailsUrl,
			Handle: DouBanDetails,
			Meta: &scheduler.TaskMeta{
				Host:    meta.Host,
				UrlPath: strings.ReplaceAll(detailsUrl, meta.Host, ""),
			},
		})
	})

	// next
	if nextUrl, exists := selection.Find("div.paginator>span.next>a").Attr("href"); exists {
		log.Printf("NextUrl: %s \n", meta.Host+"/top250"+nextUrl)
		keepTasks = append(keepTasks, scheduler.Task{
			Url:    meta.Host + "/top250" + nextUrl,
			Handle: DouBanTop250List,
			Meta: &scheduler.TaskMeta{
				Host:    meta.Host,
				UrlPath: nextUrl,
			},
		})
	}

	return scheduler.TaskOut{
		Tasks: keepTasks,
	}, nil
}

func DouBanDetails(meta *scheduler.TaskMeta, selection *goquery.Selection) (scheduler.TaskOut, error) {
	title := selection.Find("#content>h1>span").First().Text()
	log.Println("---------------------------------------------")
	log.Println(title)
	log.Println("---------------------------------------------")

	return scheduler.TaskOut{}, nil
}
