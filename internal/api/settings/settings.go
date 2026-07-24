package settings

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/downloader/cloud_downloader"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/siku2/arigo"
)

type TestResult struct {
	OK        bool      `json:"ok"`
	Message   string    `json:"message,omitempty"`
	LatencyMs int64     `json:"latency_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

func List(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		safe := *cfg
		safe.JwtSecret = mask(cfg.JwtSecret)
		if cfg.Aria2 != nil {
			aria2 := *cfg.Aria2
			aria2.Secret = mask(cfg.Aria2.Secret)
			safe.Aria2 = &aria2
		}
		if cfg.DB != nil {
			database := *cfg.DB
			database.Dsn = maskDSN(cfg.DB.Dsn)
			safe.DB = &database
		}
		respond.Ok(w, safe)
	}
}

func TestCloudDriver(cfg *config.Config) http.HandlerFunc {
	return testHandler(func(ctx context.Context) error {
		return cloud_downloader.CheckHealth(ctx, cfg.CloudDriver)
	})
}

func TestAria2(cfg *config.Config) http.HandlerFunc {
	return testHandler(func(_ context.Context) error { return CheckAria2(cfg.Aria2) })
}

func TestDrissionRod(cfg *config.Config) http.HandlerFunc {
	return testHandler(func(ctx context.Context) error { return CheckDrissionRod(ctx, cfg.Crawler) })
}

func testHandler(check func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		err := check(ctx)
		result := TestResult{OK: err == nil, LatencyMs: time.Since(started).Milliseconds(), CheckedAt: time.Now()}
		if err != nil {
			result.Message = err.Error()
		}
		respond.Ok(w, result)
	}
}

func CheckAria2(cfg *config.Aria2Config) error {
	if cfg == nil || strings.TrimSpace(cfg.JsonRpc) == "" {
		return fmt.Errorf("aria2.jsonrpc 未配置")
	}
	client, err := arigo.Dial(cfg.JsonRpc, cfg.Secret)
	if err != nil {
		return err
	}
	defer client.Close()
	_, err = client.GetVersion()
	return err
}

func CheckDrissionRod(ctx context.Context, cfg *config.CrawlerConfig) error {
	if cfg == nil || strings.TrimSpace(cfg.DrissionRodGrpcIp) == "" || cfg.DrissionRodGrpcPort <= 0 {
		return fmt.Errorf("DrissionRod gRPC 地址未配置")
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", cfg.DrissionRodGrpcIp, cfg.DrissionRodGrpcPort))
	if err != nil {
		return err
	}
	return conn.Close()
}

func mask(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

func maskDSN(value string) string {
	if value == "" {
		return ""
	}
	if at := strings.LastIndex(value, "@"); at >= 0 {
		return "****" + value[at:]
	}
	return mask(value)
}
