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

func Handler() crawler.WorkerTaskHandler {
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
			return nil, nil, nil
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
			newTasks = append(newTasks, crawler.NewStaticWorkerTask(task.RawURLHost+href, &movieDetails{}))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(newTasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := s.Find(".pagination>a.pagination-next").First().Attr("href")
			if existsNext {
				// 提交下一页的任务，添加列表解析任务
				newTasks = append(newTasks, crawler.NewStaticWorkerTask(task.RawURLHost+nextHref, &movieList{}))
			}
		}

		return newTasks, nil, nil
	}
	return nil, nil, nil
}
