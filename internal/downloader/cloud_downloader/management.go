package cloud_downloader

import (
	"context"
	"fmt"
	"strings"

	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
)

type CloudTask struct {
	TaskID         string      `json:"task_id"`
	ProviderTaskID string      `json:"provider_task_id,omitempty"`
	Status         string      `json:"status"`
	Name           string      `json:"name,omitempty"`
	Progress       float64     `json:"progress,omitempty"`
	SavePath       string      `json:"save_path,omitempty"`
	SaveDir        *CloudFile  `json:"save_dir,omitempty"`
	ErrorCode      string      `json:"error_code,omitempty"`
	ErrorMessage   string      `json:"error_message,omitempty"`
	Files          []CloudFile `json:"files"`
	Warnings       []string    `json:"warnings,omitempty"`
}

type CloudFile struct {
	ID           string         `json:"id,omitempty"`
	FileID       string         `json:"file_id,omitempty"`
	ParentID     string         `json:"parent_id,omitempty"`
	Name         string         `json:"name,omitempty"`
	Path         string         `json:"path,omitempty"`
	RelativePath string         `json:"relative_path,omitempty"`
	IsDir        bool           `json:"is_dir,omitempty"`
	Size         int64          `json:"size,omitempty"`
	Extra        map[string]any `json:"extra,omitempty"`
}

type RebuildSTRMResult struct {
	MagnetID int64    `json:"magnet_id"`
	TaskID   string   `json:"task_id,omitempty"`
	Paths    []string `json:"paths"`
}

func CheckHealth(ctx context.Context, cfg *config.CloudDriverConfig) error {
	return newCloudClient(normalizeCloudConfig(cfg)).health(ctx)
}

func GetTask(ctx context.Context, cfg *config.CloudDriverConfig, taskID string) (CloudTask, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return CloudTask{}, fmt.Errorf("taskID 不能为空")
	}
	task, err := newCloudClient(normalizeCloudConfig(cfg)).getOfflineTask(ctx, taskID)
	if err != nil {
		return CloudTask{}, err
	}
	return exportTask(task), nil
}

func RebuildSTRM(ctx context.Context, appCfg *config.AppConfig, cloudCfg *config.CloudDriverConfig, strmCfg *config.STRMConfig, magnet *table.Magnets) (RebuildSTRMResult, error) {
	if magnet == nil || magnet.Id <= 0 {
		return RebuildSTRMResult{}, fmt.Errorf("资源不存在")
	}
	taskID := strings.TrimSpace(magnet.FollowedBy)
	if taskID == "" || taskID == "unknow" {
		return RebuildSTRMResult{}, fmt.Errorf("资源未关联下载任务")
	}
	task, err := newCloudClient(normalizeCloudConfig(cloudCfg)).getOfflineTask(ctx, taskID)
	if err != nil {
		return RebuildSTRMResult{}, err
	}
	if strings.ToLower(task.Status) != "completed" {
		return RebuildSTRMResult{}, fmt.Errorf("下载任务尚未完成: %s", task.Status)
	}
	allowFiles, _ := selectBestCloudFiles(task.Files)
	if len(allowFiles) == 0 {
		return RebuildSTRMResult{}, fmt.Errorf("没有可生成strm的目标视频文件")
	}
	d := &CloudDownloader{
		appCfg:  normalizeAppConfig(appCfg),
		cfg:     normalizeCloudConfig(cloudCfg),
		strmCfg: normalizeSTRMConfig(strmCfg),
		client:  newCloudClient(normalizeCloudConfig(cloudCfg)),
	}
	paths, err := d.writeSTRMFiles(magnet, allowFiles)
	if err != nil {
		return RebuildSTRMResult{}, err
	}
	selectedFile := allowFiles[0]
	selectedPath := ""
	if len(paths) > 0 {
		selectedPath = paths[0]
	}
	if err := magnet_repo.MarkPostProcessDoneWithPlayInfo(magnet.Id, selectedFile.identity(), cloudFilePath(selectedFile), selectedFile.Size, selectedPath); err != nil {
		return RebuildSTRMResult{}, err
	}
	return RebuildSTRMResult{
		MagnetID: magnet.Id,
		TaskID:   taskID,
		Paths:    paths,
	}, nil
}

func normalizeAppConfig(cfg *config.AppConfig) *config.AppConfig {
	if cfg == nil {
		return &config.AppConfig{}
	}
	return cfg
}

func normalizeCloudConfig(cfg *config.CloudDriverConfig) *config.CloudDriverConfig {
	if cfg == nil {
		return &config.CloudDriverConfig{}
	}
	return cfg
}

func normalizeSTRMConfig(cfg *config.STRMConfig) *config.STRMConfig {
	if cfg == nil {
		return &config.STRMConfig{}
	}
	return cfg
}

func exportTask(task offlineTask) CloudTask {
	files := make([]CloudFile, 0, len(task.Files))
	for _, file := range task.Files {
		files = append(files, exportFile(file))
	}
	var saveDir *CloudFile
	if task.SaveDir != nil {
		exported := exportFile(*task.SaveDir)
		saveDir = &exported
	}
	return CloudTask{
		TaskID:         task.TaskID,
		ProviderTaskID: task.ProviderTaskID,
		Status:         task.Status,
		Name:           task.Name,
		Progress:       task.Progress,
		SavePath:       task.SavePath,
		SaveDir:        saveDir,
		ErrorCode:      task.ErrorCode,
		ErrorMessage:   task.ErrorMessage,
		Files:          files,
		Warnings:       task.Warnings,
	}
}

func exportFile(file cloudFile) CloudFile {
	return CloudFile{
		ID:           file.ID,
		FileID:       file.FileID,
		ParentID:     file.ParentID,
		Name:         file.Name,
		Path:         file.Path,
		RelativePath: file.RelativePath,
		IsDir:        file.IsDir,
		Size:         file.Size,
		Extra:        file.Extra,
	}
}
