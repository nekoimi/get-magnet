package douban

import (
	"get-magnet/internal/model"
	"get-magnet/internal/task"
	"github.com/PuerkitoBio/goquery"
	"log"
)

func Top250List(meta *task.Meta, selection *goquery.Selection) (*task.Out, error) {
	var keepTasks []*task.Task

	selection.Find("div.article>ol.grid_view>li>div.item").Each(func(i int, s *goquery.Selection) {
		title := s.Find("div.info>div.hd>a>span").First().Text()
		log.Printf("Title: %s \n", title)
		detailsUrl := s.Find("div.pic>a").AttrOr("href", "")
		log.Printf("DetailsUrl: %s \n", detailsUrl)
		imgUrl := s.Find("div.pic>a>img").AttrOr("src", "")
		log.Printf("ImgUrl: %s \n", imgUrl)

		keepTasks = append(keepTasks, task.NewTask(detailsUrl, Details))
	})

	// next
	if nextUrl, exists := selection.Find("div.paginator>span.next>a").Attr("href"); exists {
		log.Printf("NextUrl: %s \n", meta.Host+"/top250"+nextUrl)
		keepTasks = append(keepTasks, task.NewTask(meta.Host+"/top250"+nextUrl, Top250List))
	}

	return task.NewOut(keepTasks, nil), nil
}

func Details(meta *task.Meta, selection *goquery.Selection) (*task.Out, error) {
	title := selection.Find("#content>h1>span").First().Text()
	if imgSrc, exists := selection.Find("#mainpic>a.nbgnbg>img").Attr("src"); exists {
		return task.NewSingleOut(nil, &model.MagnetItem{
			Title:       title,
			OptimalLink: imgSrc,
			ResHost:     meta.Host,
			ResPath:     meta.UrlPath,
		}), nil
	}

	return task.NewEmptyOut(), nil
}
