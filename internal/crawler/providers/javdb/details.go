package javdb

import (
	"github.com/nekoimi/get-magnet/internal/crawler"
	"log"
)

type movieDetails struct {
}

func (p *movieDetails) Handle(t crawler.WorkerTask) (tasks []crawler.WorkerTask, outputs []crawler.Magnet, err error) {
	log.Printf("处理详情任务：%s\n", t.RawUrl())

	return nil, nil, err
	//ss := s.Find("section.section>div.container").First()
	//
	//// Title
	//var title = s.Find("title").Text()
	//// Number
	//var number = ss.Find(".movie-panel-info>div.first-block>span.value").Text()
	//// Links
	//var linksMap = make(map[string]string)
	//ss.Find("#magnets-content>.item>div>a").Each(func(i int, as *goquery.Selection) {
	//	if torrentUrl, exists := as.Attr("href"); exists {
	//		torrentName := as.Find("span.name").Text()
	//		tagsText := as.Find("div.tags").Text()
	//		if strings.Contains(tagsText, "高清") && strings.Contains(tagsText, "字幕") {
	//			log.Printf("高清字幕: %s => %s \n", torrentName, torrentUrl)
	//			linksMap[torrentUrl] = strings.ToUpper(torrentName)
	//		} else {
	//			log.Printf("非高清字幕: %s => %s \n", torrentName, torrentUrl)
	//		}
	//	}
	//})
	//
	//// Links clean
	//var links []string
	//for link, _ := range linksMap {
	//	links = append(links, link)
	//}
	//
	//if len(links) <= 0 {
	//	// Ignore
	//	return model2.NewEmptyOut(), nil
	//}
	//
	//// optimalLink
	//var optimalLink string
	//for link, linkName := range linksMap {
	//	if strings.Contains(linkName, "-UC") {
	//		optimalLink = link
	//		break
	//	}
	//}
	//if len(optimalLink) <= 0 {
	//	for link, linkName := range linksMap {
	//		if strings.Contains(linkName, "-C") {
	//			optimalLink = link
	//			break
	//		}
	//	}
	//	if len(optimalLink) <= 0 {
	//		for link, linkName := range linksMap {
	//			if strings.Contains(linkName, "-U") {
	//				optimalLink = link
	//				break
	//			}
	//		}
	//
	//		if len(optimalLink) <= 0 {
	//			optimalLink = links[0]
	//		}
	//	}
	//}
	//
	//log.Printf("Title: %s, Number: %s, OptimalLink: %s \n", title, number, optimalLink)
	//return model2.NewSingleOut(nil, &model2.Item{
	//	Title:       title,
	//	Number:      number,
	//	OptimalLink: optimalLink,
	//	Links:       links,
	//	ResHost:     meta.Host,
	//	ResPath:     meta.UrlPath,
	//}), nil
}
