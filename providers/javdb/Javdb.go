package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/nekoimi/get-magnet/common/model"
	"github.com/nekoimi/get-magnet/common/task"
	"github.com/nekoimi/get-magnet/pkg/db"
	"log"
	"strings"
)

// ChineseSubtitlesMovieList movie list parser
func ChineseSubtitlesMovieList(meta *task.Meta, selection *goquery.Selection) (*task.Out, error) {
	var detailsHrefArr []string
	selection.Find(".movie-list>div>a.box").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		detailsHrefArr = append(detailsHrefArr, href)
	})

	if len(detailsHrefArr) == 0 {
		log.Println("Details href arr is empty！")
		return task.NewEmptyOut(), nil
	}

	// 判断是否需要继续解析执行下一页
	// 需要判断details详情页是否处理过
	existsPathSet := make(map[string]int8)
	// 查询这些连接在数据库中是否存在
	var sqlArgs []any
	for _, s := range detailsHrefArr {
		sqlArgs = append(sqlArgs, s)
	}
	sql := "SELECT res_path FROM magnets WHERE res_path IN (?" + strings.Repeat(", ?", len(sqlArgs)-1) + ")"
	rs, err := db.Get().Query(sql, sqlArgs...)
	if err != nil {
		return task.NewEmptyOut(), err
	}
	defer rs.Close()
	for rs.Next() {
		var resPath string
		err := rs.Scan(&resPath)
		if err != nil {
			log.Printf("sql result err: %s \n", err.Error())
			continue
		}
		log.Printf("exists path: %s \n", resPath)

		existsPathSet[resPath] = 0
	}

	// 获取不存在的href列表
	var notExistsPathArr []string
	for _, href := range detailsHrefArr {
		if _, ok := existsPathSet[href]; !ok {
			notExistsPathArr = append(notExistsPathArr, href)
		}
	}

	var keepTasks []*task.Task

	// 当前新获取的path列表没有一个是存在于数据库记录的
	if len(existsPathSet) == 0 {
		// 不存在已经解析的link，继续下一页
		nextHref, existsNext := selection.Find(".pagination>a.pagination-next").First().Attr("href")
		if existsNext {
			// 提交下一页的任务
			log.Printf("nextHref: %s, fullNextUrl: %s \n", nextHref, meta.Host+nextHref)
			keepTasks = append(keepTasks, task.NewTask(meta.Host+nextHref, ChineseSubtitlesMovieList))
		}
	}

	for _, href := range notExistsPathArr {
		keepTasks = append(keepTasks, task.NewTask(meta.Host+href, MovieDetails))
	}

	return task.NewOut(keepTasks, nil), nil
}

// MovieDetails movie detail parser
func MovieDetails(meta *task.Meta, s *goquery.Selection) (*task.Out, error) {
	ss := s.Find("section.section>div.container").First()

	// Title
	var title = s.Find("title").Text()
	// Number
	var number = ss.Find(".movie-panel-info>div.first-block>span.value").Text()
	// Links
	var linksMap = make(map[string]string)
	ss.Find("#magnets-content>.item>div>a").Each(func(i int, as *goquery.Selection) {
		if torrentUrl, exists := as.Attr("href"); exists {
			torrentName := as.Find("span.name").Text()
			tagsText := as.Find("div.tags").Text()
			if strings.Contains(tagsText, "高清") && strings.Contains(tagsText, "字幕") {
				log.Printf("高清字幕: %s => %s \n", torrentName, torrentUrl)
				linksMap[torrentUrl] = strings.ToUpper(torrentName)
			} else {
				log.Printf("非高清字幕: %s => %s \n", torrentName, torrentUrl)
			}
		}
	})

	// Links clean
	var links []string
	for link, _ := range linksMap {
		links = append(links, link)
	}

	if len(links) <= 0 {
		// Ignore
		return task.NewEmptyOut(), nil
	}

	// optimalLink
	var optimalLink string
	for link, linkName := range linksMap {
		if strings.Contains(linkName, "-UC") {
			optimalLink = link
			break
		}
	}
	if len(optimalLink) <= 0 {
		for link, linkName := range linksMap {
			if strings.Contains(linkName, "-C") {
				optimalLink = link
				break
			}
		}
		if len(optimalLink) <= 0 {
			for link, linkName := range linksMap {
				if strings.Contains(linkName, "-U") {
					optimalLink = link
					break
				}
			}

			if len(optimalLink) <= 0 {
				optimalLink = links[0]
			}
		}
	}

	log.Printf("Title: %s, Number: %s, OptimalLink: %s \n", title, number, optimalLink)
	return task.NewSingleOut(nil, &model.Item{
		Title:       title,
		Number:      number,
		OptimalLink: optimalLink,
		Links:       links,
		ResHost:     meta.Host,
		ResPath:     meta.UrlPath,
	}), nil
}
