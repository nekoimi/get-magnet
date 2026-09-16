package cloud_downloader

import (
	"context"
	"fmt"
	"strings"

	"github.com/nekoimi/get-magnet/internal/config"
)

func ResolveMediaURL(ctx context.Context, cfg *config.CloudDriverConfig, taskID string) (string, error) {
	return ResolveMediaURLWithFile(ctx, cfg, taskID, "", "")
}

func ResolveMediaURLWithFile(ctx context.Context, cfg *config.CloudDriverConfig, taskID, fileID, filePath string) (string, error) {
	if strings.TrimSpace(taskID) == "" {
		return "", fmt.Errorf("下载任务ID为空")
	}
	if cfg == nil {
		cfg = &config.CloudDriverConfig{}
	}

	client := newCloudClient(cfg)
	if strings.TrimSpace(fileID) != "" || strings.TrimSpace(filePath) != "" {
		return client.getMediaURL(ctx, cloudFile{
			FileID: strings.TrimSpace(fileID),
			Path:   strings.TrimSpace(filePath),
		})
	}

	task, err := client.getOfflineTask(ctx, taskID)
	if err != nil {
		return "", err
	}
	if strings.ToLower(task.Status) != "completed" {
		return "", fmt.Errorf("下载任务尚未完成: %s", task.Status)
	}

	allowFiles, _ := selectBestCloudFiles(task.Files)
	if len(allowFiles) == 0 {
		return "", fmt.Errorf("下载任务没有可播放视频文件")
	}
	return client.getMediaURL(ctx, allowFiles[0])
}
