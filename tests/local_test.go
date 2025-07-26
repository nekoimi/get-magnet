package tests

import (
	"context"
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/logger"
	"github.com/nekoimi/get-magnet/internal/server"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
	"time"
)

func Test_Run(t *testing.T) {
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")

	os.Setenv("PORT", "11234")
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("LOG_DIR", "logs")
	os.Setenv("JWT_SECRET", "xxxxxxx")
	os.Setenv("BROWSER_BIN", "")
	os.Setenv("BROWSER_HEADLESS", "false")
	os.Setenv("BROWSER_DATA_DIR", "C:\\Users\\nekoimi\\Downloads\\rod-data")
	os.Setenv("ARIA2_JSONRPC", "http://127.0.0.1:6800/jsonrpc")
	os.Setenv("ARIA2_SECRET", "123456")
	os.Setenv("ARIA2_MOVE_TO_JAVDB_DIR", "/tmp")
	os.Setenv("CRAWLER_EXEC_ON_STARTUP", "false")
	os.Setenv("CRAWLER_WORKER_NUM", "8")
	os.Setenv("CRAWLER_OCR_BIN", "C:\\Users\\nekoimi\\Downloads\\x86_64-pc-windows-msvc-inline\\ddddocr.exe")
	os.Setenv("JAVDB_USERNAME", "111111111111")
	os.Setenv("JAVDB_PASSWORD", "222222222222")
	os.Setenv("DB_DNS", "postgres://devtest:devtest@10.1.1.100:5432/get_magnet_dev?sslmode=disable")

	cfg := config.Load()
	logger.Initialize(cfg.LogLevel, cfg.LogDir)
	log.Infof("配置信息：\n%s", cfg)

	ctx := context.Background()

	s := server.NewServer(ctx, cfg)

	time.AfterFunc(30*time.Second, func() {
		t.Log("提交测试任务...")
		bus.Event().Publish(bus.SubmitJavDB.Topic(), "https://javdb.com/login")
	})

	s.Start(ctx)

}
