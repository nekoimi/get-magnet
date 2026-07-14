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
	if len(allowFiles) != 1 {
		t.Fatalf("expected 1 allowed file, got %d", len(allowFiles))
	}
	if allowFiles[0].Name != "movie.mp4" {
		t.Fatalf("unexpected allowed files: %#v", allowFiles)
	}
	if len(delFiles) != 4 {
		t.Fatalf("expected 4 deleted files, got %d", len(delFiles))
	}
}

func TestSelectBestCloudFilesSkipsDirectory(t *testing.T) {
	files := []cloudFile{
		{Name: "movie-dir", Path: "/movie-dir", IsDir: true},
		{Name: "movie.mp4", Path: "/movie-dir/movie.mp4", Size: MinVideoSize + 1},
	}

	allowFiles, delFiles := selectBestCloudFiles(files)
	if len(allowFiles) != 1 {
		t.Fatalf("expected 1 allowed file, got %d", len(allowFiles))
	}
	if allowFiles[0].Name != "movie.mp4" {
		t.Fatalf("unexpected allowed files: %#v", allowFiles)
	}
	if len(delFiles) != 0 {
		t.Fatalf("expected directory to be skipped instead of deleted, got %d deleted files", len(delFiles))
	}
}
