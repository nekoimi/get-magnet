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
)

type details struct {
}

var (
	// 实例
	detailsSingleton = singleton.New[*details](func() *details {
		return &details{}
	})
)

func detailsHandler() task.Handler {
	return detailsSingleton.Get()
}

func (p *details) Handle(t task.Task) (tasks []task.Task, outputs []task.MagnetEntry, err error) {
	if taskEntry, ok := t.(*task.Entry); ok {
		rawUrl := taskEntry.RawUrl()
		log.Infof("处理详情任务：%s\n", rawUrl)

		var u *url.URL
		u, err = url.Parse(rawUrl)
		if err != nil {
			return
		}

		// https://www.sehuatang.net/forum.php?mod=viewthread&tid=2862298&extra=page=1&filter=typeid&typeid=684
		if !u.Query().Has("tid") || !u.Query().Has("typeid") {
			return nil, nil, errors.New("详情页面url缺少id参数：" + rawUrl)
		}

		tid := u.Query().Get("tid")
		typeId := u.Query().Get("typeid")
		// 检查编号
		// Number: Name-typeId-tid
		var number = fmt.Sprintf("%s-%s-%s", Name, typeId, tid)
		if repository.ExistsByNumber(number) {
			log.Debugf("资源已经存在：%s - %s", number, rawUrl)
			return
		}

		var root *goquery.Selection
		root, err = taskEntry.Downloader().Download(rawUrl)
		if err != nil {
			return
		}

		// Title
		var title = root.Find("#thread_subject").Text()
		// optimalLink
		var optimalLink = root.Find("[id^='code_']").Find("ol > li").Text()
		log.Debugf("Title: %s, Number: %s, OptimalLink: %s \n", title, number, optimalLink)
		var links = []string{optimalLink}

		outputs = append(outputs, task.MagnetEntry{
			Origin:      Name,
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
