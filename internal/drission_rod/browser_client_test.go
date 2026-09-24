package drission_rod

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
