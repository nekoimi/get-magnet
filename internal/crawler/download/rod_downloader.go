package download

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RodBrowserDownloader 浏览器下载
type RodBrowserDownloader struct {
	// 浏览器
	browser *rod_browser.Browser
	// cloudflare pass
	cloudflarePassApi string
}

type cloudflareRespBody struct {
	Solution struct {
		URL     string `json:"url,omitempty"`
		Status  int    `json:"status,omitempty"`
		Headers struct {
			Status                  string `json:"status,omitempty"`
			Date                    string `json:"date,omitempty"`
			Expires                 string `json:"expires,omitempty"`
			CacheControl            string `json:"cache-control,omitempty"`
			ContentType             string `json:"content-type,omitempty"`
			StrictTransportSecurity string `json:"strict-transport-security,omitempty"`
			P3P                     string `json:"p3p,omitempty"`
			ContentEncoding         string `json:"content-encoding,omitempty"`
			Server                  string `json:"server,omitempty"`
			ContentLength           string `json:"content-length,omitempty"`
			XXSSProtection          string `json:"x-xss-protection,omitempty"`
			XFrameOptions           string `json:"x-frame-options,omitempty"`
			SetCookie               string `json:"set-cookie,omitempty"`
		} `json:"headers,omitempty"`
		Response string `json:"response,omitempty"`
		Cookies  []struct {
			Name     string  `json:"name,omitempty"`
			Value    string  `json:"value,omitempty"`
			Domain   string  `json:"domain,omitempty"`
			Path     string  `json:"path,omitempty"`
			Expires  float64 `json:"expires,omitempty"`
			Size     int     `json:"size,omitempty"`
			HTTPOnly bool    `json:"httpOnly,omitempty"`
			Secure   bool    `json:"secure,omitempty"`
			Session  bool    `json:"session,omitempty"`
			SameSite string  `json:"sameSite,omitempty"`
		} `json:"cookies,omitempty"`
		UserAgent string `json:"userAgent,omitempty"`
	} `json:"solution,omitempty"`
	Status         string `json:"status,omitempty"`
	Message        string `json:"message,omitempty"`
	StartTimestamp int64  `json:"startTimestamp,omitempty"`
	EndTimestamp   int64  `json:"endTimestamp,omitempty"`
	Version        string `json:"version,omitempty"`
}

func NewRodBrowserDownloader(browser *rod_browser.Browser, cloudflarePassApi string) Downloader {
	return &RodBrowserDownloader{browser: browser, cloudflarePassApi: cloudflarePassApi}
}

func (s *RodBrowserDownloader) SetCookies(u *url.URL, cookies []*http.Cookie) {
}

func (s *RodBrowserDownloader) Download(rawUrl string) (selection *goquery.Selection, err error) {
	page, closeFunc := s.browser.NewTabPage()
	defer closeFunc(rawUrl)

	page.MustNavigate(rawUrl)
	// 等待页面加载
	log.Debugf("等待页面 %s 加载...", rawUrl)
	page.Timeout(10 * time.Second).MustWaitStable()

	html, err := page.HTML()
	if err != nil {
		return nil, err
	}

	log.Debugf("rod页面 %s 加载: %s", rawUrl, html)

	// 检查 challenges.cloudflare.com
	if false && strings.Contains(html, "challenges.cloudflare.com") && strings.Contains(html, "Security Verification") {
		log.Debugf("处理cloudflare rawUrl: %s", rawUrl)
		resp, respBody, err := s.cloudflare(rawUrl, page)
		if err != nil {
			return nil, err
		}

		if strings.ToLower(respBody.Status) != "ok" {
			return nil, errors.New("cloudflare error: " + resp)
		}

		// refresh cookies
		newCookies := make([]*proto.NetworkCookieParam, 0)
		cookies := respBody.Solution.Cookies
		for _, c := range cookies {
			newCookies = append(newCookies, &proto.NetworkCookieParam{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   c.Domain,
				Path:     c.Path,
				Secure:   c.Secure,
				HTTPOnly: c.HTTPOnly,
				Expires:  proto.TimeSinceEpoch(c.Expires),
			})
		}
		page.MustSetCookies(newCookies...)
		err = page.Reload()
		if err != nil {
			return nil, err
		}
		page.Timeout(10 * time.Second).MustWaitStable()
		// reset new html
		html, err = page.HTML()
		if err != nil {
			log.Errorf("cloudflare refresh page error: %s", err.Error())
			html = respBody.Solution.Response
		}
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(html))
	if err != nil {
		return nil, err
	}

	return doc.Selection, nil
}

func (s *RodBrowserDownloader) cloudflare(rawUrl string, page *rod.Page) (string, cloudflareRespBody, error) {
	var respBody cloudflareRespBody
	var cookies []map[string]interface{}
	rawCookies := page.MustCookies()
	for _, c := range rawCookies {
		cookies = append(cookies, map[string]interface{}{
			"name":  c.Name,
			"value": c.Value,
		})
	}
	// 构造请求体
	data := map[string]interface{}{
		"cmd":               "request.get",
		"url":               rawUrl,
		"maxTimeout":        60000,
		"cookies":           cookies,
		"returnOnlyCookies": false,
	}
	proxyEnv := httpproxy.FromEnvironment()
	if proxyEnv.HTTPProxy != "" {
		data["proxy"] = map[string]interface{}{
			"url": proxyEnv.HTTPProxy,
		}
	}

	// 转为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", respBody, err
	}

	log.Debugf("处理cloudflare 请求参数: %s", jsonData)

	// 构造请求
	req, err := http.NewRequest("POST", s.cloudflarePassApi, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", respBody, err
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", respBody, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", respBody, err
	}

	log.Debugf("处理cloudflare 响应: %s", string(body))

	err = json.Unmarshal(body, &respBody)
	if err != nil {
		return "", respBody, err
	}

	return string(body), respBody, nil
}
