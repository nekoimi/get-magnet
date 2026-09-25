package settings

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/nekoimi/scrapio/internal/config"
	"github.com/nekoimi/scrapio/internal/pkg/respond"
)

type TestResult struct {
	OK        bool      `json:"ok"`
	Message   string    `json:"message,omitempty"`
	LatencyMs int64     `json:"latency_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

func List(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respond.Ok(w, cfg.Redacted())
	}
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

func CheckDrissionRod(ctx context.Context, cfg *config.CrawlerConfig) error {
	if cfg == nil || cfg.DrissionRodGrpcIp == "" || cfg.DrissionRodGrpcPort <= 0 {
		return fmt.Errorf("DrissionRod gRPC 地址未配置")
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", cfg.DrissionRodGrpcIp, cfg.DrissionRodGrpcPort))
	if err != nil {
		return err
	}
	return conn.Close()
}
