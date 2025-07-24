package aria2

import (
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/db/repository"
	"github.com/nekoimi/get-magnet/internal/pkg/files"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
)

func downloadCompleteEventHandle(status arigo.Status, followedBys []string) {
	if len(followedBys) >= 1 {
		// 不是最终的下载任务，尝试更新数据表中关联的id
		followedBy := followedBys[0]
		log.Debugf("任务[%s]下载完成 - 尝试更新 FollowedBy: [%s -> %s]", display(status), status.GID, followedBy)
		if err := repository.UpdateFollowedBy(status.GID, followedBy); err != nil {
			log.Errorf("任务[%s]下载完成 - 尝试更新 FollowedBy: [%s -> %s]，异常：%s", display(status), status.GID, followedBy, err.Error())
			return
		}
	} else {
		log.Debugf("bt任务下载完成 - FollowedBy: %s - %s - %s", status.GID, status.FollowedBy, display(status))

		javDBDir := config.Get().BtMove.JavDBDir
		if javDBDir == "" {
			// 没有配置文件夹路径 ignore
			return
		}
		// 最终完成，需要移动位置
		if m, exists := repository.GetByFollowed(status.GID); exists {
			if m.Origin == "JavDB" {
				allowFiles, _ := bestSelectFile(status.Files)
				for _, file := range allowFiles {
					// root: {rootDir}
					// source: {rootDir}/JavDB/2025-07-22/SONE-566-C/SONE-566-C.mp4
					// target: {javDBDir}/{女演员}/2025-07-22/{标题}/SONE-566-C.mp4
					sourcePath := file.Path
					sourceFile := filepath.Base(sourcePath)

					actress := strings.Split(m.Actress0, ",")
					targetPrefix := filepath.Join(javDBDir, actress[0], m.CreatedAt.Format("2006-01-02"))
					targetPath := filepath.Join(targetPrefix, m.Title, sourceFile)
					if len(targetPath) >= MaxFileNameLength {
						// fix 需要缩短文件名称
						nameLen := len(sourceFile)
						prefixLen := len(targetPrefix)
						// 缩短标题
						maxLen := MaxFileNameLength - (nameLen + prefixLen + 10)
						targetPath = filepath.Join(targetPrefix, m.Title[:maxLen], sourceFile)
					}

					err := files.MoveOnce(sourcePath, targetPath)
					if err != nil {
						log.Errorf("[JavDB] bt任务下载完成 - 移动文件: %s -> %s，异常：%s", sourcePath, targetPath, err.Error())
						return
					}
					log.Debugf("[JavDB] bt任务下载完成 - 移动文件：%s -> %s", sourcePath, targetPath)
				}
			}
		}
	}
}
