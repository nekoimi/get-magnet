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

	var detailsLinks []string
	doc.Find(".movie-list>div>a.box").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		fullLink, err := url.JoinPath(JavdbRootDomain, href)
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("idx: %d, fullLink: %s \n", i, fullLink)

		detailsLinks = append(detailsLinks, href)
	})

	// 查询这些连接在数据库中是否存在
	existsSql := "SELECT link FROM history WHERE link IN (?)"
	rs, err := db.Query(existsSql, strings.Join(detailsLinks, ","))
	if err != nil {
		log.Fatalln(err)
	}
	defer rs.Close()

	existsSet := make(map[string]bool)
	for rs.Next() {
		var rLink string
		err := rs.Scan(&rLink)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("exists fullLink: %s \n", rLink)

		existsSet[rLink] = true
	}

	// 判断是否需要继续解析执行下一页
	if len(existsSet) == 0 {
		// 不存在已经解析的link，继续下一页
		nextHref, existsNext := doc.Find(".pagination>a.pagination-next").First().Attr("href")
		if existsNext {
			log.Printf("Next: %s \n", nextHref)
		}
	}

	// 解析详情页
	for _, href := range detailsLinks {
		if !existsSet[href] {
			detailsChan <- href
			log.Printf("details: %s \n", href)
		}
	}

	for {
	}
}
