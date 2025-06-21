package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"log"
)

type movieList struct {
}

func RunEndpoint() crawler.WorkerTaskHandler {
	return &movieList{}
}

func (p *movieList) Handle(t crawler.WorkerTask) (tasks []crawler.WorkerTask, outputs []crawler.Magnet, err error) {
	if task, ok := t.(*crawler.TaskEntry); ok {
		rawUrl := task.RawUrl()
		log.Printf("处理任务：%s\n", task.RawUrl())
		s, err := task.Downloader().Download(rawUrl)
		if err != nil {
			return nil, nil, err
		}

		var detailsHrefs []string
		s.Find(".movie-list>div>a.box").Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			detailsHrefs = append(detailsHrefs, href)
		})
		if len(detailsHrefs) == 0 {
			return nil, nil, err
		}

		// 获取新任务列表
		var newTasks []crawler.WorkerTask
		for _, href := range detailsHrefs {
			m := new(table.Magnets)
			m.ResPath = task.RawURLPath
			if count, err := db.Instance().Count(m); err != nil {
				log.Printf("查询资源(%s)是否存在异常：%s\n", href, err.Error())
				continue
			} else if count > 0 {
				// 已经存在
				continue
			}

			// 添加详情解析任务
			newTasks = append(newTasks, crawler.NewWorkerTask(task.RawURLHost+href, &movieDetails{}))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(newTasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := s.Find(".pagination>a.pagination-next").First().Attr("href")
			if existsNext {
				// 提交下一页的任务，添加列表解析任务
				newTasks = append(newTasks, crawler.NewWorkerTask(task.RawURLHost+nextHref, &movieList{}))
			}
		}

		return newTasks, nil, err
	}
	return nil, nil, nil
}

type movieDetails struct {
}

func (p *movieDetails) Handle(t crawler.WorkerTask) (tasks []crawler.WorkerTask, outputs []crawler.Magnet, err error) {
	log.Printf("处理详情任务：%s\n", t.RawUrl())

	return nil, nil, err
	//ss := s.Find("section.section>div.container").First()
	//
	//// Title
	//var title = s.Find("title").Text()
	//// Number
	//var number = ss.Find(".movie-panel-info>div.first-block>span.value").Text()
	//// Links
	//var linksMap = make(map[string]string)
	//ss.Find("#magnets-content>.item>div>a").Each(func(i int, as *goquery.Selection) {
	//	if torrentUrl, exists := as.Attr("href"); exists {
	//		torrentName := as.Find("span.name").Text()
	//		tagsText := as.Find("div.tags").Text()
	//		if strings.Contains(tagsText, "高清") && strings.Contains(tagsText, "字幕") {
	//			log.Printf("高清字幕: %s => %s \n", torrentName, torrentUrl)
	//			linksMap[torrentUrl] = strings.ToUpper(torrentName)
	//		} else {
	//			log.Printf("非高清字幕: %s => %s \n", torrentName, torrentUrl)
	//		}
	//	}
	//})
	//
	//// Links clean
	//var links []string
	//for link, _ := range linksMap {
	//	links = append(links, link)
	//}
	//
	//if len(links) <= 0 {
	//	// Ignore
	//	return model2.NewEmptyOut(), nil
	//}
	//
	//// optimalLink
	//var optimalLink string
	//for link, linkName := range linksMap {
	//	if strings.Contains(linkName, "-UC") {
	//		optimalLink = link
	//		break
	//	}
	//}
	//if len(optimalLink) <= 0 {
	//	for link, linkName := range linksMap {
	//		if strings.Contains(linkName, "-C") {
	//			optimalLink = link
	//			break
	//		}
	//	}
	//	if len(optimalLink) <= 0 {
	//		for link, linkName := range linksMap {
	//			if strings.Contains(linkName, "-U") {
	//				optimalLink = link
	//				break
	//			}
	//		}
	//
	//		if len(optimalLink) <= 0 {
	//			optimalLink = links[0]
	//		}
	//	}
	//}
	//
	//log.Printf("Title: %s, Number: %s, OptimalLink: %s \n", title, number, optimalLink)
	//return model2.NewSingleOut(nil, &model2.Item{
	//	Title:       title,
	//	Number:      number,
	//	OptimalLink: optimalLink,
	//	Links:       links,
	//	ResHost:     meta.Host,
	//	ResPath:     meta.UrlPath,
	//}), nil
}
