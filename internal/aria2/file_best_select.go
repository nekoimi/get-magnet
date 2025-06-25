package aria2

import (
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"github.com/siku2/arigo"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LowSpeedNum 低速下载区间测速检查数量
const LowSpeedNum = 30

// LowSpeedTimeout 低速下载多长时间超时
const LowSpeedTimeout = 35 * time.Minute

// LowSpeedInterval 低速下载检查周期
const LowSpeedInterval = 1 * time.Minute

// LowSpeedCleanupInterval 低速下载记录清除检查周期
const LowSpeedCleanupInterval = 5 * time.Minute

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

func (a *Aria2) downloadFileBestSelect(files []arigo.File) (selectIndex string, ok bool) {
	if len(files) <= 1 {
		// 只有一个文件，不做处理
		return "", false
	}

	needChangeOps := false
	for _, f := range files {
		// if selected non best, need re-change options
		if f.Selected && !isBestFile(f) {
			needChangeOps = true
			break
		}
	}

	if needChangeOps {
		allowFiles := bestSelectFile(files)
		if len(allowFiles) == 0 {
			// 不做处理
			return "", false
		}

		var builder strings.Builder
		for _, a := range allowFiles {
			builder.WriteString(strconv.Itoa(a.Index))
			builder.WriteString(",")
		}
		return builder.String(), true
	}

	return "", false
}

func display(status arigo.Status) string {
	if len(status.Files) == 0 {
		return "unknow"
	}

	var maxFile arigo.File
	var maxSize uint

	statusFiles := status.Files
	for i := range statusFiles {
		length := statusFiles[i].Length
		if length > maxSize {
			maxSize = length
			maxFile = statusFiles[i]
		}
	}

	name := filepath.Base(maxFile.Path)
	if name == "." {
		return "GID#" + status.GID
	}
	return name
}
