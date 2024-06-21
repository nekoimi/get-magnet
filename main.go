package main

import (
	"database/sql"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	JavdbRootDomain = "https://javdb.com"
	dsn             = "root:mysql#123456@(10.1.1.100:3306)/get_magnet_db"
)

var detailsChan = make(chan string, 2)
var stateChan = make(chan bool)

// Magnet 磁力信息实体 magnets
type Magnet struct {
	Id          int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Title       string
	Number      string
	OptimalLink string
	Links       []string
	ResHost     string
	ResPath     string
	Status      uint8
}

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)

	// tmp env
	_ = os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:2080")
	_ = os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:2080")
}

func loopDetails() {
	select {
	case href := <-detailsChan:
		log.Printf("details task: %s", href)
	case <-stateChan:
		break
	}
}

func main() {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalln(err)
	}

	// Start details task
	go loopDetails()

	// http client
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://javdb.com/censored?vft=2&vst=2", nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("respStatus: %d", resp.StatusCode)
	}

	//rawBody, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//body := string(rawBody)
	//
	//log.Println(body)

	// 解析其实列表页
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var detailsPathArr []string
	doc.Find(".movie-list>div>a.box").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		fullPath, err := url.JoinPath(JavdbRootDomain, href)
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("idx: %d, fullPath: %s \n", i, fullPath)

		detailsPathArr = append(detailsPathArr, href)
	})

	if len(detailsPathArr) == 0 {
		log.Println("Details path arr is empty！")
		return
	}

	// 查询这些连接在数据库中是否存在
	selectExistsDetailsSql := "SELECT res_path FROM magnets WHERE res_path IN (?)"
	rs, err := db.Query(selectExistsDetailsSql, strings.Join(detailsPathArr, ","))
	if err != nil {
		log.Fatalln(err)
	}
	defer rs.Close()

	existsPathSet := make(map[string]bool)
	for rs.Next() {
		var resPath string
		err := rs.Scan(&resPath)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("exists path: %s \n", resPath)

		existsPathSet[resPath] = true
	}

	// 判断是否需要继续解析执行下一页
	// 当前新获取的path列表没有一个是存在于数据库记录的
	if len(existsPathSet) == 0 {
		// 不存在已经解析的link，继续下一页
		nextHref, existsNext := doc.Find(".pagination>a.pagination-next").First().Attr("href")
		if existsNext {
			// 提交下一页的任务
			log.Printf("Next: %s \n", nextHref)
		}
	}

	// 解析详情页
	for _, href := range detailsPathArr {
		if !existsPathSet[href] {
			detailsChan <- href
			// 提交详情页的任务
			log.Printf("details: %s \n", href)
		}
	}

	for {
	}
}
