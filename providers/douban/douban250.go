package douban

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/common"
	"github.com/nekoimi/get-magnet/common/model"
	"log"
)

// Top250List
// Start
// e.Submit(task.NewTask("https://movie.douban.com/top250", douban.Top250List))
func Top250List(meta *common.Meta, selection *goquery.Selection) (*common.Out, error) {
	var keepTasks []*common.Task

	selection.Find("div.article>ol.grid_view>li>div.item").Each(func(i int, s *goquery.Selection) {
		title := s.Find("div.info>div.hd>a>span").First().Text()
		log.Printf("Title: %s \n", title)
		detailsUrl := s.Find("div.pic>a").AttrOr("href", "")
		log.Printf("DetailsUrl: %s \n", detailsUrl)
		imgUrl := s.Find("div.pic>a>img").AttrOr("src", "")
		log.Printf("ImgUrl: %s \n", imgUrl)

		keepTasks = append(keepTasks, common.NewTask(detailsUrl, Details))
	})

	// next
	if nextUrl, exists := selection.Find("div.paginator>span.next>a").Attr("href"); exists {
		log.Printf("NextUrl: %s \n", meta.Host+"/top250"+nextUrl)
		keepTasks = append(keepTasks, common.NewTask(meta.Host+"/top250"+nextUrl, Top250List))
	}

	return common.NewOut(keepTasks, nil), nil
}

func Details(meta *common.Meta, selection *goquery.Selection) (*common.Out, error) {
	title := selection.Find("#content>h1>span").First().Text()
	if imgSrc, exists := selection.Find("#mainpic>a.nbgnbg>img").Attr("src"); exists {
		return common.NewSingleOut(nil, &model.Item{
			Title:       title,
			OptimalLink: imgSrc,
			ResHost:     meta.Host,
			ResPath:     meta.UrlPath,
		}), nil
	}

	return common.NewEmptyOut(), nil
}
