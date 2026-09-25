package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/plugin"
)

const (
	CloudCode = "cloud-driver"
	Aria2Code = "aria2"
)

// RegisterBuiltins adds delivery adapters to the generic plugin registry. The
// adapters are inert when their legacy configuration is absent; collection
// remains fully usable without either external service.
func RegisterBuiltins(registry *plugin.Registry, cfg *config.Config) {
	if registry == nil || cfg == nil {
		return
	}
	if cfg.CloudDriver != nil && strings.TrimSpace(cfg.CloudDriver.BaseURL) != "" {
		_ = registry.Register(NewCloudDriver(*cfg.CloudDriver))
	}
	if cfg.Aria2 != nil && strings.TrimSpace(cfg.Aria2.JsonRpc) != "" {
		_ = registry.Register(NewAria2(*cfg.Aria2))
	}
}

type CloudDriver struct {
	cfg    config.CloudDriverConfig
	client *http.Client
}

func NewCloudDriver(cfg config.CloudDriverConfig) *CloudDriver {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &CloudDriver{cfg: cfg, client: &http.Client{Timeout: timeout}}
}
func (p *CloudDriver) Code() string           { return CloudCode }
func (p *CloudDriver) Capabilities() []string { return []string{"magnet.offline_add"} }
func (p *CloudDriver) Health(ctx context.Context) error {
	var value map[string]any
	return p.do(ctx, http.MethodGet, "/health", nil, &value)
}
func (p *CloudDriver) Handle(ctx context.Context, task plugin.Task) (any, string, error) {
	rawURL := inputString(task.Input, "url", "link", "optimal_link")
	if rawURL == "" {
		return nil, "", errors.New("cloud plugin requires input.url")
	}
	if err := validateDownloadURL(rawURL); err != nil {
		return nil, "", err
	}
	category := inputString(task.Input, "category", "origin")
	if category == "" {
		category = "resource"
	}
	savePath := inputString(task.Input, "save_path")
	request := map[string]any{"url": rawURL, "category": category, "client_task_id": fmt.Sprintf("resource-%d-%s", task.ResourceID, task.EventType), "metadata": map[string]string{"event_type": task.EventType}}
	if savePath != "" {
		request["save_path"] = savePath
	}
	var response struct {
		TaskID         string `json:"task_id"`
		ProviderTaskID string `json:"provider_task_id,omitempty"`
		Status         string `json:"status,omitempty"`
	}
	if err := p.do(ctx, http.MethodPost, p.driverPath("/offline/add"), request, &response); err != nil {
		return nil, "", err
	}
	if response.TaskID == "" {
		return nil, "", errors.New("cloud-driver returned empty task_id")
	}
	return map[string]any{"task_id": response.TaskID, "provider_task_id": response.ProviderTaskID, "status": response.Status}, response.TaskID, nil
}

type Aria2 struct {
	cfg    config.Aria2Config
	client *http.Client
}

func NewAria2(cfg config.Aria2Config) *Aria2 {
	return &Aria2{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}
func (p *Aria2) Code() string           { return Aria2Code }
func (p *Aria2) Capabilities() []string { return []string{"magnet.add_uri"} }
func (p *Aria2) Health(ctx context.Context) error {
	_, err := p.call(ctx, "aria2.getVersion", nil)
	return err
}
func (p *Aria2) Handle(ctx context.Context, task plugin.Task) (any, string, error) {
	rawURL := inputString(task.Input, "url", "link", "optimal_link")
	if rawURL == "" {
		return nil, "", errors.New("aria2 plugin requires input.url")
	}
	if err := validateDownloadURL(rawURL); err != nil {
		return nil, "", err
	}
	params := []any{}
	if p.cfg.Secret != "" {
		params = append(params, "token:"+p.cfg.Secret)
	}
	params = append(params, []string{rawURL})
	if options, ok := task.Input["options"].(map[string]any); ok {
		params = append(params, options)
	}
	result, err := p.call(ctx, "aria2.addUri", params)
	if err != nil {
		return nil, "", err
	}
	gid := fmt.Sprint(result)
	if gid == "" {
		return nil, "", errors.New("aria2 returned empty gid")
	}
	return map[string]any{"gid": gid, "url": rawURL}, gid, nil
}

func (p *Aria2) call(ctx context.Context, method string, params []any) (any, error) {
	request := map[string]any{"jsonrpc": "2.0", "id": fmt.Sprintf("plugin-%d", time.Now().UnixNano()), "method": method, "params": params}
	encoded, _ := json.Marshal(request)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.JsonRpc, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aria2 request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("aria2 returned status %d", resp.StatusCode)
	}
	var value struct {
		Result any `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&value); err != nil {
		return nil, err
	}
	if value.Error != nil {
		return nil, fmt.Errorf("aria2 error %d: %s", value.Error.Code, value.Error.Message)
	}
	return value.Result, nil
}

func (p *CloudDriver) do(ctx context.Context, method, path string, body any, output any) error {
	if strings.TrimSpace(p.cfg.BaseURL) == "" {
		return errors.New("cloud_driver.base_url is not configured")
	}
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(p.cfg.BaseURL, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if p.cfg.ProfileID != "" {
		req.Header.Set("X-Profile-ID", p.cfg.ProfileID)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("cloud-driver request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cloud-driver returned status %d", resp.StatusCode)
	}
	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Msg     string          `json:"msg"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		message := envelope.Message
		if message == "" {
			message = envelope.Msg
		}
		return fmt.Errorf("cloud-driver error %d: %s", envelope.Code, message)
	}
	if output != nil && len(envelope.Data) > 0 {
		return json.Unmarshal(envelope.Data, output)
	}
	return nil
}

func (p *CloudDriver) driverPath(path string) string {
	platform := p.cfg.Platform
	if platform == "" {
		platform = "115"
	}
	return "/drivers/" + url.PathEscape(platform) + path
}
func inputString(input map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := input[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
func validateDownloadURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" {
		return errors.New("download URL is invalid")
	}
	if !strings.EqualFold(parsed.Scheme, "magnet") && !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("unsupported download URL scheme: %s", parsed.Scheme)
	}
	return nil
}
