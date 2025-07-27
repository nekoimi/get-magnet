package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	log "github.com/sirupsen/logrus"
	"net/url"
	"strings"
)

var (
	// 资源过滤优选排序关键字集合
	torrentFilterKeys = map[string]int{
		"-UC.":            1,
		"-C.":             2,
		"-U.":             3,
		"-UNCENSORED-HD.": 4,
		"-AI.":            5,
	}
)

type Parser struct {
	// 下载器
	downloader download.Downloader
}

func (p *Parser) parseList(t crawler.CrawlerTask) (tasks []crawler.CrawlerTask, outputs []crawler.MagnetEntry, err error) {
	if taskEntry, ok := t.(*crawler.TaskEntry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理任务：%s", rawUrl)
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
		for _, href := range detailsHrefs {
			joinUrl := util.JoinUrl(taskEntry.RawURLHost, href)

			u, err := url.Parse(joinUrl)
			if err != nil {
				log.Errorf("url解析错误：%s", joinUrl)
				continue
			}

			if magnet_repo.ExistsByPath(u.RequestURI()) {
				log.Debugf("请求地址已经处理过了：%s", u.RequestURI())
				continue
			}

			// 添加详情解析任务
			tasks = append(tasks, crawler.NewCrawlerTask(
				joinUrl,
				taskEntry.Origin,
				crawler.WithHandle(p.parsePage),
				crawler.WithDownloader(p.downloader),
			))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(tasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := root.Find(".pagination>a.pagination-next").First().Attr("href")
			if existsNext {
				nextUrl := util.JoinUrl(taskEntry.RawURLHost, nextHref)
				log.Debugf("处理下一页：%s", nextUrl)
				// 提交下一页的任务，添加列表解析任务
				tasks = append(tasks, crawler.NewCrawlerTask(
					nextUrl,
					taskEntry.Origin,
					crawler.WithHandle(p.parseList),
					crawler.WithDownloader(p.downloader),
				))
			}
		}

		return tasks, nil, nil
	}
	return nil, nil, nil
}

func (p *Parser) parsePage(t crawler.CrawlerTask) (tasks []crawler.CrawlerTask, outputs []crawler.MagnetEntry, err error) {
	if taskEntry, ok := t.(*crawler.TaskEntry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理详情任务：%s", rawUrl)

		var root *goquery.Selection
		root, err = taskEntry.Downloader().Download(rawUrl)
		if err != nil {
			log.Errorf("处理详情任务error：%s -> %s", rawUrl, err.Error())
			return
		}

		s := root.Find("section.section>div.container").First()
		// Title
		var title = s.Find("div.video-detail > h2").Text()
		title = strings.ReplaceAll(title, "\n", "")
		title = strings.ReplaceAll(title, "  ", "")
		title = strings.TrimSpace(title)
		// Number
		var number = s.Find(".movie-panel-info>div.first-block>span.value").Text()
		if magnet_repo.ExistsByNumber(number) {
			// 已经存在了
			log.Debugf("处理详情任务 number已经存在：%s -> %s", rawUrl, number)
			return
		}
		// Actress0
		var actress []string
		s.Find("nav.panel.movie-panel-info > div.panel-block").Each(func(i int, sub *goquery.Selection) {
			titleVal := sub.Find("strong").Text()
			if strings.Contains(titleVal, "演員") || strings.Contains(titleVal, "Actor") {
				var (
					currALink         *goquery.Selection
					currALinkNext     *goquery.Selection
					aLinkText         string
					aLinkTextNextText string
					ok                bool
				)
				currALink = sub.Find("span.value").Find("a").First()
				for {
					aLinkText = strings.TrimSpace(currALink.Text())
					if aLinkText == "" {
						break
					}
					currALinkNext = currALink.Next()
					if aLinkTextNextText, ok = currALinkNext.Attr("class"); ok {
						if strings.Contains(aLinkTextNextText, "female") {
							actress = append(actress, aLinkText)
						}
						currALink = currALinkNext.Next()
					} else {
						break
					}
				}
			}
		})
		actress0 := strings.Join(actress, ",")

		// TorrentLinks
		var torrentLinks = make([]crawler.TorrentLink, 0)
		s.Find("#magnets-content>.item>div>a").Each(func(i int, as *goquery.Selection) {
			if torrentUrl, exists := as.Attr("href"); exists {
				torrentName := strings.ToUpper(as.Find("span.name").Text())
				for key, sort := range torrentFilterKeys {
					if strings.Contains(torrentName, key) {
						torrentLinks = append(torrentLinks, crawler.TorrentLink{
							Sort: sort,
							Name: torrentName,
							Link: torrentUrl,
						})
					}
				}
			}
		})
		if len(torrentLinks) == 0 {
			log.Debugf("处理详情任务 torrentLinks == 0：%s", rawUrl)
			return
		}

		// 优选排序
		util.Sort[crawler.TorrentLink](torrentLinks, func(a crawler.TorrentLink, b crawler.TorrentLink) bool {
			return a.Sort < b.Sort
		})

		// optimalLink
		var optimalLink = torrentLinks[0].Link
		log.Debugf("Title: %s, Number: %s, OptimalLink: %s", title, number, optimalLink)

		var links []string
		for _, link := range torrentLinks {
			links = append(links, link.Link)
		}
		outputs = append(outputs, crawler.MagnetEntry{
			Origin:      taskEntry.Origin,
			Title:       title,
			Number:      number,
			Actress0:    actress0,
			OptimalLink: optimalLink,
			Links:       links,
			RawURLHost:  taskEntry.RawURLHost,
			RawURLPath:  taskEntry.RawURLPath,
		})

		return
	}
	return
}
