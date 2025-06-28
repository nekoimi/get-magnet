package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/db/repository"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"net/url"
)

type Seeder struct {
}

var (
	// seeder实例
	seederSingleton = singleton.New[*Seeder](func() *Seeder {
		return &Seeder{}
	})
)

func TaskSeeder() *Seeder {
	return seederSingleton.Get()
}

func (p *Seeder) Name() string {
	return "JavDB"
}

func (p *Seeder) Exec(cron *cron.Cron) {
	// 每天1点执行
	cron.AddFunc("00 1 * * *", func() {
		bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://javdb.com/censored?vft=2&vst=1", task.WithHandle(TaskSeeder())))
		log.Infof("启动任务：%s", p.Name())
	})

	// 每周执行
	cron.AddFunc("30 1 * * 0", func() {
		bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://javdb.com/actors/O2Q30?t=c&sort_type=0", task.WithHandle(TaskSeeder())))
		bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://javdb.com/actors/x7wn?t=c&sort_type=0", task.WithHandle(TaskSeeder())))
		bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://javdb.com/actors/0rva?t=c&sort_type=0", task.WithHandle(TaskSeeder())))
	})
}

func (p *Seeder) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理任务：%s\n", rawUrl)
		root, err := taskEntry.Downloader().Download(rawUrl)
		if err != nil {
			return nil, nil, err
		}

		var detailsHrefs []string
		root.Find(".movie-list>div>a.box").Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			detailsHrefs = append(detailsHrefs, href)
		})
		if len(detailsHrefs) == 0 {
			return nil, nil, nil
		}

		// 获取新任务列表
		var newTasks []task.Task
		for _, href := range detailsHrefs {
			joinUrl := util.JoinUrl(taskEntry.RawURLHost, href)

			u, err := url.Parse(joinUrl)
			if err != nil {
				return nil, nil, err
			}

			if repository.ExistsByPath(u.RequestURI()) {
				continue
			}

			// 添加详情解析任务
			newTasks = append(newTasks, task.NewTask(joinUrl, task.WithHandle(detailsHandler())))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(newTasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := root.Find(".pagination>a.pagination-next").First().Attr("href")
			if existsNext {
				// 提交下一页的任务，添加列表解析任务
				newTasks = append(newTasks, task.NewTask(util.JoinUrl(taskEntry.RawURLHost, nextHref), task.WithHandle(TaskSeeder())))
			}
		}

		return newTasks, nil, nil
	}
	return nil, nil, nil
}
