package aria2

import (
	"errors"
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MinVideoSize 文件最小大小：100M
const MinVideoSize = 100_000_000

// MaxFileNameLength 最大文件名长度 255
const MaxFileNameLength = 255

func (a *Aria2) handleFileBestSelect(task arigo.Status) {
	log.Debugf("下载任务(%s)文件优选：%s", task.GID, display(task))
	if selectIndex, ok, delFiles := downloadFileBestSelect(task.Files); ok {
		if task.Status == arigo.StatusActive || task.Status == arigo.StatusWaiting {
			if err := a.client().ChangeOptions(task.GID, arigo.Options{
				SelectFile: selectIndex,
			}); err != nil {
				log.Errorf("下载任务(%s)文件优选异常：%s \n", display(task), err.Error())
			} else {
				log.Infof("下载任务(%s)文件优选：%s", display(task), selectIndex)
			}
		}

		for _, delFile := range delFiles {
			deleteUnBestFile(delFile.Path)
		}
	}
}

func bestSelectFile(files []arigo.File) ([]arigo.File, []arigo.File) {
	var allowFiles []arigo.File
	var notAllowFiles []arigo.File
	for _, f := range files {
		// 检查文件名称，超过限制就跳过该文件
		if err := isValidFileName(f.Path); err != nil {
			notAllowFiles = append(notAllowFiles, f)
			continue
		}

		if isBestFile(f) {
			allowFiles = append(allowFiles, f)
		} else {
			notAllowFiles = append(notAllowFiles, f)
		}
	}
	return allowFiles, notAllowFiles
}

func deleteUnBestFile(filepath string) {
	if exists, err := files.Exists(filepath); err != nil {
		log.Errorf(err.Error())
	} else if exists {
		if err = os.Remove(filepath); err != nil {
			log.Errorf("删除下载文件异常：%s", err.Error())
		}
		log.Debugf("删除不符合要求的下载文件：%s", filepath)
	}
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

func downloadFileBestSelect(files []arigo.File) (selectIndex string, ok bool, delFiles []arigo.File) {
	if len(files) <= 1 {
		// 只有一个文件，不做处理
		return "", false, delFiles
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
		allowFiles, notAllowFiles := bestSelectFile(files)
		if len(allowFiles) == 0 {
			// 不做处理
			return "", false, notAllowFiles
		}

		var builder strings.Builder
		for _, a := range allowFiles {
			builder.WriteString(strconv.Itoa(a.Index))
			builder.WriteString(",")
		}
		return builder.String(), true, notAllowFiles
	}

	return "", false, delFiles
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
