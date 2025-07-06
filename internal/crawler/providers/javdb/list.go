package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/db/repository"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
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

func (p *Seeder) Exec() {
	job.Register("05 3 * * *", &job.Job{
		Name: "JavDB",
		Cmd: func() {
			bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://javdb.com/censored?vft=2&vst=1",
				task.WithHandle(TaskSeeder()),
				task.WithDownloader(GetBypassDownloader()),
			))
		},
	})

	job.Register("30 3 * * 0", &job.Job{
		Name: "JavDB-Actors",
		Cmd: func() {
			bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://javdb.com/actors/O2Q30?t=c&sort_type=0",
				task.WithHandle(TaskSeeder()),
				task.WithDownloader(GetBypassDownloader()),
			))

			bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://javdb.com/actors/x7wn?t=c&sort_type=0",
				task.WithHandle(TaskSeeder()),
				task.WithDownloader(GetBypassDownloader()),
			))
		},
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
			html, _ := root.Html()
			log.Debugf("任务列表未获取到有效任务信息，源页面：\n %s", html)
			return nil, nil, nil
		}

		// 获取新任务列表
		var newTasks []task.Task
		for _, href := range detailsHrefs {
			joinUrl := util.JoinUrl(taskEntry.RawURLHost, href)

			u, err := url.Parse(joinUrl)
			if err != nil {
				log.Errorf("url解析错误：%s", joinUrl)
				continue
			}

			if repository.ExistsByPath(u.RequestURI()) {
				continue
			}

			// 添加详情解析任务
			newTasks = append(newTasks, task.NewTask(joinUrl,
				task.WithHandle(detailsHandler()),
				task.WithDownloader(GetBypassDownloader()),
			))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(newTasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := root.Find(".pagination>a.pagination-next").First().Attr("href")
			if existsNext {
				// 提交下一页的任务，添加列表解析任务
				newTasks = append(newTasks, task.NewTask(util.JoinUrl(taskEntry.RawURLHost, nextHref),
					task.WithHandle(TaskSeeder()),
					task.WithDownloader(GetBypassDownloader()),
				))
			}
		}

		return newTasks, nil, nil
	}
	return nil, nil, nil
}
