package sehuatang

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/db/repository"
	"github.com/nekoimi/get-magnet/internal/job"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	log "github.com/sirupsen/logrus"
	"net/url"
)

const Name = "SeHuaTang"
const FC2PPV = "FC2PPV"

type Seeder struct {
	downloader download.Downloader
}

var (
	// seeder实例
	seederSingleton = singleton.New[*Seeder](func() *Seeder {
		return &Seeder{
			downloader: GetBypassDownloader(),
		}
	})
)

func TaskSeeder() *Seeder {
	return seederSingleton.Get()
}

func (p *Seeder) Name() string {
	return Name
}

func (p *Seeder) Downloader() download.Downloader {
	return p.downloader
}

func (p *Seeder) Exec() {
	job.Register("50 3 * * *", &job.Job{
		Name: Name,
		Cmd: func() {
			bus.Event().Publish(bus.SubmitTask.String(), task.NewTask(
				"https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=1",
				task.WithHandle(TaskSeeder()),
				task.WithDownloader(GetBypassDownloader()),
			))
		},
	})

	job.Register("55 3 * * *", &job.Job{
		Name: "FC2PPV",
		Cmd: func() {
			// FC2PPV
			bus.Event().Publish(bus.SubmitTask.String(), task.NewTask(
				"https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=36&filter=typeid&typeid=368",
				task.WithHandle(TaskSeeder()),
				task.WithDownloader(GetBypassDownloader()),
			))
		},
	})
}

func (p *Seeder) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理任务：%s", rawUrl)

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
			log.Warnf("任务列表未获取到有效任务信息，源页面：\n %s", rawUrl)
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
				log.Debugf("请求地址已经处理过了：%s", u.RequestURI())
				continue
			}

			// 添加详情解析任务
			newTasks = append(newTasks, task.NewTask(
				joinUrl,
				task.WithHandle(detailsHandler()),
				task.WithDownloader(GetBypassDownloader())))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(newTasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := root.Find("#fd_page_bottom").First().Find("#fd_page_bottom > div > a:nth-child(2)").Attr("href")
			if existsNext {
				log.Debugf("处理下一页：%s", nextHref)
				// 提交下一页的任务，添加列表解析任务
				newTasks = append(newTasks, task.NewTask(
					util.JoinUrl(taskEntry.RawURLHost, nextHref),
					task.WithHandle(TaskSeeder()),
					task.WithDownloader(GetBypassDownloader())))
			}
		}

		return newTasks, nil, nil
	}
	return nil, nil, nil
}
