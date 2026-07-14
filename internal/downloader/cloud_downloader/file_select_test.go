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

func TestSelectBestCloudFilesUsesRelativePath(t *testing.T) {
	files := []cloudFile{
		{Path: "/cloud/save/movie-dir/movie.mp4", RelativePath: "movie-dir/movie.mp4", Size: MinVideoSize + 1},
	}

	allowFiles, delFiles := selectBestCloudFiles(files)
	if len(allowFiles) != 1 {
		t.Fatalf("expected 1 allowed file, got %d", len(allowFiles))
	}
	if len(delFiles) != 0 {
		t.Fatalf("expected no deleted files, got %d", len(delFiles))
	}
	if got := cloudFilePath(allowFiles[0]); got != "/cloud/save/movie-dir/movie.mp4" {
		t.Fatalf("cloud file path = %q, want absolute provider path", got)
	}
	if got := cloudFileDisplayPath(allowFiles[0]); got != "movie-dir/movie.mp4" {
		t.Fatalf("cloud display path = %q, want relative path", got)
	}
}

func TestSelectBestCloudFilesSortsVideosBySize(t *testing.T) {
	files := []cloudFile{
		{Name: "small.mp4", Size: MinVideoSize + 1},
		{Name: "large.mp4", Size: MinVideoSize + 1000},
	}

	allowFiles, _ := selectBestCloudFiles(files)
	if len(allowFiles) != 2 {
		t.Fatalf("expected 2 allowed files, got %d", len(allowFiles))
	}
	if allowFiles[0].Name != "large.mp4" {
		t.Fatalf("first allowed file = %q, want largest file", allowFiles[0].Name)
	}
}
