package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/plugin"
)

func TestAria2Handle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":"gid-1"}`))
	}))
	defer server.Close()
	handler := NewAria2(config.Aria2Config{JsonRpc: server.URL})
	output, externalID, err := handler.Handle(context.Background(), plugin.Task{ResourceID: 1, Input: map[string]any{"url": "magnet:?xt=urn:btih:demo"}})
	if err != nil || externalID != "gid-1" || output == nil {
		t.Fatalf("unexpected aria2 result: %#v, %s, %v", output, externalID, err)
	}
}

func TestCloudDriverHandle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"cloud-1","status":"queued"}}`))
	}))
	defer server.Close()
	handler := NewCloudDriver(config.CloudDriverConfig{BaseURL: server.URL})
	output, externalID, err := handler.Handle(context.Background(), plugin.Task{ResourceID: 1, Input: map[string]any{"url": "https://example.test/file"}})
	if err != nil || externalID != "cloud-1" || output == nil {
		t.Fatalf("unexpected cloud result: %#v, %s, %v", output, externalID, err)
	}
}

func TestValidateDownloadURL(t *testing.T) {
	if err := validateDownloadURL("file:///tmp/a"); err == nil {
		t.Fatal("expected local file URL to be rejected")
	}
}
