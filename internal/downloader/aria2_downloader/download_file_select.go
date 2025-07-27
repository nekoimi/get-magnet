package aria2_downloader

import (
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

// MinVideoSize 文件最小大小：100M
const MinVideoSize = 100_000_000

func (c *Client) handleFileBestSelect(status arigo.Status) {
	log.Debugf("下载任务文件优选：%s", friendly(status))
	selectIndex, ok, delFiles := selectDownloadFileBestIndex(status.Files)
	if ok {
		if status.Status == arigo.StatusActive || status.Status == arigo.StatusWaiting {
			if err := c.client().ChangeOptions(status.GID, arigo.Options{
				SelectFile: selectIndex,
			}); err != nil {
				log.Errorf("下载任务(%s)文件优选异常：%s", friendly(status), err.Error())
			} else {
				log.Infof("下载任务(%s)文件优选：%s", friendly(status), selectIndex)
			}
		}
	}

	for _, delFile := range delFiles {
		files.Delete(delFile.Path)
	}
}

func selectDownloadFileBestIndex(files []arigo.File) (selectIndex string, ok bool, delFiles []arigo.File) {
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
		allowFiles, notAllowFiles := extrBestFile(files)
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

func extrBestFile(fs []arigo.File) ([]arigo.File, []arigo.File) {
	var allowFiles []arigo.File
	var notAllowFiles []arigo.File
	for _, f := range fs {
		// 检查文件名称，超过限制就跳过该文件
		if err := files.IsValidFileName(f.Path); err != nil {
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

func isBestFile(f arigo.File) bool {
	return (files.IsVideo(f.Path) && f.Length > MinVideoSize) || files.IsTorrentFile(f.Path)
}
