package sehuatang

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"net/http"
	"net/url"
	"sync"
)

const MaxRefreshCookieCount = 5

type Seeder struct {
	mux sync.Mutex
}

var (
	// seeder实例
	seederSingleton = singleton.New[*Seeder](func() *Seeder {
		return &Seeder{
			mux: sync.Mutex{},
		}
	})
)

func TaskSeeder() *Seeder {
	return seederSingleton.Get()
}

func (p *Seeder) Name() string {
	return "SeHuaTang"
}

func (p *Seeder) Exec(cron *cron.Cron) {
	// 每天1点执行
	cron.AddFunc("00 1 * * *", func() {
		bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://www.sehuatang.net/forum.php?mod=forumdisplay&fid=2&typeid=684&typeid=684&filter=typeid&page=1", task.WithHandle(TaskSeeder())))
		log.Infof("启动任务：%s", p.Name())
	})
}

func (p *Seeder) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理任务：%s\n", taskEntry.RawUrl())

		var refreshCookieNum = 1
		var root *goquery.Selection
		var tableSelect *goquery.Selection
		for {
			if refreshCookieNum > MaxRefreshCookieCount {
				return nil, nil, errors.New("刷新cookie次数太多: " + taskEntry.RawUrl())
			}

			root, err = taskEntry.Downloader().Download(rawUrl)
			if err != nil {
				return nil, nil, err
			}

			tableSelect = root.Find("#threadlisttableid")
			if tableSelect.Size() <= 0 {
				p.mux.Lock()
				log.Debugf("未获取到页面列表信息，尝试刷新cookies，retryNum(%d): %s", refreshCookieNum, taskEntry.RawUrl())

				func() {
					defer func() {
						p.mux.Unlock()
						if r := recover(); r != nil {
							log.Errorf("刷新cookies异常: %s - %v", taskEntry.RawUrl(), r)
						}
					}()

					refreshCookies(taskEntry)

					refreshCookieNum++
				}()

				continue
			}

			break
		}

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
			m := new(table.Magnets)
			m.RawURLPath = href
			if count, err := db.Instance().Count(m); err != nil {
				log.Errorf("查询资源(%s)是否存在异常：%s\n", href, err.Error())
				continue
			} else if count > 0 {
				continue
			}

			// 添加详情解析任务
			newTasks = append(newTasks, task.NewTask(taskEntry.RawURLHost+href, task.WithHandle(detailsHandler())))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(newTasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := root.Find("#fd_page_bottom").First().Find("#fd_page_bottom > div > a:nth-child(2)").Attr("href")
			if existsNext {
				// 提交下一页的任务，添加列表解析任务
				newTasks = append(newTasks, task.NewTask(taskEntry.RawURLHost+nextHref, task.WithHandle(TaskSeeder())))
			}
		}

		return newTasks, nil, nil
	}
	return nil, nil, nil
}

func refreshCookies(task *task.Entry) {
	proxyEnv := httpproxy.FromEnvironment()
	launch := launcher.New().Proxy(proxyEnv.HTTPProxy).MustLaunch()
	page := rod.New().ControlURL(launch).MustConnect().MustPage(task.RawUrl())
	// 等待页面加载
	page.MustWaitStable()
	btn := page.MustElementByJS(`() => document.querySelector("body > a:nth-child(5)")`)
	log.Debugf("点击访问按钮: %s", task.RawUrl())
	btn.MustClick()
	// 等待加载
	page.MustWaitLoad()

	// 保存 cookie
	cookies := page.MustCookies()
	u, err := url.Parse(task.RawUrl())
	if err != nil {
		panic(err)
	}

	var stdCookies []*http.Cookie
	for _, c := range cookies {
		stdCookies = append(stdCookies, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		})
	}
	// 设置cookies
	task.Downloader().SetCookies(u, stdCookies)
	log.Debugf("刷新cookies完成: %s - size:%d", task.RawUrl(), len(stdCookies))
}
