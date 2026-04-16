package aria2_downloader

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
)

func handleDownloadComplete(status arigo.Status, origin string, moveToDir string) {
	followedBys := status.FollowedBy
	if len(followedBys) >= 1 {
		// 不是最终的下载任务，尝试更新数据表中关联的id
		followedBy := followedBys[0]
		if err := magnet_repo.UpdateFollowedBy(status.GID, followedBy); err != nil {
			log.Errorf("任务[%s]下载完成 - 更新 FollowedBy: [%s -> %s]，异常：%s", friendly(status), status.GID, followedBy, err.Error())
			return
		}
		log.Debugf("任务[%s]下载完成 - 更新 FollowedBy: [%s -> %s]", friendly(status), status.GID, followedBy)
		return
	}

	log.Debugf("bt任务下载完成 [origin: %s, moveToDir: %s] - FollowedBy: %s - %s", origin, moveToDir, status.GID, friendly(status))
	m, exists := magnet_repo.GetByFollowed(status.GID)
	if !exists {
		// 相关任务下载记录不存在，等待补偿任务兜底
		log.Warnf("bt相关任务下载记录不存在，忽略本次后处理：%s", friendly(status))
		return
	}

	if strings.ToUpper(m.Origin) != strings.ToUpper(origin) {
		// 不是当前来源，ignore
		return
	}

	if m.PostProcessDone {
		log.Debugf("任务[%s]下载完成后处理已执行，忽略重复处理", friendly(status))
		return
	}

	allowFiles, delFiles := extrBestFile(status.Files)
	for _, delFile := range delFiles {
		files.Delete(delFile.Path)
	}

	var moveErrs []error
	if moveToDir != "" {
		for _, file := range allowFiles {
			if err := moveSingleFile(status, file, m, moveToDir); err != nil {
				moveErrs = append(moveErrs, err)
			}
		}
	}

	if len(moveErrs) > 0 {
		log.Errorf("任务[%s]下载完成 - 后处理存在失败，等待补偿重试: %v", friendly(status), moveErrs)
		return
	}

	if err := magnet_repo.MarkPostProcessDone(m.Id); err != nil {
		log.Errorf("任务[%s]下载完成 - 标记后处理完成失败：%s", friendly(status), err.Error())
	}
}

// moveSingleFile 移动单个下载文件
func moveSingleFile(status arigo.Status, file arigo.File, magnet *table.Magnets, moveToDir string) error {
	sourcePath := file.Path
	sourceFile := filepath.Base(sourcePath)

	// 获取女演员名称
	actress := getActressName(magnet.Actress0)

	// 获取截断后的标题
	title := files.TruncateFilename(magnet.Title, files.MaxFileNameLength)

	targetFile := sourceFile
	if normalizedFile, ok := buildNormalizedVideoFilename(magnet.Number, sourceFile); ok {
		targetFile = normalizedFile
	}

	// 构建目标路径
	targetPath := buildTargetPath(moveToDir, actress, magnet.CreatedAt, title, targetFile)

	err := files.MoveOnce(sourcePath, targetPath)
	if err != nil {
		log.Errorf("[JavDB] bt任务下载完成 - 移动文件: %s -> %s，异常：%s", sourcePath, targetPath, err.Error())
		return fmt.Errorf("move %s -> %s: %w", sourcePath, targetPath, err)
	}
	log.Debugf("[JavDB] bt任务下载完成 - 移动文件：%s -> %s", sourcePath, targetPath)
	return nil
}

// getActressName 获取女演员名称，默认返回 "0未知"
func getActressName(actress0 string) string {
	if len(actress0) == 0 {
		return "0未知"
	}
	parts := strings.Split(actress0, ",")
	return parts[0]
}

// buildTargetPath 构建目标文件路径
// target: {moveToDir}/{女演员}/2025-07-22/{标题}/SONE-566-C.mp4
func buildTargetPath(moveToDir, actress string, createdAt time.Time, title, sourceFile string) string {
	targetPrefix := filepath.Join(moveToDir, actress, createdAt.Format("2006-01-02"))
	return filepath.Join(targetPrefix, title, sourceFile)
}
