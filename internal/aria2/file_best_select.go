package aria2

import (
	"errors"
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// LowSpeedNum 低速下载区间测速检查数量
const LowSpeedNum = 15

// LowSpeedTimeout 低速下载多长时间超时
const LowSpeedTimeout = 20 * time.Minute

// LowSpeedInterval 低速下载检查周期
const LowSpeedInterval = 1 * time.Minute

// LowSpeedCleanupInterval 低速下载记录清除检查周期
const LowSpeedCleanupInterval = 5 * time.Minute

// LowSpeedThreshold 速度小于100KB，speed 单位是 Bytes/s
const LowSpeedThreshold = 102400

// MinVideoSize 文件最小大小：100M
const MinVideoSize = 100_000_000

// MaxFileNameLength 最大文件名长度 255
const MaxFileNameLength = 255

func (a *Aria2) handleFileBestSelect(task arigo.Status) {
	if selectIndex, ok := downloadFileBestSelect(task.Files); ok {
		if err := a.client().ChangeOptions(task.GID, arigo.Options{
			SelectFile: selectIndex,
		}); err != nil {
			log.Errorf("下载任务(%s)文件优选异常：%s \n", display(task), err.Error())
		} else {
			log.Infof("下载任务(%s)文件优选：%s", display(task), selectIndex)
		}
	}
}

func bestSelectFile(files []arigo.File) []arigo.File {
	var allowFiles []arigo.File
	for _, f := range files {
		// 检查文件名称，超过限制就跳过该文件
		if err := isValidFileName(f.Path); err != nil {
			continue
		}

		if isBestFile(f) {
			allowFiles = append(allowFiles, f)
		}
	}
	return allowFiles
}

// 检查文件名是否合法（长度与非法字符）
func isValidFileName(path string) error {
	base := filepath.Base(path)

	// 检查是否为空
	if strings.TrimSpace(base) == "" {
		return errors.New("文件名为空")
	}

	// 检查是否包含非法字符（可扩展）
	illegalChars := []string{"/", "\\", "\x00"} // 你可以根据需求增加
	for _, ch := range illegalChars {
		if strings.Contains(base, ch) {
			return errors.New("文件名包含非法字符: " + ch)
		}
	}

	// 检查文件名长度（字节数，非字符数）
	if len(base) > MaxFileNameLength {
		return errors.New("文件名过长（字节数超过 255）")
	}

	// 可选：检查字符数量（非必要）
	if utf8.RuneCountInString(base) == 0 {
		return errors.New("文件名无效")
	}

	return nil
}

func isBestFile(f arigo.File) bool {
	return (files.IsVideo(f.Path) && f.Length > MinVideoSize) || isTorrentFile(f.Path)
}

func isTorrentFile(filename string) bool {
	return strings.HasSuffix(filename, ".torrent")
}

func downloadFileBestSelect(files []arigo.File) (selectIndex string, ok bool) {
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
