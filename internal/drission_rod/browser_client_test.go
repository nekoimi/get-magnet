package drission_rod

import (
	"context"
	"testing"
	"time"

	"github.com/nekoimi/scrapio/internal/bean"
	"github.com/nekoimi/scrapio/internal/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBrowserLifecycleDoesNotBlockHTTPWorkflows(t *testing.T) {
	ctx := bean.ContextWithDefaultRegistry(context.Background())
	bean.MustRegisterPtr(ctx, &config.Config{Crawler: &config.CrawlerConfig{DrissionRodGrpcIp: "127.0.0.1", DrissionRodGrpcPort: 1}})
	browser := NewDrissionRod()
	started := time.Now()
	if err := browser.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer browser.Stop(ctx)
	if time.Since(started) > time.Second {
		t.Fatal("browser startup blocked on an unavailable service")
	}
}

func TestBrowserJobValidation(t *testing.T) {
	_, err := NewDrissionRod().Execute(context.Background(), BrowserJob{})
	if err == nil {
		t.Fatal("expected invalid job error")
	}
	jobErr, ok := err.(*BrowserError)
	if !ok || jobErr.Code != "INVALID_JOB" || jobErr.Retryable {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestBrowserUnavailableAndRetryClassification(t *testing.T) {
	_, err := NewDrissionRod().Execute(context.Background(), BrowserJob{URL: "https://example.test", Timeout: time.Second})
	jobErr, ok := err.(*BrowserError)
	if !ok || jobErr.Code != "BROWSER_UNAVAILABLE" || !jobErr.Retryable {
		t.Fatalf("unexpected unavailable error: %#v", err)
	}
	if !retryableRPC(status.Error(codes.Unavailable, "down")) || retryableRPC(status.Error(codes.InvalidArgument, "bad")) {
		t.Fatal("unexpected retry classification")
	}
}
