package delivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nekoimi/scrapio/internal/config"
	"github.com/nekoimi/scrapio/internal/plugin"
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

func TestCloudDriverPollPending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/drivers/115/offline/tasks/cloud-1" {
			t.Fatalf("unexpected poll path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"cloud-1","status":"downloading","progress":42.5}}`))
	}))
	defer server.Close()
	handler := NewCloudDriver(config.CloudDriverConfig{BaseURL: server.URL})
	output, done, err := handler.Poll(context.Background(), plugin.Task{}, "cloud-1")
	if err != nil || done {
		t.Fatalf("unexpected pending result: %#v, %v, %v", output, done, err)
	}
	if output.(map[string]any)["status"] != "downloading" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestCloudDriverPollComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"cloud-2","status":"completed","progress":100,"files":[{"file_id":"f1","name":"movie.mp4","size":123}]}}`))
	}))
	defer server.Close()
	handler := NewCloudDriver(config.CloudDriverConfig{BaseURL: server.URL})
	output, done, err := handler.Poll(context.Background(), plugin.Task{}, "cloud-2")
	if err != nil || !done || output == nil {
		t.Fatalf("unexpected completed result: %#v, %v, %v", output, done, err)
	}
}

func TestCloudDriverPollFailureIsPermanent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"cloud-3","status":"failed","error_code":"NO_SPACE","error_message":"quota exceeded"}}`))
	}))
	defer server.Close()
	handler := NewCloudDriver(config.CloudDriverConfig{BaseURL: server.URL})
	_, done, err := handler.Poll(context.Background(), plugin.Task{}, "cloud-3")
	var permanent *plugin.PermanentError
	if !done || !errors.As(err, &permanent) || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("expected permanent cloud failure, done=%v err=%v", done, err)
	}
}

func TestValidateDownloadURL(t *testing.T) {
	if err := validateDownloadURL("file:///tmp/a"); err == nil {
		t.Fatal("expected local file URL to be rejected")
	}
}
