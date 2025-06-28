package sehuatang

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/db/repository"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

const Name = "SeHuaTang"

type Seeder struct {
	downloader download.Downloader
}

var (
	pageIndex = 0
	// seeder实例
	seederSingleton = singleton.New[*Seeder](func() *Seeder {
		return &Seeder{
			downloader: download.NewClickBypassDownloader(
				func(root *goquery.Selection) bool {
					return root.Find("#hd").Size() == 0
				},
				func(page *rod.Page) error {
					btn := page.MustElementByJS(`() => document.querySelector("body > a:nth-child(5)")`)
					text, err := btn.Text()
					if err != nil {
						return err
					}
					log.Debugf("点击访问按钮: %s", text)
					btn.MustClick()
					return nil
				},
			),
		}
	})
)

func TaskSeeder() *Seeder {
	return seederSingleton.Get()
}

func (p *Seeder) Name() string {
	return Name
}

func (p *Seeder) Exec(cron *cron.Cron) {
	bus.Event().Publish(bus.SubmitTask.String(), task.NewTask(
		"https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=1",
		task.WithHandle(TaskSeeder()),
		task.WithDownloader(p.downloader),
	))
	log.Infof("启动任务：%s", p.Name())

	// 每天1点执行
	cron.AddFunc("55 1 * * *", func() {
		bus.Event().Publish(bus.SubmitTask.String(), task.NewTask(
			"https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=1",
			task.WithHandle(TaskSeeder()),
			task.WithDownloader(p.downloader),
		))
		log.Infof("启动任务：%s", p.Name())
	})
}

func (p *Seeder) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理任务：%s\n", rawUrl)

		var root *goquery.Selection
		var tableSelect *goquery.Selection
		root, err = taskEntry.Downloader().Download(rawUrl)
		if err != nil {
			return nil, nil, err
		}

		tableSelect = root.Find("#threadlisttableid")

		var detailsHrefs []string
		tableSelect.Find("[id^='normalthread_']").Each(func(i int, s *goquery.Selection) {
			aLink := s.Find("tr > th > a.s.xst").First()
			href, _ := aLink.Attr("href")
			detailsHrefs = append(detailsHrefs, href)
		})
		if len(detailsHrefs) == 0 {
			return nil, nil, nil
		}

		// 获取新任务列表
		var newTasks []task.Task
		for _, href := range detailsHrefs {
			if repository.ExistsByPath(href) {
				continue
			}

			// 添加详情解析任务
			newTasks = append(newTasks, task.NewTask(
				util.JoinUrl(taskEntry.RawURLHost, href),
				task.WithHandle(detailsHandler()),
				task.WithDownloader(p.downloader)))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(newTasks) > 0 && pageIndex <= 5 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := root.Find("#fd_page_bottom").First().Find("#fd_page_bottom > div > a:nth-child(2)").Attr("href")
			if existsNext {
				pageIndex++
				// 提交下一页的任务，添加列表解析任务
				newTasks = append(newTasks, task.NewTask(
					util.JoinUrl(taskEntry.RawURLHost, nextHref),
					task.WithHandle(TaskSeeder()),
					task.WithDownloader(p.downloader)))
			}
		}

		return newTasks, nil, nil
	}
	return nil, nil, nil
}
