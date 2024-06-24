package javdb

import (
	"get-magnet/pkg/db"
	"get-magnet/scheduler"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/url"
	"strings"
)

const JavdbRootDomain = "https://javdb.com"

const (
	SelectDetailsHrefExistsSql = "SELECT res_path FROM magnets WHERE res_path IN (?)"
)

// ParseMovieList movie list parser
func ParseMovieList(selection *goquery.Selection) (scheduler.TaskOut, error) {
	var detailsHrefArr []string
	selection.Find(".movie-list>div>a.box").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		detailsHrefArr = append(detailsHrefArr, href)
	})

	if len(detailsHrefArr) == 0 {
		log.Println("Details href arr is empty！")
		return scheduler.TaskOut{}, nil
	}

	var keepTasks = make([]scheduler.Task, len(detailsHrefArr))
	out := scheduler.TaskOut{
		Tasks: keepTasks,
		Items: nil,
	}

	// 判断是否需要继续解析执行下一页
	// 需要判断details详情页是否处理过
	existsPathSet := make(map[string]bool)
	// 查询这些连接在数据库中是否存在
	rs, err := db.GetDb().Query(SelectDetailsHrefExistsSql, strings.Join(detailsHrefArr, ","))
	if err != nil {
		log.Fatalln(err)
	}
	defer rs.Close()
	for rs.Next() {
		var resPath string
		err := rs.Scan(&resPath)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("exists path: %s \n", resPath)

		existsPathSet[resPath] = true
	}

	// 当前新获取的path列表没有一个是存在于数据库记录的
	if len(existsPathSet) == 0 {
		// 不存在已经解析的link，继续下一页
		nextHref, existsNext := selection.Find(".pagination>a.pagination-next").First().Attr("href")
		if existsNext {
			// 提交下一页的任务
			fullNextUrl, err := url.JoinPath(JavdbRootDomain, nextHref)
			if err != nil {
				log.Fatalln(err)
			}
			log.Printf("fullNextUrl: %s \n", fullNextUrl)
			keepTasks = append(keepTasks, scheduler.Task{
				Url:    fullNextUrl,
				Handle: ParseMovieList,
			})
		}
	}

	for _, href := range detailsHrefArr {
		if !existsPathSet[href] {
			fullDetailsUrl, err := url.JoinPath(JavdbRootDomain, href)
			if err != nil {
				log.Fatalln(err)
			}
			log.Printf("fullDetailsUrl: %s \n", fullDetailsUrl)
			// append task list
			keepTasks = append(keepTasks, scheduler.Task{
				Url:    fullDetailsUrl,
				Handle: ParseMovieDetails,
			})
		}
	}

	return out, nil
}

// ParseMovieDetails movie detail parser
func ParseMovieDetails(selection *goquery.Selection) (scheduler.TaskOut, error) {
	return scheduler.TaskOut{}, nil
}
