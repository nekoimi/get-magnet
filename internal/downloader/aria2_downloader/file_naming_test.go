package aria2_downloader

import "testing"

func TestBuildNormalizedVideoFilename(t *testing.T) {
	t.Run("保留可靠版本后缀", func(t *testing.T) {
		name, ok := buildNormalizedVideoFilename("DASS-891", "xxxxx.com@DASS-891-C.mp4")
		if !ok {
			t.Fatal("expected filename to be normalized")
		}
		if name != "DASS-891-C.mp4" {
			t.Fatalf("unexpected filename: %s", name)
		}
	})

	t.Run("无法可靠识别后缀时仅保留番号", func(t *testing.T) {
		name, ok := buildNormalizedVideoFilename("DASS-891", "xxxxx.com@DASS-891-C-1080p.mp4")
		if !ok {
			t.Fatal("expected filename to be normalized")
		}
		if name != "DASS-891.mp4" {
			t.Fatalf("unexpected filename: %s", name)
		}
	})

	t.Run("非视频文件不重命名", func(t *testing.T) {
		_, ok := buildNormalizedVideoFilename("DASS-891", "xxxxx.com@DASS-891-C.srt")
		if ok {
			t.Fatal("expected subtitle file to keep original name")
		}
	})
}
