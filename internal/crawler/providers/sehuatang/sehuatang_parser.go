package sehuatang

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	log "github.com/sirupsen/logrus"
	"net/url"
	"regexp"
	"strings"
)

const Fc2TypeId = "368"

var (
	// 编号正则
	fc2NumberRe = regexp.MustCompile(`(FC2PPV-\d{5,10})`)
)

type Parser struct {
	// 下载器
	downloader download.Downloader
}

func (p *Parser) parseList(t crawler.CrawlerTask) (tasks []crawler.CrawlerTask, outputs []crawler.MagnetEntry, err error) {
	if taskEntry, ok := t.(*crawler.TaskEntry); ok {
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
				crawler.WithDownloader(p.downloader)))
		}

		// 当前新获取的path列表存在需要处理的新任务
		if len(tasks) > 0 {
			// 不存在已经解析的link，继续下一页
			nextHref, existsNext := root.Find("#fd_page_bottom").First().Find("#fd_page_bottom > div > a:nth-child(2)").Attr("href")
			if existsNext {
				nextUrl := util.JoinUrl(taskEntry.RawURLHost, nextHref)
				log.Debugf("处理下一页：%s", nextUrl)
				// 提交下一页的任务，添加列表解析任务
				tasks = append(tasks, crawler.NewCrawlerTask(
					nextUrl,
					taskEntry.Origin,
					crawler.WithHandle(p.parseList),
					crawler.WithDownloader(p.downloader)))
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

		var u *url.URL
		u, err = url.Parse(rawUrl)
		if err != nil {
			log.Errorf("处理详情任务error：%s -> %s", rawUrl, err.Error())
			return
		}

		// https://www.sehuatang.net/forum.php?mod=viewthread&tid=2862298&extra=page=1&filter=typeid&typeid=684
		if !u.Query().Has("tid") || !u.Query().Has("typeid") {
			return nil, nil, fmt.Errorf("详情页面url缺少id参数：%s", rawUrl)
		}

		tid := u.Query().Get("tid")
		typeId := u.Query().Get("typeid")

		// 检查编号
		var number string

		if Fc2TypeId != typeId {
			// 不是fc2 使用自定义编号规则
			// Number: Name-typeId-tid
			number = fmt.Sprintf("%s-%s-%s", Name, typeId, tid)
			if magnet_repo.ExistsByNumber(number) {
				log.Debugf("资源已经存在：%s - %s", number, rawUrl)
				return
			}
		}

		var root *goquery.Selection
		root, err = taskEntry.Downloader().Download(rawUrl)
		if err != nil {
			log.Errorf("处理详情任务error：%s -> %s", rawUrl, err.Error())
			return
		}

		var origin = Name

		// Title
		var title = root.Find("#thread_subject").Text()
		if strings.Contains(title, FC2PPV) {
			origin = FC2PPV
			// 重新获取编号
			number = fc2NumberRe.FindString(strings.ToUpper(title))
			if number == "" {
				log.Warnf("FC2资源未匹配到编号，忽略：%s - %s", title, rawUrl)
				return
			}

			if magnet_repo.ExistsByNumber(number) {
				log.Debugf("FC2资源已经存在：%s - %s", number, rawUrl)
				return
			}
		}

		// optimalLink
		var optimalLink = root.Find("[id^='code_']").Find("ol > li").Text()
		log.Debugf("Title: %s, Number: %s, OptimalLink: %s", title, number, optimalLink)
		var links = []string{optimalLink}

		outputs = append(outputs, crawler.MagnetEntry{
			Origin:      origin,
			Title:       title,
			Number:      number,
			OptimalLink: optimalLink,
			Links:       links,
			RawURLHost:  taskEntry.RawURLHost,
			RawURLPath:  taskEntry.RawURLPath,
		})
	}
	return
}
