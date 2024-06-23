package main

import (
	"get-magnet/engine"
	"get-magnet/handlers/javdb"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"os"
	"time"
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
	e := engine.Default()
	e.Scheduler.Submit(engine.Task{
		Url:    "https://javdb.com/censored?vft=2&vst=2",
		Handle: javdb.ParseMovieList,
	})
	e.Run()

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
	_, err = goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
}
