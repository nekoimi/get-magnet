package download

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const MaxRetryLoginBypassCount = 5
const MaxRetryLoginErrorCount = 5

// LoginBypassDownloader 登录绕过验证
type LoginBypassDownloader struct {
	loginMux sync.Mutex
	// 下载器
	downloader Downloader
	// 判断需不需要绕过验证函数，需要返回true，不需要返回false
	shouldLoginFunc func(root *goquery.Selection) bool
	// 绕过验证逻辑
	handleLoginFunc func(page *rod.Page) error
}

func NewLoginBypassDownloader(downloader Downloader, shouldLoginFunc func(root *goquery.Selection) bool, handleLoginFunc func(page *rod.Page) error) Downloader {
	return &LoginBypassDownloader{
		loginMux:        sync.Mutex{},
		downloader:      downloader,
		shouldLoginFunc: shouldLoginFunc,
		handleLoginFunc: handleLoginFunc,
	}
}

func (s *LoginBypassDownloader) SetCookies(u *url.URL, cookies []*http.Cookie) {
	s.downloader.SetCookies(u, cookies)
}

func (s *LoginBypassDownloader) Download(rawUrl string) (selection *goquery.Selection, err error) {
	var retryBypassNum = 1
	var root *goquery.Selection
	for {
		if retryBypassNum > MaxRetryLoginBypassCount {
			return nil, errors.New("登录过验证重试次数太多: " + rawUrl)
		}

		root, err = s.downloader.Download(rawUrl)
		if err != nil {
			return nil, err
		}

		if s.shouldLoginFunc(root) {
			// 需要登录过验证
			func() {
				s.loginMux.Lock()
				log.Debugf("未获取到页面信息，尝试登录验证刷新cookies，retryNum(%d): %s", retryBypassNum, rawUrl)
				defer func() {
					s.loginMux.Unlock()
					retryBypassNum++

					if r := recover(); r != nil {
						log.Errorf("刷新cookies异常: %s - %v", rawUrl, r)
					}
				}()

				s.StartRodHandleLogin(rawUrl)
			}()

			// 绕过验证后，继续下一次
			continue
		}

		// 不需要验证，直接跳出
		break
	}

	return root, nil
}

func (s *LoginBypassDownloader) StartRodHandleLogin(rawUrl string) {
	page, closeFunc := rod_browser.NewBrowser()
	defer func() {
		time.AfterFunc(10*time.Second, func() {
			closeFunc()
		})
	}()

	s.HandleLoginRefreshCookies(page, rawUrl)
}

func (s *LoginBypassDownloader) HandleLoginRefreshCookies(page *rod.Page, rawUrl string) {
	page.MustNavigate(rawUrl)
	// 等待页面加载
	log.Debugf("等待页面 %s 加载...", rawUrl)
	err := page.WaitLoad()
	if err != nil {
		panic(err)
	}
	log.Debugf("页面 %s 加载完毕...", rawUrl)

	defer func(page *rod.Page) {
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
	}(page)

	loginErrNum := 0
	for {
		if loginErrNum > MaxRetryLoginErrorCount {
			break
		}

		log.Debugf("检查登录验证，retryNum(%d)...", loginErrNum)
		html, err := page.HTML()
		if err != nil {
			panic(err)
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(html))
		if err != nil {
			panic(err)
		}

		if s.shouldLoginFunc(doc.Selection) {
			// 执行绕过验证逻辑
			if err = s.handleLoginFunc(page); err != nil {
				log.Debugf("执行绕过验证 %s 异常：%s", rawUrl, err.Error())
				panic(err)
			}

			loginErrNum++

			time.Sleep(10 * time.Second)

			// 刷新页面
			err = page.Reload()
			if err != nil {
				panic(err)
			}
			err = page.WaitLoad()
			if err != nil {
				panic(err)
			}
		}
	}

	log.Debugf("页面 %s 加载完毕...", rawUrl)
	time.Sleep(3 * time.Second) // 观察手动登录
}
