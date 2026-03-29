package aria2_downloader

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
)

func handleDownloadCompleteDelFile(status arigo.Status) {
	_, delFiles := extrBestFile(status.Files)
	// 删除文件
	for _, delFileFile := range delFiles {
		files.Delete(delFileFile.Path)
	}
}

func handleDownloadCompleteMoveFile(status arigo.Status, origin string, moveToDir string) {
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
	if origin == "" || moveToDir == "" {
		// 没有配置 ignore
		return
	}

	// 最终完成，需要移动位置
	m, exists := magnet_repo.GetByFollowed(status.GID)
	if !exists {
		// 相关任务下载记录不存在 ignore
		log.Warnf("bt相关任务下载记录不存在，忽略文件移动：%s", friendly(status))
		return
	}

	if strings.ToUpper(m.Origin) != strings.ToUpper(origin) {
		// 不是当前来源，ignore
		return
	}

	allowFiles, _ := extrBestFile(status.Files)
	for _, file := range allowFiles {
		moveSingleFile(status, file, m, moveToDir)
	}
}

// moveSingleFile 移动单个下载文件
func moveSingleFile(status arigo.Status, file arigo.File, magnet *table.Magnets, moveToDir string) {
	sourcePath := file.Path
	sourceFile := filepath.Base(sourcePath)

	// 获取女演员名称
	actress := getActressName(magnet.Actress0)

	// 获取截断后的标题
	title := files.TruncateFilename(magnet.Title, files.MaxFileNameLength)

	// 构建目标路径
	targetPath := buildTargetPath(moveToDir, actress, magnet.CreatedAt, title, sourceFile)

	err := files.MoveOnce(sourcePath, targetPath)
	if err != nil {
		log.Errorf("[JavDB] bt任务下载完成 - 移动文件: %s -> %s，异常：%s", sourcePath, targetPath, err.Error())
		return
	}
	log.Debugf("[JavDB] bt任务下载完成 - 移动文件：%s -> %s", sourcePath, targetPath)
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
