package cloud_downloader

import "testing"

func TestSelectBestCloudFiles(t *testing.T) {
	files := []cloudFile{
		{Name: "movie.mp4", Size: MinVideoSize + 1},
		{Name: "sample.mp4", Size: MinVideoSize - 1},
		{Name: "movie.srt", Size: 1000},
		{Name: "source.torrent", Size: 1000},
		{Name: "   ", Size: MinVideoSize + 1},
	}

	allowFiles, delFiles := selectBestCloudFiles(files)
	if len(allowFiles) != 2 {
		t.Fatalf("expected 2 allowed files, got %d", len(allowFiles))
	}
	if allowFiles[0].Name != "movie.mp4" || allowFiles[1].Name != "source.torrent" {
		t.Fatalf("unexpected allowed files: %#v", allowFiles)
	}
	if len(delFiles) != 3 {
		t.Fatalf("expected 3 deleted files, got %d", len(delFiles))
	}
}
