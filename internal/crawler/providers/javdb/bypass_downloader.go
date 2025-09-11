package javdb

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/crawler/download"
	"github.com/nekoimi/get-magnet/internal/ocr"
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	log "github.com/sirupsen/logrus"
	"time"
)

func newBypassDownloader(cfg *config.JavDBConfig, browser *rod_browser.Browser, cloudflarePassApi string) download.Downloader {
	clickBypassDownloader := download.NewClickBypassDownloader(
		browser,
		cloudflarePassApi,
		func(root *goquery.Selection) bool {
			return root.Find("body > div.modal.is-active.over18-modal").Size() > 0
		},
		func(page *rod.Page) error {
			btn := page.MustElementByJS(`() => document.querySelector("body > div.modal.is-active.over18-modal > div.modal-card > footer > a.button.is-success.is-large")`)
			text, err := btn.Text()
			if err != nil {
				return err
			}
			log.Debugf("点击访问按钮: %s", text)
			btn.MustClick()
			return nil
		},
	)

	return download.NewLoginBypassDownloader(
		browser,
		clickBypassDownloader,
		func(root *goquery.Selection) bool {
			return root.Find("#password").Size() > 0 &&
				root.Find("#remember").Size() > 0
		},
		func(page *rod.Page) error {
			log.Debugln("执行自定义Login...")
			// captcha image
			captchaImage, err := page.Element("img.rucaptcha-image")
			if err != nil {
				return err
			}
			log.Debugf("图片验证码：%s", captchaImage.MustHTML())
			_, tempPath, cleanup, err := files.TempFile("png")
			if err != nil {
				return err
			}
			defer cleanup()

			imaeBytes := captchaImage.MustScreenshot(tempPath)
			log.Debugf("图片验证码文件：%s", tempPath)
			code, err := ocr.Call(imaeBytes)
			if err != nil {
				return err
			}
			log.Debugf("OCR识别结果：%s", code)
			time.Sleep(3 * time.Second)

			// username
			usernameInput, err := page.Element("#email")
			if err != nil {
				return err
			}
			usernameInput.MustInput(cfg.Username)
			time.Sleep(1 * time.Second)

			// password
			passwordInput, err := page.Element("#password")
			if err != nil {
				return err
			}
			passwordInput.MustInput(cfg.Password)
			time.Sleep(1 * time.Second)

			// captcha code
			captchaCodeInput, err := page.Element("input.rucaptcha-input")
			if err != nil {
				return err
			}
			captchaCodeInput.MustInput(code)
			time.Sleep(1 * time.Second)

			// remember
			rememberInput, err := page.Element("#remember")
			if err != nil {
				return err
			}
			rememberInput.MustClick()
			time.Sleep(1 * time.Second)

			// Login
			submitBtn, err := page.Element(`input[type="submit"].button.is-link`)
			if err != nil {
				return err
			}
			submitBtn.MustClick()

			err = page.WaitLoad()

			log.Debugf("submit login...")

			return err
		},
	)
}
