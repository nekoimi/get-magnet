# 采集任务与下载任务解耦改造计划

## 背景

当前采集链路在 `internal/crawler/engine.go` 的 `Engine.Success` 中同时做了两件事：

1. 保存采集到的磁力信息。
2. 立即调用 `downloader.DownloadService.Download` 提交下载任务。

这导致采集任务和下载任务强耦合：采集侧需要感知下载器是否可用、下载提交失败会影响采集结果语义、后续补偿只能依赖已经写入的 `status`/`followed_by` 状态。目标是让采集任务只负责采集和入库，下载任务由后台调度器定期扫描数据库中未下载的磁力记录并提交下载，同时持续同步下载状态。

## 当前实现梳理

### 采集链路

- 入口：
  - `internal/crawler/providers/javdb/javdb.go`
  - 定时执行 `Crawler.Run()`，或通过 `/quick-api/download/submit/javdb`、`/quick-api/download/submit/javdb_page` 发布手动任务。
- 任务执行：
  - `internal/crawler/worker.go`
  - `Worker.do` 调用 `CrawlerTask.Handler()`，返回后续任务和 `[]MagnetEntry`。
- 采集结果处理：
  - `internal/crawler/engine.go`
  - `Engine.Success` 先提交后续采集任务，再遍历 `outputs` 保存 `table.Magnets`。
  - 这里当前会直接调用 `e.downloadService.Download(output.Origin, output.OptimalLink)`。

### 下载链路

- 下载接口：
  - `internal/downloader/downloader.go`
  - `DownloadService.Download(category, url)` 返回外部下载任务 ID。
- 当前启动注册：
  - `internal/bootstrap/bootstrap.go`
  - 当前注册的是 `cloud_downloader.NewCloudDownloadService()`，aria2 实现仍保留。
- 网盘离线下载实现：
  - `internal/downloader/cloud_downloader/cloud_downloader.go`
  - `Download` 负责向网盘中间服务提交离线下载任务。
  - `pollPendingTasks` 定时扫描 `ListPendingPostProcess`，根据 `followed_by` 查询外部任务状态。
  - 完成后执行文件优选、STRM 生成，并标记 `post_process_done = true`。
- aria2 下载实现：
  - `internal/downloader/aria2_downloader/aria2_downloader.go`
  - 通过 aria2 事件和补偿 Job 处理完成事件。
  - `handleDownloadComplete` 完成文件移动并标记 `post_process_done = true`。

### 数据库状态

`table.Magnets` 当前关键字段：

- `status uint8`
  - 当前代码中 `Engine.Success` 构造时先设为 `1`，提交下载成功后改为 `0`。
  - API 列表支持按 `status` 过滤。
  - 该字段没有集中定义枚举，语义不够清晰。
- `followed_by string`
  - 当前保存外部下载任务 ID。
  - 初始失败或未提交下载时使用 `"unknow"`。
- `post_process_done bool`
  - 表示下载完成后的后处理是否完成，不适合直接表示下载是否完成。
- `play_file_*`、`strm_path`
  - 下载完成后处理产物信息。

## 目标设计

### 职责边界

改造后模块职责如下：

- `crawler.Engine`
  - 只提交后续采集任务。
  - 只保存采集结果。
  - 不依赖 `downloader.DownloadService`。
  - 不调用任何下载提交逻辑。
- 新增下载调度服务，例如 `internal/downloader/scheduler` 或 `internal/downloader/download_job`
  - 生命周期组件，启动时注册 cron。
  - 定期扫描未提交下载的磁力记录。
  - 批量领取任务，调用 `DownloadService.Download`。
  - 成功后写入外部任务 ID，并更新下载状态。
  - 失败后记录失败状态、失败原因和重试信息。
- 具体下载器实现
  - 仍只负责对接 aria2 或网盘中间服务。
  - 继续负责外部任务状态查询和下载完成后的后处理。
  - 下载完成/失败时同步数据库下载状态。

### 状态模型

建议将下载状态从隐式 `status` 语义升级为明确枚举。为了降低 UI/API 改造成本，可以先复用 `magnets.status` 字段，但必须集中定义常量。

建议枚举：

```go
const (
    MagnetStatusCollected uint8 = 0 // 已采集，待提交下载
    MagnetStatusSubmitting uint8 = 1 // 正在提交下载，防止重复领取
    MagnetStatusDownloading uint8 = 2 // 已提交外部下载任务
    MagnetStatusCompleted uint8 = 3 // 下载完成且后处理完成
    MagnetStatusFailed uint8 = 4 // 下载提交失败或外部下载失败
)
```

为了支持后台补偿和失败排查，建议新增字段：

- `download_error text not null default ''`
- `download_retry_count int not null default 0`
- `last_submit_at timestamp null`

可选字段：

- `download_completed_at timestamp null`
- `next_retry_at timestamp null`

第一阶段可以只新增前三个字段；如果需要更完整的退避重试，再加入 `next_retry_at`。

### 状态流转

正常流转：

1. 采集保存：`Collected`
2. 下载调度领取：`Submitting`
3. 提交下载成功：`Downloading`，写入 `followed_by`
4. 外部下载完成并后处理成功：`Completed`，`post_process_done = true`

失败流转：

1. 提交下载失败：`Failed`，记录 `download_error`，`download_retry_count + 1`
2. 外部下载失败：`Failed`，记录外部错误
3. 定时补偿按策略把可重试失败记录重新置为 `Collected` 或直接重新领取

注意：`post_process_done` 仍只表示完成后的整理动作，不参与“是否需要提交下载”的判断。

## 改造步骤

### 1. 定义状态常量

新增文件建议：

- `internal/db/table/magnet_status.go`

内容包含：

- 状态常量。
- 状态说明 map。
- 可选的 `CanSubmitDownload(status uint8) bool` 辅助函数。

这样 API、repo、调度器和下载器都使用同一套语义。

### 2. 修改采集结果保存

修改 `internal/crawler/engine.go`：

- 删除 `downloadService downloader.DownloadService` 字段。
- 删除 `Start` 中对 `DownloadService` 的获取。
- `Success` 中只保存磁力记录。
- 新记录状态设为 `MagnetStatusCollected`。
- `FollowedBy` 建议使用空字符串，不再使用 `"unknow"`。
- 如果保留兼容逻辑，repo 查询中短期仍兼容 `"unknow"`。

保存逻辑目标：

```go
m := &table.Magnets{
    Origin: output.Origin,
    Title: output.Title,
    Number: strings.ToUpper(output.Number),
    OptimalLink: output.OptimalLink,
    Links: output.Links,
    RawURLHost: output.RawURLHost,
    RawURLPath: output.RawURLPath,
    Status: table.MagnetStatusCollected,
    Actress0: output.Actress0,
    FollowedBy: "",
}
magnet_repo.Save(m)
```

### 3. 扩展 repo 方法

在 `internal/repo/magnet_repo/magnets.go` 增加任务领取和状态更新方法：

- `ListPendingDownload(limit int) ([]table.Magnets, error)`
  - 查询 `status in (Collected, Failed)`。
  - `optimal_link <> ''`。
  - `followed_by = '' or followed_by = 'unknow'`。
  - 按 `created_at asc` 或 `updated_at asc`。
- `MarkDownloadSubmitting(id int64) (bool, error)`
  - 条件更新：只有当前仍是 `Collected`/可重试 `Failed` 时才更新为 `Submitting`。
  - 返回是否成功领取，避免多实例或同一轮重复提交。
- `MarkDownloadSubmitted(id int64, followedBy string) error`
  - 更新为 `Downloading`，写入 `followed_by`，清空 `download_error`。
- `MarkDownloadSubmitFailed(id int64, err error) error`
  - 更新为 `Failed`，增加重试次数，写入错误信息。
- `MarkDownloadCompletedByFollowed(followedBy string) error`
  - 下载完成且后处理成功后更新为 `Completed`。
- `MarkDownloadFailedByFollowed(followedBy string, reason string) error`
  - 外部下载失败时更新为 `Failed`。

重点：领取任务要使用数据库条件更新，不要只依赖内存锁。

### 4. 新增下载调度服务

新增生命周期组件，例如：

- `internal/downloader/scheduler/download_scheduler.go`

职责：

- 从 context 获取：
  - `config.Config`
  - `job.CronScheduler`
  - `downloader.DownloadService`
- 启动时注册 cron：
  - 建议新增配置 `download.submit_cron`，默认 `*/5 * * * *`
  - 建议新增配置 `download.batch_size`，默认 `20`
- 每轮执行：
  - 查询待下载记录。
  - 对每条记录先 `MarkDownloadSubmitting`。
  - 调用 `DownloadService.Download(m.Origin, m.OptimalLink)`。
  - 成功：`MarkDownloadSubmitted(m.Id, taskID)`。
  - 失败：`MarkDownloadSubmitFailed(m.Id, err)`。
- 增加本进程内 `running` 标识，避免同一个 cron 周期重入。

示意流程：

```go
func (s *Scheduler) RunOnce(ctx context.Context) {
    list, err := magnet_repo.ListPendingDownload(s.batchSize)
    if err != nil { return }

    for _, m := range list {
        ok, err := magnet_repo.MarkDownloadSubmitting(m.Id)
        if err != nil || !ok { continue }

        taskID, err := s.downloadService.Download(m.Origin, m.OptimalLink)
        if err != nil {
            _ = magnet_repo.MarkDownloadSubmitFailed(m.Id, err)
            continue
        }
        _ = magnet_repo.MarkDownloadSubmitted(m.Id, taskID)
    }
}
```

### 5. 下载完成/失败时同步状态

网盘下载器：

- `cloud_downloader.handleComplete`
  - `MarkPostProcessDoneWithPlayInfo` 成功后，再调用 `MarkDownloadCompletedByFollowed(task.TaskID)`。
- `cloud_downloader.handleError`
  - 调用 `MarkDownloadFailedByFollowed(task.TaskID, reason)`。

aria2 下载器：

- `handleDownloadComplete`
  - `MarkPostProcessDone` 成功后，再调用 `MarkDownloadCompletedByFollowed(status.GID)`。
- `Aria2Downloader` 收到 `ErrorEvent` 后调用 `MarkDownloadFailedByFollowed(e.Id(), reason)`。

注意顺序：只有后处理成功后再置为 `Completed`，否则保持 `Downloading + post_process_done=false`，继续由现有补偿轮询处理。

### 6. 数据库迁移

新增迁移文件，例如：

- `internal/db/migrate/migrate_v1_0_6.go`

历史数据处理策略：所有改造前已经存在的 `magnets` 记录统一视为已处理完成，不再重新进入下载调度队列。这样上线后不会把历史磁力重新提交到 aria2 或网盘离线下载服务。

建议 SQL：

```sql
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS download_error text NOT NULL DEFAULT '';
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS download_retry_count integer NOT NULL DEFAULT 0;
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS last_submit_at timestamp NULL;
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS download_completed_at timestamp NULL;

UPDATE magnets
SET followed_by = ''
WHERE followed_by = 'unknow';

UPDATE magnets
SET status = 3,
    post_process_done = true,
    download_error = '',
    download_completed_at = COALESCE(updated_at, created_at, NOW());
```

迁移完成后，只有新采集入库的记录会使用 `MagnetStatusCollected = 0`，并被下载调度器扫描提交。历史记录统一是 `MagnetStatusCompleted = 3`，不再补偿下载，也不再执行后处理。

### 7. 配置扩展

建议新增：

```go
type DownloadConfig struct {
    SubmitCron string `json:"submit_cron,omitempty" mapstructure:"submit_cron"`
    BatchSize int `json:"batch_size,omitempty" mapstructure:"batch_size"`
    MaxRetry int `json:"max_retry,omitempty" mapstructure:"max_retry"`
}
```

挂载到 `config.Config`：

```go
Download *DownloadConfig `json:"download,omitempty" mapstructure:"download"`
```

默认值：

- `download.submit_cron = "*/5 * * * *"`
- `download.batch_size = 20`
- `download.max_retry = 5`

环境变量：

- `DOWNLOAD_SUBMIT_CRON`
- `DOWNLOAD_BATCH_SIZE`
- `DOWNLOAD_MAX_RETRY`

### 8. 启动注册

修改 `internal/bootstrap/bootstrap.go`：

- 保留 `DownloadService` 注册。
- 新增下载调度服务注册，放在下载器之后、HTTP 服务之前。
- 采集引擎不再需要下载器依赖，但调度服务需要下载器依赖。

建议顺序：

1. 配置
2. 数据库
3. CronScheduler
4. DrissionRod
5. DownloadService
6. DownloadScheduler
7. CrawlerManager
8. CrawlerEngine
9. HTTP Server

### 9. API/UI 适配

后端：

- `internal/api/magnets/magnets.go`
  - 继续支持按 `status` 过滤。
  - 返回新增字段 `download_error`、`download_retry_count`、`last_submit_at`。
- 可选新增接口：
  - `POST /api/v1/magnets/retry-download`
  - 将指定 `Failed` 记录重置为 `Collected`。

前端：

- 磁力列表状态展示改为明确文案：
  - 待下载、提交中、下载中、已完成、失败。
- 失败记录展示错误原因和重试次数。
- 可选增加“重新下载”按钮。

## 兼容策略

### 历史数据兼容

历史记录存在 `followed_by = "unknow"`，且旧 `status` 语义不明确。改造时按“历史数据不再处理”的原则统一收敛：

- 迁移中将 `followed_by = "unknow"` 统一改为空字符串。
- 迁移中将所有历史记录统一置为 `MagnetStatusCompleted`。
- 迁移中将 `post_process_done` 统一置为 `true`，避免后处理补偿任务重新扫描历史记录。
- repo 查询短期仍可兼容 `"unknow"`，后续清理。

### 多实例/重复提交

即使当前项目单实例运行，调度器也应该使用数据库条件更新领取任务：

- 先查待下载列表。
- 再按 ID + 当前状态条件更新为 `Submitting`。
- 只有更新成功的记录才能提交下载。

这样可以避免 cron 重入、手动重试和未来多实例部署导致同一个磁力重复提交。

### 采集重复数据

当前 JavDB 通过 `ExistsByPath` 和 `ExistsByNumber` 做去重。解耦后仍保留这两个判断。

建议后续补充数据库唯一索引：

- `number` 唯一或部分唯一。
- `raw_url_path` 唯一或普通索引。

是否加唯一索引要先确认历史数据是否存在重复。

## 测试计划

### 单元测试

重点补充 repo 和调度器测试：

- `ListPendingDownload` 只返回待下载/可重试记录。
- `MarkDownloadSubmitting` 对状态条件敏感，重复领取只能成功一次。
- `MarkDownloadSubmitted` 正确写入 `followed_by` 和状态。
- `MarkDownloadSubmitFailed` 正确记录错误和重试次数。
- 下载调度器使用 mock `DownloadService`：
  - 提交成功。
  - 提交失败。
  - 空列表。
  - 重入保护。

### 集成验证

本地验证步骤：

1. 执行迁移。
2. 启动服务。
3. 触发 JavDB 采集。
4. 确认采集完成后只入库，状态为 `Collected`，没有立即创建外部下载任务。
5. 等待下载调度 cron 或手动调用 `RunOnce`。
6. 确认记录变为 `Downloading`，`followed_by` 写入外部任务 ID。
7. 模拟下载完成。
8. 确认后处理完成后状态为 `Completed`，`post_process_done = true`。

## 建议实施顺序

1. 定义磁力状态常量和新增字段迁移。
2. 修改 `crawler.Engine`，让采集只入库。
3. 增加 repo 领取、提交成功、失败、完成状态方法。
4. 新增下载调度生命周期组件。
5. 在 bootstrap 中注册调度组件。
6. 修改 cloud/aria2 下载完成和失败回调，同步 `status`。
7. 增加后端 API 字段和前端状态文案。
8. 补测试并跑 `go test ./...`。

## 风险点

- 历史数据会统一标记为已完成，不再自动下载或后处理；如果后续确实要重跑某条历史记录，需要提供手动重置为 `Collected` 的入口。
- 当前 `Save` 方法吞掉错误，只打日志；如果后续需要严格保证采集入库成功，应改为返回 error。
- 网盘下载器已有 `pollPendingTasks`，新调度器不要和它混淆：新调度器负责“未提交下载 -> 已提交下载”，网盘轮询负责“已提交下载 -> 完成后处理”。
- `followed_by` 从 `"unknow"` 改为空字符串后，所有查询要统一兼容并逐步清理旧判断。
