package sehuatang

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/db/repository"
	"github.com/nekoimi/get-magnet/internal/pkg/singleton"
	log "github.com/sirupsen/logrus"
	"net/url"
	"regexp"
	"strings"
)

type details struct {
}

const Fc2TypeId = "368"

var (
	// 实例
	detailsSingleton = singleton.New[*details](func() *details {
		return &details{}
	})
	// 编号正则
	fc2NumberRe = regexp.MustCompile(`(FC2PPV-\d{5,10})`)
)

func detailsHandler() task.Handler {
	return detailsSingleton.Get()
}

func (p *details) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
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
			return nil, nil, errors.New("详情页面url缺少id参数：" + rawUrl)
		}

		tid := u.Query().Get("tid")
		typeId := u.Query().Get("typeid")

		// 检查编号
		var number string

		if Fc2TypeId != typeId {
			// 不是fc2 使用自定义编号规则
			// Number: Name-typeId-tid
			number = fmt.Sprintf("%s-%s-%s", Name, typeId, tid)
			if repository.ExistsByNumber(number) {
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

			if repository.ExistsByNumber(number) {
				log.Debugf("FC2资源已经存在：%s - %s", number, rawUrl)
				return
			}
		}

		// optimalLink
		var optimalLink = root.Find("[id^='code_']").Find("ol > li").Text()
		log.Debugf("Title: %s, Number: %s, OptimalLink: %s", title, number, optimalLink)
		var links = []string{optimalLink}

		outputs = append(outputs, task.MagnetEntry{
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
