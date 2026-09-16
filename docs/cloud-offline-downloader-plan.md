# 网盘离线下载器对接落地计划

## 目标

在当前 `get-magnet` 项目中新增一个网盘离线下载器，用于把爬虫采集到的磁力链接提交到 `drission-cloud-driver` 中间服务，由中间服务完成网盘离线下载。

本次落地目标是尽量少改 `get-magnet`：

- 不修改爬虫采集流程。
- 不修改 `downloader.DownloadService` 接口。
- 不新增 `DOWNLOADER_TYPE=aria2|cloud` 之类的下载器切换配置。
- 下载器使用哪个实现，由 `internal/bootstrap/bootstrap.go` 中手动写死注册。
- 网盘平台差异、浏览器登录态、离线任务状态转换尽量放在 `drission-cloud-driver` 中间服务中处理。

## 当前项目接入点

当前下载器抽象位于 `internal/downloader/downloader.go`：

```go
type DownloadService interface {
    bean.Lifecycle

    Download(category string, url string) (string, error)
    OnComplete(callback DownloadCallback)
    OnError(callback DownloadCallback)
}
```

爬虫引擎只依赖这个接口：

- `internal/crawler/engine.go` 调用 `Download(category, url)` 提交下载。
- 下载成功后把返回的任务 ID 保存到 `magnets.followed_by`。
- 下载器完成或异常时可通过 `OnComplete`、`OnError` 触发后续处理。

因此新增网盘下载器时，不需要改爬虫和任务队列。

## 中间服务接口要求

`drission-cloud-driver` 需要向 `get-magnet` 提供稳定的 HTTP API。中间服务可以内部对接 115、PikPak、夸克、迅雷等不同网盘，但对 `get-magnet` 暴露统一结构。

### 必需接口

#### 健康检查

```http
GET /health
```

用途：

- `CloudDownloader.Start()` 启动时检查中间服务是否可用。
- 后续断线重试时判断服务恢复。

建议响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "ok"
  },
  "error": null
}
```

#### 提交离线任务

```http
POST /drivers/:platform/offline/add
Header: X-Profile-ID: <profile_id>
Content-Type: application/json
```

请求体：

```json
{
  "url": "magnet:?xt=urn:btih:xxx",
  "category": "JavDB",
  "save_path": "/get-magnet/JavDB/2026-05-30",
  "client_task_id": "optional-idempotency-key",
  "metadata": {
    "origin": "JavDB"
  }
}
```

字段说明：

- `url`: 磁力链接或其他可离线下载链接。
- `category`: 当前来源，例如 `JavDB`。
- `save_path`: 网盘保存目录。建议由 `get-magnet` 生成，也可以由中间服务按规则兜底。
- `client_task_id`: 幂等键，可选。后续如果需要避免重复提交，可以用磁力 hash 或 `origin + url` 生成。
- `metadata`: 透传信息，中间服务可以记录但不强依赖。

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "cloud-task-id",
    "provider_task_id": "115-task-id",
    "status": "pending"
  },
  "error": null
}
```

`get-magnet` 只依赖 `data.task_id`。这个 ID 会写入 `magnets.followed_by`。

#### 查询任务状态

```http
GET /drivers/:platform/offline/tasks/:id
Header: X-Profile-ID: <profile_id>
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "cloud-task-id",
    "provider_task_id": "115-task-id",
    "status": "completed",
    "name": "example",
    "progress": 100,
    "save_path": "/get-magnet/JavDB/2026-05-30",
    "error_code": "",
    "error_message": "",
    "files": [
      {
        "file_id": "file-id",
        "name": "example.mp4",
        "path": "/get-magnet/JavDB/2026-05-30/example.mp4",
        "size": 123456789
      }
    ]
  },
  "error": null
}
```

任务状态建议统一为：

- `pending`: 已提交，等待网盘开始处理。
- `running`: 正在离线下载。
- `completed`: 离线任务完成，文件已进入网盘目录。
- `failed`: 离线任务失败。
- `canceled`: 任务被取消。
- `unknown`: 中间服务无法确认状态。

#### 查询任务列表

```http
GET /drivers/:platform/offline/tasks
Header: X-Profile-ID: <profile_id>
```

用途：

- 补偿轮询。
- `get-magnet` 重启后恢复正在跟踪的任务。

建议支持查询参数：

```text
status=pending,running,completed,failed
limit=100
```

### 建议接口

#### 删除离线任务

```http
DELETE /drivers/:platform/offline/tasks/:id
Header: X-Profile-ID: <profile_id>
```

用于后续手动清理失败任务或取消任务。

#### 搜索文件

```http
GET /drivers/:platform/fs/search?keyword=xxx
Header: X-Profile-ID: <profile_id>
```

用于任务状态丢失时，通过番号、标题或磁力 hash 反查最终文件。

#### 获取文件下载链接

```http
GET /drivers/:platform/media/url?file_id=xxx
Header: X-Profile-ID: <profile_id>
```

当前阶段不是必需，但后续如果 `get-magnet` 要提供在线播放或转存后的下载入口，可以直接复用。

## get-magnet 实现方案

### 新增配置

不增加下载器类型切换配置，只增加网盘中间服务本身需要的配置。

建议在 `internal/config/config.go` 中新增：

```go
type Config struct {
    // ...
    CloudDriver *CloudDriverConfig `json:"cloud_driver,omitempty" mapstructure:"cloud_driver"`
}

type CloudDriverConfig struct {
    BaseURL   string `json:"base_url,omitempty" mapstructure:"base_url"`
    Platform  string `json:"platform,omitempty" mapstructure:"platform"`
    ProfileID string `json:"profile_id,omitempty" mapstructure:"profile_id"`
    SaveRoot  string `json:"save_root,omitempty" mapstructure:"save_root"`
    Timeout   int    `json:"timeout,omitempty" mapstructure:"timeout"`
    PollCron  string `json:"poll_cron,omitempty" mapstructure:"poll_cron"`
}
```

默认值建议：

```go
v.SetDefault("cloud_driver.platform", "115")
v.SetDefault("cloud_driver.save_root", "/get-magnet")
v.SetDefault("cloud_driver.timeout", 30)
v.SetDefault("cloud_driver.poll_cron", "*/10 * * * *")
```

环境变量：

```go
v.BindEnv("cloud_driver.base_url")
v.BindEnv("cloud_driver.platform")
v.BindEnv("cloud_driver.profile_id")
v.BindEnv("cloud_driver.save_root")
v.BindEnv("cloud_driver.timeout")
v.BindEnv("cloud_driver.poll_cron")
```

### 新增下载器包

新增目录：

```text
internal/downloader/cloud_downloader/
├── cloud_downloader.go
├── cloud_client.go
└── types.go
```

职责划分：

- `cloud_downloader.go`: 实现 `downloader.DownloadService` 和生命周期。
- `cloud_client.go`: 封装 HTTP 请求、响应解析、错误处理。
- `types.go`: 定义中间服务请求和响应结构。

### CloudDownloader 行为

#### Start

启动时执行：

1. 从 bean context 获取 `config.Config`。
2. 读取 `CloudDriverConfig`。
3. 初始化 HTTP client。
4. 调用 `/health` 检查中间服务。
5. 注册定时任务，周期性轮询未完成任务。

轮询可以先做成简单版本：

- 查询本地数据库中 `post_process_done = false` 且 `followed_by != ''` 的记录。
- 对每条记录调用 `/offline/tasks/:id`。
- `completed` 时触发完成处理。
- `failed` 或 `canceled` 时触发错误回调。

如果暂时不想新增 repo 查询方法，第一阶段可以只实现提交任务，不实现轮询。第二阶段再补补偿任务。

#### Download

`Download(category, url)` 内部逻辑：

1. 根据 `SaveRoot`、`category`、当前日期生成 `save_path`。
2. 调用中间服务提交离线任务。
3. 返回 `task_id`。

伪代码：

```go
func (d *CloudDownloader) Download(category string, url string) (string, error) {
    savePath := path.Join(d.cfg.SaveRoot, category, util.NowDate("-"))
    resp, err := d.client.AddOfflineTask(category, url, savePath)
    if err != nil {
        return "", err
    }
    return resp.TaskID, nil
}
```

#### OnComplete / OnError

保持和 aria2 下载器一致：

```go
func (d *CloudDownloader) OnComplete(callback downloader.DownloadCallback) {
    d.onComplete = append(d.onComplete, callback)
}

func (d *CloudDownloader) OnError(callback downloader.DownloadCallback) {
    d.onError = append(d.onError, callback)
}
```

### 手动注册下载器

不做配置切换。需要使用网盘下载器时，手动改 `internal/bootstrap/bootstrap.go`。

当前：

```go
bean.MustRegister[downloader.DownloadService](ctx, aria2_downloader.NewAria2DownloadService())
```

改成：

```go
bean.MustRegister[downloader.DownloadService](ctx, cloud_downloader.NewCloudDownloadService())
```

如果要切回 aria2，再手动改回去。

## 后处理策略

aria2 下载完成后会移动本地文件，网盘离线下载没有本地文件移动这一步。因此云盘下载器的后处理应更轻：

1. 离线任务完成后，确认中间服务返回的 `files` 非空。
2. 根据 `task_id` 找到本地 `magnets.followed_by` 对应记录。
3. 标记 `post_process_done = true`。
4. 后续如果要记录网盘文件 ID，可再新增字段或新表。

第一阶段不建议强行复用 `aria2_downloader.handleDownloadComplete`，因为它处理的是本地文件路径、文件筛选和移动逻辑。

建议后续新增云盘专用后处理：

```text
internal/downloader/cloud_downloader/handle_download_complete.go
```

最小实现只做：

- 查找 `magnets.followed_by = task_id`。
- 标记 `post_process_done = true`。
- 日志记录网盘文件列表。

## 数据库改动

第一阶段不需要改表结构。

沿用现有字段：

- `magnets.followed_by`: 保存中间服务返回的 `task_id`。
- `magnets.post_process_done`: 标记网盘离线任务是否完成后处理。
- `magnets.status`: 继续沿用现有下载状态语义。

如果后续需要更完整的网盘文件管理，再考虑新增表：

```text
cloud_files
cloud_tasks
```

不要在第一阶段引入，避免扩大改动范围。

## 实施步骤

### 阶段 1：固定接口契约

在 `drission-cloud-driver` 中确认并固定：

- `POST /drivers/:platform/offline/add`
- `GET /drivers/:platform/offline/tasks/:id`
- 统一响应结构。
- 统一任务状态枚举。
- `X-Profile-ID` Header 行为。

验收标准：

- 使用 curl 可以提交磁力链接。
- 返回稳定的 `task_id`。
- 能通过 `task_id` 查询到任务状态。

### 阶段 2：get-magnet 新增 CloudDownloader

新增：

- `internal/downloader/cloud_downloader/types.go`
- `internal/downloader/cloud_downloader/cloud_client.go`
- `internal/downloader/cloud_downloader/cloud_downloader.go`

实现：

- 读取 `cloud_driver` 配置。
- `Download(category, url)` 提交离线任务。
- 返回中间服务 `task_id`。

验收标准：

- `go test ./...` 通过。
- 爬虫采集到磁力链接后，会提交到中间服务。
- `magnets.followed_by` 保存的是网盘中间服务任务 ID。

### 阶段 3：手动切换注册

在 `internal/bootstrap/bootstrap.go` 中手动替换注册实现：

```go
bean.MustRegister[downloader.DownloadService](ctx, cloud_downloader.NewCloudDownloadService())
```

验收标准：

- 启动后不再连接 aria2。
- 下载提交走 `drission-cloud-driver`。

### 阶段 4：补偿轮询和完成标记

新增云盘任务轮询：

- 定时扫描未完成的本地记录。
- 查询中间服务任务状态。
- `completed` 时标记 `post_process_done = true`。
- `failed/canceled` 时触发错误回调并记录日志。

验收标准：

- get-magnet 重启后仍能继续检查历史未完成任务。
- 网盘任务完成后，本地记录能被标记完成。
- 中间服务短暂不可用时不会导致主程序退出。

### 阶段 5：可选增强

后续再考虑：

- 离线任务幂等提交。
- 文件 ID 落库。
- 网盘文件重命名和目录整理。
- 失败任务重试。
- webhook 回调替代轮询。
- Web UI 显示云盘任务状态和文件入口。

## 风险点

### 网盘状态不稳定

浏览器自动化可能出现登录态失效、页面结构变化、任务状态查询失败。

处理建议：

- 中间服务返回 `unknown`，不要让 `get-magnet` 直接处理平台异常细节。
- `get-magnet` 轮询时遇到 `unknown` 只记录日志，等待下次补偿。

### 重复提交

爬虫重复采集或重试时可能重复提交同一个磁力链接。

处理建议：

- 第一阶段沿用当前 `get-magnet` 的去重逻辑。
- 第二阶段在中间服务支持 `client_task_id` 幂等。

### 后处理语义不同

aria2 是本地文件完成，云盘是网盘文件完成，两者不能完全共用后处理。

处理建议：

- 云盘下载器使用独立后处理。
- 第一阶段只标记任务完成，不做复杂文件移动。

## 最小改动清单

必须改动：

- `internal/config/config.go`: 新增 `CloudDriverConfig`。
- `internal/downloader/cloud_downloader/*`: 新增云盘下载器实现。
- `internal/bootstrap/bootstrap.go`: 手动替换下载器注册。

可选改动：

- `internal/repo/magnet_repo/magnets.go`: 增加查询未完成任务的方法。
- `internal/downloader/cloud_downloader/handle_download_complete.go`: 云盘任务完成后处理。

不改动：

- `internal/downloader/downloader.go`
- `internal/crawler/engine.go`
- 爬虫 provider。
- 数据库表结构。
- Web UI。

