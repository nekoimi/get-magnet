package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	log "github.com/sirupsen/logrus"
	"strings"
)

type details struct {
}

var (
	// 实例
	detailsSingleton = singleton.New[*details](func() *details {
		return &details{}
	})
	// 资源过滤优选排序关键字集合
	torrentFilterKeys = map[string]int{
		"-UC.":            1,
		"-C.":             2,
		"-U.":             3,
		"-UNCENSORED-HD.": 4,
		"-AI.":            5,
	}
)

func detailsHandler() task.Handler {
	return detailsSingleton.Get()
}

func (p *details) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理详情任务：%s\n", taskEntry.RawUrl())

		var root *goquery.Selection
		root, err = taskEntry.Downloader().Download(rawUrl)
		if err != nil {
			return
		}

		s := root.Find("section.section>div.container").First()
		// Title
		var title = root.Find("title").Text()
		// Number
		var number = s.Find(".movie-panel-info>div.first-block>span.value").Text()

		// TorrentLinks
		var torrentLinks = make([]task.TorrentLink, 0)
		s.Find("#magnets-content>.item>div>a").Each(func(i int, as *goquery.Selection) {
			if torrentUrl, exists := as.Attr("href"); exists {
				torrentName := strings.ToUpper(as.Find("span.name").Text())
				for key, sort := range torrentFilterKeys {
					if strings.Contains(torrentName, key) {
						torrentLinks = append(torrentLinks, task.TorrentLink{
							Sort: sort,
							Name: torrentName,
							Link: torrentUrl,
						})
					}
				}
			}
		})
		if len(torrentLinks) <= 0 {
			return
		}

		// 优选排序
		util.Sort[task.TorrentLink](torrentLinks, func(a task.TorrentLink, b task.TorrentLink) bool {
			return a.Sort < b.Sort
		})

		// optimalLink
		var optimalLink = torrentLinks[0].Link
		log.Debugf("Title: %s, Number: %s, OptimalLink: %s \n", title, number, optimalLink)

		var links []string
		for _, link := range torrentLinks {
			links = append(links, link.Link)
		}
		outputs = append(outputs, task.MagnetEntry{
			Title:       title,
			Number:      number,
			OptimalLink: optimalLink,
			Links:       links,
			ResHost:     taskEntry.RawURLHost,
			ResPath:     taskEntry.RawURLPath,
		})

		return
	}
	return
}
