package download

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"net/http"
	"net/url"
	"os"
	"sync"
)

const MaxRetryBypassCount = 5

type ClickBypassDownloader struct {
	bypassMux sync.Mutex
	// 下载器
	downloader Downloader
	// 判断需不需要绕过验证函数，需要返回true，不需要返回false
	shouldBypassFunc func(root *goquery.Selection) bool
	// 绕过验证逻辑
	handleBypassFunc func(page *rod.Page) error
}

func NewClickBypassDownloader(shouldBypassFunc func(root *goquery.Selection) bool, handleBypassFunc func(page *rod.Page) error) Downloader {
	return &ClickBypassDownloader{
		bypassMux:        sync.Mutex{},
		downloader:       Default(),
		shouldBypassFunc: shouldBypassFunc,
		handleBypassFunc: handleBypassFunc,
	}
}

func (s *ClickBypassDownloader) SetCookies(u *url.URL, cookies []*http.Cookie) {
	s.downloader.SetCookies(u, cookies)
}

func (s *ClickBypassDownloader) Download(rawUrl string) (selection *goquery.Selection, err error) {
	var retryBypassNum = 1
	var root *goquery.Selection
	for {
		if retryBypassNum > MaxRetryBypassCount {
			return nil, errors.New("点击过验证重试次数太多: " + rawUrl)
		}

		root, err = s.downloader.Download(rawUrl)
		if err != nil {
			return nil, err
		}

		if s.shouldBypassFunc(root) {
			// 需要点击过验证
			func() {
				s.bypassMux.Lock()
				log.Debugf("未获取到页面信息，尝试点击验证刷新cookies，retryNum(%d): %s", retryBypassNum, rawUrl)
				defer func() {
					s.bypassMux.Unlock()
					retryBypassNum++

					if r := recover(); r != nil {
						log.Errorf("刷新cookies异常: %s - %v", rawUrl, r)
					}
				}()

				s.handleBypassRefreshCookies(rawUrl)
			}()

			// 绕过验证后，继续下一次
			continue
		}

		// 不需要验证，直接跳出
		break
	}

	return root, nil
}

func (s *ClickBypassDownloader) handleBypassRefreshCookies(rawUrl string) {
	proxyEnv := httpproxy.FromEnvironment()
	launch := launcher.New().Bin(os.Getenv("ROD_BROWSER_PATH")).Proxy(proxyEnv.HTTPProxy).MustLaunch()
	log.Debugf("启动页面 %s 浏览...", rawUrl)
	browser := rod.New().ControlURL(launch).MustConnect()
	defer func() {
		if err := browser.Close(); err != nil {
			log.Errorf("退出浏览页面异常：%s", err.Error())
			return
		}
		log.Debugf("退出页面 %s 浏览...", rawUrl)
	}()

	page := browser.MustPage(rawUrl)
	// 等待页面加载
	log.Debugf("等待页面 %s 加载...", rawUrl)
	page.MustWaitStable()

	// 执行绕过验证逻辑
	if err := s.handleBypassFunc(page); err != nil {
		log.Debugf("执行绕过验证 %s 异常：%s", rawUrl, err.Error())
		panic(err)
	}

	// 等待加载
	page.MustWaitLoad()

	// 保存 cookie
	cookies := page.MustCookies()
	u, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}

	var stdCookies []*http.Cookie
	for _, c := range cookies {
		stdCookies = append(stdCookies, &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires.Time(),
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		})
	}
	// 设置cookies
	s.SetCookies(u, stdCookies)
	log.Debugf("刷新cookies完成: %s - size: %d", rawUrl, len(stdCookies))
}
