package tests

import (
	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/crawler/task"
	"github.com/nekoimi/get-magnet/internal/server"
	"os"
	"testing"
	"time"
)

func Test_Run(t *testing.T) {
	os.Setenv("ROD_HEADLESS", "false")
	os.Setenv("ROD_DATA_DIR", "C:\\Users\\nekoimi\\Downloads\\rod-data")
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:12080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:12080")
	os.Setenv("DB_DSN", "postgres://devtest:devtest@10.1.1.100:5432/get_magnet_dev?sslmode=disable")
	os.Setenv("OCR_BIN_PATH", "C:\\Users\\nekoimi\\Downloads\\x86_64-pc-windows-msvc-inline\\ddddocr.exe")
	os.Setenv("JAVDB_USERNAME", "111111111111")
	os.Setenv("JAVDB_PASSWORD", "111111111111")
	// DB_DSN

	cfg := config.Default()

	s := server.Default(cfg)

	time.AfterFunc(30*time.Second, func() {
		t.Log("提交测试任务...")
		bus.Event().Publish(bus.SubmitTask.String(), task.NewTask("https://javdb.com/login",
			task.WithHandle(javdb.TaskSeeder()),
			task.WithDownloader(javdb.GetBypassDownloader()),
		))
	})

	s.Run()

}
