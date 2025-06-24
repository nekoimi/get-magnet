package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type Seeder struct {
}

func (p *Seeder) Name() string {
	return "JavDB"
}

func (p *Seeder) Exec(cron *cron.Cron) {
	// 每天2点执行
	cron.AddFunc("00 2 * * *", func() {
		bus.Event().Publish(bus.SubmitTask.String(), task.NewStaticWorkerTask("https://javdb.com/censored?vft=2&vst=1", &Seeder{}))
		log.Infoln("启动任务：https://javdb.com/censored?vft=2&vst=1")
	})

	// 每周执行
	cron.AddFunc("00 12 * * 0", func() {
		bus.Event().Publish(bus.SubmitTask.String(), task.NewStaticWorkerTask("https://javdb.com/actors/O2Q30?t=c&sort_type=0", &Seeder{}))
		bus.Event().Publish(bus.SubmitTask.String(), task.NewStaticWorkerTask("https://javdb.com/actors/x7wn?t=c&sort_type=0", &Seeder{}))
		bus.Event().Publish(bus.SubmitTask.String(), task.NewStaticWorkerTask("https://javdb.com/actors/0rva?t=c&sort_type=0", &Seeder{}))
	})
}

func (p *Seeder) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理任务：%s\n", taskEntry.RawUrl())
		s, err := taskEntry.Downloader().Download(rawUrl)
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
		var newTasks []task.Task
		for _, href := range detailsHrefs {
			m := new(table.Magnets)
			m.ResPath = href
			if count, err := db.Instance().Count(m); err != nil {
				log.Errorf("查询资源(%s)是否存在异常：%s\n", href, err.Error())
				continue
			} else if count > 0 {
				continue
			}

			// 添加详情解析任务
			newTasks = append(newTasks, task.NewStaticWorkerTask(taskEntry.RawURLHost+href, &movieDetails{}))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(newTasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := s.Find(".pagination>a.pagination-next").First().Attr("href")
			if existsNext {
				// 提交下一页的任务，添加列表解析任务
				newTasks = append(newTasks, task.NewStaticWorkerTask(taskEntry.RawURLHost+nextHref, &Seeder{}))
			}
		}

		return newTasks, nil, nil
	}
	return nil, nil, nil
}
