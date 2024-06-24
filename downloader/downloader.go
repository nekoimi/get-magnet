package downloader

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"time"
)

func Download(url string) (selection *goquery.Selection, err error) {
	return nil, errors.New("fsfsd")
}

func A() {
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
