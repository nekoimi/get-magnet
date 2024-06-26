package test

import (
	"get-magnet/internal/model"
	"github.com/PuerkitoBio/goquery"
	"log"
	"strings"
)

func DouBanTop250List(meta *model.TaskMeta, selection *goquery.Selection) (model.TaskOut, error) {
	var keepTasks []model.Task

	selection.Find("div.article>ol.grid_view>li>div.item").Each(func(i int, s *goquery.Selection) {
		title := s.Find("div.info>div.hd>a>span").First().Text()
		log.Printf("Title: %s \n", title)
		detailsUrl := s.Find("div.pic>a").AttrOr("href", "")
		log.Printf("DetailsUrl: %s \n", detailsUrl)
		imgUrl := s.Find("div.pic>a>img").AttrOr("src", "")
		log.Printf("ImgUrl: %s \n", imgUrl)

		keepTasks = append(keepTasks, model.Task{
			Url:    detailsUrl,
			Handle: DouBanDetails,
			Meta: &model.TaskMeta{
				Host:    meta.Host,
				UrlPath: strings.ReplaceAll(detailsUrl, meta.Host, ""),
			},
		})
	})

	// next
	if nextUrl, exists := selection.Find("div.paginator>span.next>a").Attr("href"); exists {
		log.Printf("NextUrl: %s \n", meta.Host+"/top250"+nextUrl)
		keepTasks = append(keepTasks, model.Task{
			Url:    meta.Host + "/top250" + nextUrl,
			Handle: DouBanTop250List,
			Meta: &model.TaskMeta{
				Host:    meta.Host,
				UrlPath: nextUrl,
			},
		})
	}

	return model.TaskOut{
		Tasks: keepTasks,
	}, nil
}

func DouBanDetails(meta *model.TaskMeta, selection *goquery.Selection) (model.TaskOut, error) {
	title := selection.Find("#content>h1>span").First().Text()
	log.Println("---------------------------------------------")
	log.Println(title)
	log.Println("---------------------------------------------")

	return model.TaskOut{}, nil
}
