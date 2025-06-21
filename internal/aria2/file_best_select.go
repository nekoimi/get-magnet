package aria2

import (
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"strings"
	"time"
)

// LowSpeedTimeout 低速下载多长时间超时
const LowSpeedTimeout = 25 * time.Minute

// LowSpeedThreshold 速度小于100KB，speed 单位是 Bytes/s
const LowSpeedThreshold = 102400

// MinVideoSize 文件最小大小：100M
const MinVideoSize = 100_000_000

func bestSelectFile(files []arigo.File) []arigo.File {
	var allowFiles []arigo.File
	for _, f := range files {
		if isBestFile(f) {
			allowFiles = append(allowFiles, f)
		}
	}
	return allowFiles
}

func isBestFile(f arigo.File) bool {
	return (files.IsVideo(f.Path) && f.Length > MinVideoSize) || isTorrentFile(f.Path)
}

func isTorrentFile(filename string) bool {
	return strings.HasSuffix(filename, ".torrent")
}
