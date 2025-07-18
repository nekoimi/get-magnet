package download

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"time"
)

const RetryLimit = 5

type Downloader interface {
	SetCookies(u *url.URL, cookies []*http.Cookie)
	Download(url string) (*goquery.Selection, error)
}

type HttpDownloader struct {
	cookieMux sync.Mutex
	client    *http.Client
}

var (
	once sync.Once
	// 默认下载器
	defaultDownloader Downloader
)

func NewHttpDownloader() Downloader {
	once.Do(func() {
		jar, _ := cookiejar.New(nil)
		defaultDownloader = &HttpDownloader{
			cookieMux: sync.Mutex{},
			client: &http.Client{
				Jar:     jar,
				Timeout: 10 * time.Second,
			},
		}
	})

	return defaultDownloader
}

func (s *HttpDownloader) SetCookies(u *url.URL, cookies []*http.Cookie) {
	s.cookieMux.Lock()
	defer s.cookieMux.Unlock()

	s.client.Jar.SetCookies(u, cookies)
}

func (s *HttpDownloader) Download(rawUrl string) (selection *goquery.Selection, err error) {
	var req *http.Request
	var resp *http.Response
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	retryNum := 1
	for {
		if retryNum > RetryLimit {
			break
		}
		log.Debugf("download url - retryNum(%d): %s", retryNum, rawUrl)
		retryNum++

		req, err = http.NewRequest("GET", rawUrl, nil)
		if err != nil {
			log.Errorf("download url error -> continue: %s - %s", rawUrl, err.Error())
			continue
		}
		req.Header.Set("Referer", rawUrl)
		req.Header.Set("User-Agent", randUserAgent())
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7,ja;q=0.6")
		resp, err = s.client.Do(req)
		if err != nil {
			log.Errorf("download url error -> continue: %s - %s", rawUrl, err.Error())
			continue
		}

		if resp.StatusCode == 429 {
			// 请求次数过多，等待一段时间
			time.Sleep(60 * time.Second)
			log.Errorf("download url StatusCode == 429 -> wait continue: %s - %s", rawUrl, err.Error())
			continue
		}

		if resp.StatusCode == 200 {
			break
		}

		err = errors.New(fmt.Sprintf("download url error: %s, StatusCode not ok => %d", rawUrl, resp.StatusCode))
	}

	// 检查处理错误
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc.Selection, nil
}

// 随机User-Agent
func randUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36 OPR/26.0.1656.60",
		"Mozilla/5.0 (Windows NT 5.1; U; en; rv:1.8.1) Gecko/20061208 Firefox/2.0.0 Opera 9.50",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/534.57.2 (KHTML, like Gecko) Version/5.1.7 Safari/534.57.2",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36",
		"Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; WOW64; Trident/5.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; Media Center PC 6.0; .NET4.0C; .NET4.0E; QQBrowser/7.0.3698.400)",
	}
	rand.New(rand.NewSource(time.Now().UnixNano()))
	ri := rand.Intn(len(userAgents))
	return userAgents[ri]
}
