package cloud_downloader

import (
	"net/url"
	"testing"

	"github.com/nekoimi/get-magnet/internal/config"
)

func TestBuildPlayURLIncludesFileID(t *testing.T) {
	got, err := buildPlayURL(&config.AppConfig{ExternalBaseURL: "http://example.test/"}, "abc-123", cloudFile{
		FileID: "file-1",
		Path:   "/cloud/save/movie.mp4",
	})
	if err != nil {
		t.Fatalf("buildPlayURL returned error: %v", err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse play url: %v", err)
	}
	if parsed.Path != "/api/play/ABC-123" {
		t.Fatalf("path = %q, want /api/play/ABC-123", parsed.Path)
	}
	if parsed.Query().Get("file_id") != "file-1" {
		t.Fatalf("file_id query = %q, want file-1", parsed.Query().Get("file_id"))
	}
	if parsed.Query().Get("path") != "" {
		t.Fatalf("path query = %q, want empty when file_id exists", parsed.Query().Get("path"))
	}
}

func TestBuildPlayURLFallsBackToPath(t *testing.T) {
	got, err := buildPlayURL(&config.AppConfig{ExternalBaseURL: "http://example.test"}, "abc-123", cloudFile{
		Path: "/cloud/save/movie.mp4",
	})
	if err != nil {
		t.Fatalf("buildPlayURL returned error: %v", err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse play url: %v", err)
	}
	if parsed.Query().Get("path") != "/cloud/save/movie.mp4" {
		t.Fatalf("path query = %q, want provider path", parsed.Query().Get("path"))
	}
}
