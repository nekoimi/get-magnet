package cloud_downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/config"
)

type cloudClient struct {
	cfg        *config.CloudDriverConfig
	httpClient *http.Client
}

func newCloudClient(cfg *config.CloudDriverConfig) *cloudClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	return &cloudClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

func (c *cloudClient) health(ctx context.Context) error {
	var data map[string]any
	return c.do(ctx, http.MethodGet, "/health", nil, &data)
}

func (c *cloudClient) addOfflineTask(ctx context.Context, req addOfflineTaskRequest) (addOfflineTaskResponse, error) {
	var data addOfflineTaskResponse
	err := c.do(ctx, http.MethodPost, c.driverPath("/offline/add"), req, &data)
	if err != nil {
		return addOfflineTaskResponse{}, err
	}
	if data.TaskID == "" {
		return addOfflineTaskResponse{}, fmt.Errorf("网盘离线任务提交成功但返回 task_id 为空")
	}
	return data, nil
}

func (c *cloudClient) getOfflineTask(ctx context.Context, taskID string) (offlineTask, error) {
	var data offlineTask
	err := c.do(ctx, http.MethodGet, c.driverPath("/offline/tasks/"+url.PathEscape(taskID)), nil, &data)
	if err != nil {
		return offlineTask{}, err
	}
	if data.TaskID == "" {
		data.TaskID = taskID
	}
	return data, nil
}

func (c *cloudClient) removeFile(ctx context.Context, file cloudFile) error {
	req := removeCloudFileRequest{
		FileID: file.FileID,
		Path:   file.Path,
	}
	if req.FileID == "" && req.Path == "" {
		req.Path = file.Name
	}
	if req.FileID == "" && req.Path == "" {
		return fmt.Errorf("网盘文件缺少 file_id/path/name，无法删除")
	}
	return c.do(ctx, http.MethodDelete, c.driverPath("/fs/remove"), req, nil)
}

func (c *cloudClient) do(ctx context.Context, method string, path string, body any, data any) error {
	if c.cfg.BaseURL == "" {
		return fmt.Errorf("cloud_driver.base_url 未配置")
	}

	var requestBody *bytes.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化网盘中间服务请求异常: %w", err)
		}
		requestBody = bytes.NewReader(bs)
	} else {
		requestBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.url(path), requestBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.ProfileID != "" {
		req.Header.Set("X-Profile-ID", c.cfg.ProfileID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求网盘中间服务异常: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("网盘中间服务返回异常状态码: %d", resp.StatusCode)
	}

	wrapped := cloudResponse[json.RawMessage]{}
	if err = json.NewDecoder(resp.Body).Decode(&wrapped); err != nil {
		return fmt.Errorf("解析网盘中间服务响应异常: %w", err)
	}
	if wrapped.Code != 0 {
		msg := wrapped.message()
		if msg == "" {
			msg = fmt.Sprintf("%v", wrapped.Error)
		}
		return fmt.Errorf("网盘中间服务业务异常: code=%d, message=%s", wrapped.Code, msg)
	}
	if data == nil || len(wrapped.Data) == 0 {
		return nil
	}
	if err = json.Unmarshal(wrapped.Data, data); err != nil {
		return fmt.Errorf("解析网盘中间服务 data 异常: %w", err)
	}
	return nil
}

func (c *cloudClient) driverPath(path string) string {
	platform := c.cfg.Platform
	if platform == "" {
		platform = "115"
	}
	return "/drivers/" + url.PathEscape(platform) + path
}

func (c *cloudClient) url(path string) string {
	return strings.TrimRight(c.cfg.BaseURL, "/") + path
}
