package aria2

import (
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/pkg/files"
	"strings"
)

// MinVideoSize 文件最小大小：100M
const MinVideoSize = 100_000_000

func BestSelectFile(files []arigo.File) []arigo.File {
	var allowFiles []arigo.File
	for _, f := range files {
		if IsBestFile(f) {
			allowFiles = append(allowFiles, f)
		}
	}
	return allowFiles
}

func IsBestFile(f arigo.File) bool {
	return (files.IsVideo(f.Path) && f.Length > MinVideoSize) || IsTorrentFile(f.Path)
}

func IsTorrentFile(filename string) bool {
	return strings.HasSuffix(filename, ".torrent")
}
