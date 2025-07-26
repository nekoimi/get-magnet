package rod_browser

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/config"
	"os"
	"testing"
	"time"
)

func TestNewTabPage(t *testing.T) {
	os.Setenv("ROD_HEADLESS", "false")

	config.Default()

	b := NewRodBrowser(context.TODO(), nil)

	<-time.After(5 * time.Second)

	go func() {
		page1, f := b.NewTabPage()
		defer func() {
			time.AfterFunc(1*time.Second, func() {
				f()
				t.Log("关闭页面1")
			})
		}()
		page1.MustNavigate("https://www.baidu.com")
		page1.MustWaitLoad()
	}()

	go func() {
		page2, f := b.NewTabPage()
		defer func() {
			time.AfterFunc(5*time.Second, func() {
				f()
				t.Log("关闭页面2")
			})
		}()
		page2.MustNavigate("https://www.baidu.com")
		page2.MustWaitLoad()
	}()

	go func() {
		page3, f := b.NewTabPage()
		defer func() {
			time.AfterFunc(10*time.Second, func() {
				f()
				t.Log("关闭页面3")
			})
		}()
		page3.MustNavigate("https://www.baidu.com")
		page3.MustWaitLoad()
	}()

	time.AfterFunc(30*time.Second, func() {
		b.Close()
	})

	<-time.After(60 * time.Second)
}
