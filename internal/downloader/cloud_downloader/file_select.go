package cloud_downloader

import (
	"sort"

	"github.com/nekoimi/get-magnet/internal/pkg/files"
	log "github.com/sirupsen/logrus"
)

// MinVideoSize 文件最小大小：100M
const MinVideoSize = 100_000_000

func selectBestCloudFiles(fs []cloudFile) ([]cloudFile, []cloudFile) {
	var allowFiles []cloudFile
	var notAllowFiles []cloudFile
	for _, f := range fs {
		if f.IsDir {
			log.Debugf("网盘下载任务文件优选 - 跳过目录: %s", cloudFilePath(f))
			continue
		}

		filePath := cloudFileDisplayPath(f)
		if err := files.IsValidFileName(filePath); err != nil {
			log.Debugf("网盘下载任务文件优选 - 文件名不合法，删除: %s - %s", filePath, err.Error())
			notAllowFiles = append(notAllowFiles, f)
			continue
		}

		if isBestCloudFile(f) {
			allowFiles = append(allowFiles, f)
		} else {
			notAllowFiles = append(notAllowFiles, f)
		}
	}
	sort.SliceStable(allowFiles, func(i, j int) bool {
		return allowFiles[i].Size > allowFiles[j].Size
	})
	return allowFiles, notAllowFiles
}

func isBestCloudFile(f cloudFile) bool {
	filePath := cloudFileDisplayPath(f)
	return files.IsVideo(filePath) && f.Size > MinVideoSize
}

func cloudFilePath(f cloudFile) string {
	if f.Path != "" {
		return f.Path
	}
	if f.RelativePath != "" {
		return f.RelativePath
	}
	return f.Name
}

func cloudFileDisplayPath(f cloudFile) string {
	if f.RelativePath != "" {
		return f.RelativePath
	}
	return cloudFilePath(f)
}
