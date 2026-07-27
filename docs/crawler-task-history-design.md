# 采集任务历史设计方案

## 1. 背景

当前项目已经能够查看采集引擎的实时状态，但不能回答下面这些日常问题：

- 一次手动、定时或快速提交的采集是否真正执行完成？
- 本次采集展开了多少列表页和详情页，成功、失败、重试各有多少？
- 本次新增了多少磁力资源，多少资源因重复而跳过？
- 某个 URL 为什么失败，已经重试了几次？
- 服务重启时有哪些任务被中断，是否可以重新执行？

当前实现只在内存中维护 `CrawlerTask`：

- `TaskEntry` 没有任务 ID、批次 ID和父任务 ID。
- `TaskQueue` 是容量为 512 的内存 channel。
- `Worker` 只暴露当前 URL 和是否运行。
- 子任务由解析器返回后继续入队，无法关联到最初的触发行为。
- 错误重试复用同一个内存对象，重启后记录丢失。
- `Manager.Run`、Cron、快速提交和启动时执行的入口没有统一的运行实例。
- 两个 JavDB Crawler 当前都返回 `Name() == "JavDB"`，历史记录无法准确区分普通 JavDB 和女优页任务。

因此不能只在接口入口增加一张简单日志表。历史记录必须覆盖一次采集运行、运行中派生的逻辑任务以及每次执行尝试。

## 2. 设计目标

本方案目标如下：

1. 每次采集触发都生成可追踪的运行记录。
2. 根任务和派生子任务都归属于同一次运行。
3. 保存每个逻辑任务的状态、父子关系、重试次数和最终错误。
4. 保存每次执行尝试，能够看到第几次失败、在哪个 Worker 执行以及耗时。
5. 汇总任务数、成功数、失败数、发现资源数、新增数和重复数。
6. 服务重启后不留下永久“执行中”的脏状态。
7. 前端提供历史列表、运行详情和失败任务查看能力。
8. 快速提交接口返回运行 ID，用户提交后可以立即追踪。
9. 保持采集与下载解耦，采集历史不追踪后续下载过程。

## 3. 非目标

第一版不包含：

- 修改采集器 Cron 配置。
- 暂停或取消正在执行的任务。
- WebSocket/SSE 实时推送。
- 跨运行复用已经完成的任务。
- 将下载状态纳入采集运行状态。
- 自动无限重试失败运行。

取消任务、实时推送可以在历史功能稳定后继续扩展。

## 4. 核心概念

采用三级模型：

```text
CrawlerRun（一次采集运行）
└── CrawlerTask（一个逻辑 URL 任务）
    ├── CrawlerTaskAttempt（第 1 次执行）
    ├── CrawlerTaskAttempt（第 2 次执行）
    └── CrawlerTaskAttempt（第 N 次执行）
```

### 4.1 CrawlerRun

表示用户或系统发起的一次完整采集。

示例：

- 用户提交一个 JavDB 详情 URL。
- 用户点击“运行 SeHuaTang”。
- Cron 触发一次 JavDB 全量采集。
- 应用启动后自动执行一次所有采集源。

一次运行可以只有一个根任务，也可以有多个根任务。根任务解析出的列表页、下一页和详情页都属于同一个运行。

### 4.2 CrawlerTask

表示一个需要处理的逻辑 URL。任务在重试时仍使用同一个任务 ID，只增加 `attempt_count`，避免一次失败重试被统计成多个任务。

任务通过 `parent_task_id` 形成父子关系：

```text
JavDB 列表首页
├── JavDB 详情 A
├── JavDB 详情 B
└── JavDB 列表下一页
    ├── JavDB 详情 C
    └── JavDB 详情 D
```

### 4.3 CrawlerTaskAttempt

表示一次实际执行。它记录 Worker、开始时间、结束时间、耗时和本次错误，用于分析重试过程。

## 5. 状态设计

### 5.1 运行状态

```go
const (
    CrawlerRunPending       = "pending"
    CrawlerRunRunning       = "running"
    CrawlerRunSucceeded     = "succeeded"
    CrawlerRunPartialFailed = "partial_failed"
    CrawlerRunFailed        = "failed"
    CrawlerRunInterrupted   = "interrupted"
)
```

状态语义：

| 状态 | 含义 |
| --- | --- |
| `pending` | 运行记录已创建，根任务尚未开始 |
| `running` | 至少一个任务已开始或仍有待处理任务 |
| `succeeded` | 所有逻辑任务终态成功或跳过 |
| `partial_failed` | 运行产出了有效结果，但仍有任务最终失败 |
| `failed` | 所有有效任务失败，未产出任何有效结果 |
| `interrupted` | 服务退出或重启导致运行未正常结束 |

“没有新资源”不等于失败。页面正常解析但所有资源均已存在时，运行仍为 `succeeded`。

### 5.2 任务状态

```go
const (
    CrawlerTaskQueued    = "queued"
    CrawlerTaskRunning   = "running"
    CrawlerTaskRetryWait = "retry_wait"
    CrawlerTaskSucceeded = "succeeded"
    CrawlerTaskSkipped   = "skipped"
    CrawlerTaskFailed    = "failed"
    CrawlerTaskInterrupted = "interrupted"
)
```

任务状态流转：

```text
queued -> running -> succeeded
                  -> skipped
                  -> retry_wait -> queued
                  -> failed
                  -> interrupted
```

`skipped` 用于 URL、编号或内容已处理，无需再次采集的正常情况，不计入失败数。

### 5.3 执行尝试状态

- `running`
- `succeeded`
- `failed`
- `panic`
- `interrupted`

## 6. 触发来源

`trigger_type` 使用固定值：

| 值 | 来源 |
| --- | --- |
| `manual` | 用户在“采集状态”页面点击运行 |
| `quick_submit` | 用户提交指定 URL |
| `cron` | 定时任务触发 |
| `startup` | `exec_on_startup` 触发 |
| `quick_api` | 外部 Quick API 触发 |
| `retry` | 从历史记录重新执行 |

`created_by` 保存 JWT Subject；非登录入口保存 `system` 或 `quick-api`。

## 7. 采集源唯一标识

当前普通 JavDB 和女优 JavDB 都返回 `"JavDB"`，必须先拆分“稳定唯一键”和“显示名称”：

```go
type Crawler interface {
    Key() string
    Name() string
    CronSpec() string
    RootTasks() []TaskSpec
    BuildTask(spec TaskSpec) (CrawlerTask, error)
}
```

建议键值：

| Key | 显示名称 |
| --- | --- |
| `javdb` | JavDB |
| `javdb-actors` | JavDB 女优 |
| `sehuatang` | 色花堂 |

数据库和 API 使用 `provider_key`，前端显示 `provider_name`。稳定键后续不得随中文名称变化。

## 8. 数据库设计

建议新增迁移：

- `internal/db/migrate/migrate_v1_0_10.go`

建议新增表结构文件：

- `internal/db/table/crawler_run.go`
- `internal/db/table/crawler_task.go`
- `internal/db/table/crawler_task_attempt.go`

### 8.1 crawler_runs

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | bigint | 主键 |
| `provider_key` | varchar(64) | 采集源唯一键 |
| `provider_name` | varchar(128) | 创建时的显示名称快照 |
| `trigger_type` | varchar(32) | 触发来源 |
| `status` | varchar(32) | 运行状态 |
| `input_type` | varchar(32) | 快速提交类型，如 `list`、`detail` |
| `input_url` | text | 快速提交 URL；全量运行可为空 |
| `created_by` | varchar(128) | 用户 ID 或系统来源 |
| `retry_of_run_id` | bigint null | 重试来源运行 ID |
| `total_tasks` | integer | 发现的逻辑任务总数 |
| `pending_tasks` | integer | 尚未进入终态的任务数 |
| `queued_tasks` | integer | 当前排队数 |
| `running_tasks` | integer | 当前执行数 |
| `succeeded_tasks` | integer | 成功数 |
| `skipped_tasks` | integer | 正常跳过数 |
| `failed_tasks` | integer | 最终失败数 |
| `output_count` | integer | 解析得到的资源数 |
| `inserted_count` | integer | 实际新增资源数 |
| `duplicate_count` | integer | 重复或已存在资源数 |
| `error_summary` | text | 运行级错误摘要 |
| `started_at` | timestamp null | 开始时间 |
| `finished_at` | timestamp null | 结束时间 |
| `created_at` | timestamp | 创建时间 |
| `updated_at` | timestamp | 更新时间 |

索引：

```sql
CREATE INDEX idx_crawler_runs_created_at
    ON crawler_runs (created_at DESC);
CREATE INDEX idx_crawler_runs_provider_created
    ON crawler_runs (provider_key, created_at DESC);
CREATE INDEX idx_crawler_runs_status_created
    ON crawler_runs (status, created_at DESC);
CREATE INDEX idx_crawler_runs_trigger_created
    ON crawler_runs (trigger_type, created_at DESC);
```

### 8.2 crawler_tasks

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | bigint | 主键 |
| `run_id` | bigint | 所属运行 |
| `parent_task_id` | bigint null | 父任务 |
| `provider_key` | varchar(64) | 采集源唯一键 |
| `task_type` | varchar(32) | `list`、`detail` 等可重建类型 |
| `raw_url` | text | 原始 URL |
| `url_hash` | char(64) | 规范化 URL 的 SHA-256 |
| `status` | varchar(32) | 任务状态 |
| `attempt_count` | integer | 已执行次数 |
| `max_attempts` | integer | 最大执行次数，默认 6 |
| `worker_id` | integer null | 当前或最后执行 Worker |
| `output_count` | integer | 解析输出数 |
| `inserted_count` | integer | 新增资源数 |
| `duplicate_count` | integer | 重复资源数 |
| `last_error` | text | 最后一次错误 |
| `available_at` | timestamp | 可被调度时间，用于退避重试和延迟根任务 |
| `queued_at` | timestamp | 入队时间 |
| `started_at` | timestamp null | 首次开始时间 |
| `finished_at` | timestamp null | 进入终态时间 |
| `created_at` | timestamp | 创建时间 |
| `updated_at` | timestamp | 更新时间 |

索引和约束：

```sql
CREATE UNIQUE INDEX uk_crawler_tasks_run_url
    ON crawler_tasks (run_id, provider_key, task_type, url_hash);
CREATE INDEX idx_crawler_tasks_run_status
    ON crawler_tasks (run_id, status, id);
CREATE INDEX idx_crawler_tasks_dispatch
    ON crawler_tasks (status, available_at, id);
CREATE INDEX idx_crawler_tasks_parent
    ON crawler_tasks (parent_task_id);
```

`url_hash` 避免直接为超长 URL 建唯一索引。写入前需要先规范化 URL：

- scheme 和 host 转小写。
- 删除 fragment。
- query 参数按键排序。
- 保留会改变页面内容的 query 参数。

### 8.3 crawler_task_attempts

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | bigint | 主键 |
| `run_id` | bigint | 所属运行，便于查询 |
| `task_id` | bigint | 所属逻辑任务 |
| `attempt_no` | integer | 第几次执行 |
| `worker_id` | integer | Worker ID |
| `status` | varchar(32) | 尝试状态 |
| `error` | text | 本次错误 |
| `started_at` | timestamp | 开始时间 |
| `finished_at` | timestamp null | 结束时间 |
| `duration_ms` | bigint | 耗时 |

约束和索引：

```sql
CREATE UNIQUE INDEX uk_crawler_task_attempt
    ON crawler_task_attempts (task_id, attempt_no);
CREATE INDEX idx_crawler_attempts_run_started
    ON crawler_task_attempts (run_id, started_at DESC);
```

不建议为三张表配置级联物理删除。历史清理时由仓储层按 run ID 在事务中删除，避免误删审计数据。

## 9. 任务元数据改造

`CrawlerTask` 增加只读元数据：

```go
type CrawlerTask interface {
    ID() int64
    RunID() int64
    ParentTaskID() int64
    ProviderKey() string
    TaskType() string
    RawOrigin() string
    RawUrl() string
    Handler() TaskHandler
}
```

`TaskEntry` 增加：

```go
IDValue           int64
RunIDValue        int64
ParentTaskIDValue int64
ProviderKeyValue  string
TaskTypeValue     string
```

解析器返回的子任务不要求主动填写运行信息。`Engine.Success` 在登记子任务时统一继承：

- `run_id = 当前任务.run_id`
- `parent_task_id = 当前任务.id`
- `provider_key = 当前任务.provider_key`

`task_type` 必须由创建任务时明确指定，不能从 Handler 函数反射推断。

## 10. 提交与执行流程

### 10.1 创建运行

所有入口统一经过 `CrawlerRunService`，不再直接向 `bus.SubmitTask` 发布裸任务。

```text
HTTP/Cron/Startup
  -> CrawlerRunService.CreateRun(...)
  -> Provider 生成根 TaskSpec
  -> 事务插入 crawler_runs + crawler_tasks
  -> 提交根任务 ID 到 Engine
  -> 返回 run_id
```

根任务必须先完整登记，再把运行状态改为 `running`。这样 JavDB 一次运行中的多个入口任务不会因为第一个任务提前结束而错误完成。

JavDB 当前第二个根任务延迟 10 分钟执行，建议改成：

```go
TaskSpec{
    AvailableAt: time.Now().Add(10 * time.Minute),
}
```

不要继续在 `Crawler.Run()` 内 `Sleep(10 * time.Minute)`。

### 10.2 Worker 领取

Worker 从队列获得的是任务 ID，而不是仅存在于内存的 `CrawlerTask`：

1. 使用条件更新把 `queued/retry_wait` 改为 `running`。
2. 条件更新失败说明任务已被其他 Worker 领取，直接跳过。
3. 插入一条状态为 `running` 的 Attempt。
4. 根据 `provider_key + task_type` 重建 Handler。
5. 执行 Handler。

任务 ID 是队列消息，数据库记录才是任务事实来源。

### 10.3 成功处理

成功时在一个数据库事务中完成：

1. 保存采集输出并统计新增、重复数量。
2. 将返回的子任务规范化并批量插入。
3. 相同运行下重复 URL 依靠唯一索引忽略。
4. 当前 Attempt 标记为 `succeeded`。
5. 当前任务标记为 `succeeded` 或 `skipped`。
6. 更新运行计数：
   - 当前任务从 pending 中减 1。
   - 新插入的子任务加入 total、pending 和 queued。
   - 累加输出、新增和重复数。
7. 如果 `pending_tasks == 0`，计算运行最终状态。

事务提交后，再把新插入的子任务 ID 放入内存唤醒队列。

这里要求将 `magnet_repo.Save` 改成返回明确结果：

```go
type SaveResult string

const (
    SaveInserted  SaveResult = "inserted"
    SaveDuplicate SaveResult = "duplicate"
)

func Save(m *table.Magnets) (SaveResult, error)
```

否则无法可靠统计新增和失败。

### 10.4 失败与重试

当前 `MaxTaskErrorNum = 5` 表示首次执行加最多 5 次重试，因此数据库中的 `max_attempts` 建议默认设为 6。

失败后：

1. 结束当前 Attempt，记录完整错误。
2. `attempt_count < max_attempts`：
   - 任务改为 `retry_wait`。
   - 设置 `available_at`。
   - 运行仍保持 `running`，pending 不变。
3. 达到最大次数：
   - 任务改为 `failed`。
   - pending 减 1，failed 加 1。
   - pending 为 0 时结束运行。

建议退避时间：

```text
第 1 次失败：10 秒
第 2 次失败：30 秒
第 3 次失败：1 分钟
第 4 次失败：3 分钟
第 5 次失败：10 分钟
```

### 10.5 运行最终状态

当 `pending_tasks == 0` 时：

```text
failed_tasks == 0
    => succeeded

failed_tasks > 0 且 (inserted_count > 0 或 succeeded_tasks > 0 或 skipped_tasks > 0)
    => partial_failed

其他
    => failed
```

设置 `finished_at`，并由数据库时间计算总耗时。

## 11. 重启恢复

第一版必须处理重启，避免历史长期停留在 `running`。

推荐采用可恢复队列：

1. Engine 启动时将遗留 `running` 任务改回 `queued`。
2. 对应的 `running` Attempt 标记为 `interrupted`。
3. 在 Attempt 错误中记录“服务重启导致执行中断”。
4. 扫描 `queued/retry_wait` 且 `available_at <= now()` 的任务重新放入队列。
5. 定时 feeder 每 5 秒扫描一次到期任务，避免 channel 唤醒丢失。

如果第一期不实现恢复执行，最低要求是：

- 启动时把遗留任务和运行标记为 `interrupted`。
- 历史页面提供“重新运行”，创建一个新 Run。

推荐直接实现恢复队列，因为增加了任务表之后，恢复成本已经不高，而且能解决“数据库已登记、事务提交后进程在入队前崩溃”的窗口问题。

## 12. 重新运行语义

历史记录保持不可变，不把失败任务直接改回成功。

点击“重新运行”时：

1. 创建新的 `crawler_runs`。
2. `trigger_type = retry`。
3. `retry_of_run_id` 指向原运行。
4. 默认只复制原运行中最终失败的任务。
5. 可选“完整重新运行”，重新生成原运行的全部根任务。

这样可以保留原始失败证据，并清晰比较前后两次运行。

## 13. 后端接口

### 13.1 创建运行

现有接口保留路径，但调整响应：

```http
POST /api/v1/crawler/submit/javdb
POST /api/v1/crawler/submit/javdbPage
POST /api/v1/crawler/run
```

响应：

```json
{
  "code": 0,
  "data": {
    "run_ids": [1024]
  }
}
```

`run` 传空名称表示运行全部采集源，此时每个 Provider 创建独立 Run，因此返回数组。

### 13.2 历史列表

```http
GET /api/v1/crawler/runs
```

参数：

- `page_num`
- `page_size`
- `provider_key`
- `trigger_type`
- `status`
- `created_at_start`
- `created_at_end`
- `keyword`：匹配 Run ID、输入 URL、错误摘要

返回运行摘要和分页总数。

### 13.3 运行详情

```http
GET /api/v1/crawler/runs/{id}
```

返回：

- 运行基本信息。
- 汇总计数。
- 进度百分比。
- 最近错误任务摘要。
- 来源运行和重试运行 ID。

进度计算：

```text
(succeeded_tasks + skipped_tasks + failed_tasks) / total_tasks * 100
```

任务运行过程中可能发现新任务，因此进度允许暂时下降，前端不应假设它严格单调递增。

### 13.4 任务列表

```http
GET /api/v1/crawler/runs/{id}/tasks
```

参数：

- `page_num`
- `page_size`
- `status`
- `task_type`
- `keyword`：URL 或错误

### 13.5 尝试记录

```http
GET /api/v1/crawler/tasks/{id}/attempts
```

按 `attempt_no DESC` 返回。

### 13.6 重新运行

```http
POST /api/v1/crawler/runs/{id}/retry
```

请求：

```json
{
  "mode": "failed"
}
```

`mode` 可选：

- `failed`：仅重试最终失败任务。
- `full`：重新执行原运行的根任务。

返回新的 `run_id`。

## 14. 前端设计

### 14.1 菜单

在“采集管理”下增加：

- 采集状态
- 快速提交
- 采集历史

路由建议：

```text
/crawler/history
```

### 14.2 历史列表

筛选项：

- 采集源
- 触发方式
- 状态
- 时间范围
- 关键字

表格列：

| 列 | 内容 |
| --- | --- |
| 运行 ID | 可点击打开详情 |
| 采集源 | 显示名称 |
| 触发方式 | 手动、定时、快速提交等 |
| 输入 | URL 或“全量运行” |
| 状态 | 带颜色 Tag |
| 进度 | 完成任务数 / 总任务数 |
| 资源 | 新增 / 重复 / 输出 |
| 失败 | 最终失败任务数 |
| 提交人 | 用户或 system |
| 开始时间 | `started_at` |
| 耗时 | 运行中动态显示，结束后固定 |
| 操作 | 详情、重新运行 |

运行中的列表每 5 秒刷新一次；页面不可见时停止轮询。

### 14.3 运行详情

建议使用独立详情页或宽 Drawer，包含：

1. 基本信息。
2. 任务总数、排队、执行中、成功、跳过、失败统计卡片。
3. 输出、新增、重复资源统计。
4. 任务列表。
5. 任务错误及执行尝试记录。
6. 重试来源和后续重试运行链接。

任务表支持只看失败任务，并展示：

- 任务类型。
- URL。
- 父任务 ID。
- 执行次数。
- 最后错误。
- Worker。
- 开始时间和耗时。

### 14.4 快速提交联动

提交成功后不再只提示“采集任务已提交”，而是：

```text
采集任务 #1024 已提交
[查看任务]
```

用户点击后进入 `/crawler/history/1024`。

### 14.5 实时状态联动

现有 Worker 状态增加：

- 当前 Run ID。
- 当前 Task ID。

点击可跳转到历史详情。Engine 的实时状态仍用于观察当前进程，历史页面用于审计和排错，两者不合并。

## 15. 模块划分

建议新增：

```text
internal/
├── crawler/
│   ├── history_service.go
│   ├── task_factory.go
│   └── recovery.go
├── repo/
│   └── crawler_history_repo/
│       ├── run.go
│       ├── task.go
│       └── attempt.go
├── db/
│   ├── table/
│   │   ├── crawler_run.go
│   │   ├── crawler_task.go
│   │   └── crawler_task_attempt.go
│   └── migrate/
│       └── migrate_v1_0_10.go
└── api/
    └── crawler/
        ├── crawler.go
        └── history.go

ui/get-magnet-ui/src/
├── api/crawler/index.ts
├── views/crawler/history/index.vue
└── views/crawler/history/detail.vue
```

职责边界：

- `history_service`：创建运行、登记根任务、重试运行。
- `crawler_history_repo`：事务状态流转和统计。
- `task_factory`：根据 Provider + TaskType 重建可执行任务。
- `Engine`：调度任务 ID、执行和恢复，不直接拼 SQL。
- Provider：生成根任务规格、解析页面和返回子任务。
- API：参数校验和响应，不直接操作状态计数。

## 16. 并发与一致性

以下操作必须通过仓储层事务完成：

1. 创建 Run 和全部根任务。
2. 领取 Task 并创建 Attempt。
3. Task 成功、插入子任务、保存输出、更新 Run 计数。
4. Task 失败、结束 Attempt、安排重试或更新 Run 终态。

领取任务使用条件更新：

```sql
UPDATE crawler_tasks
SET status = 'running', worker_id = ?, ...
WHERE id = ?
  AND status IN ('queued', 'retry_wait')
  AND available_at <= NOW();
```

只有受影响行数为 1 的 Worker 可以执行任务。

运行统计不能由前端临时扫描任务表计算，否则大运行的详情请求会越来越慢。写入时维护计数，同时提供内部一致性校验方法，用任务表重新计算并修复统计。

## 17. 数据保留

历史会持续增长，建议第一版就加入清理配置：

```yaml
crawler:
  history_retention_days: 90
```

规则：

- 只清理已结束且超过保留期的 Run。
- 按 Attempt、Task、Run 顺序分批删除。
- 每批最多 500 个 Run，避免长事务。
- `running/pending` 永不被保留期任务删除。

清理任务本身注册到现有 CronScheduler，每天低峰执行一次。

## 18. 测试方案

### 18.1 仓储层

- 创建 Run 和多个根任务计数正确。
- 同一运行重复 URL 只插入一次。
- 两个 Worker 同时领取同一任务只能成功一个。
- 成功时插入子任务和结束父任务在同一事务完成。
- 重试时 pending 不减少。
- 最终失败时 pending 和 failed 正确变化。
- pending 归零后最终状态计算正确。
- 新增、重复和输出计数正确。

### 18.2 Engine

- 根任务成功且无子任务。
- 列表任务派生多个详情任务。
- 失败后按退避时间重试。
- 达到最大次数后停止重试。
- Handler panic 被记录为 Attempt panic。
- 重启时 running Attempt 被标记中断并恢复任务。
- 数据库任务已创建但未进入 channel 时可被 feeder 找回。

### 18.3 API

- 历史分页和组合筛选。
- 运行、任务和 Attempt 详情。
- 快速提交返回 Run ID。
- 运行全部返回多个 Run ID。
- 失败任务重试创建新 Run，原历史不变化。

### 18.4 前端

- 状态文案、颜色和进度计算。
- 运行中自动刷新、离开页面停止刷新。
- 空历史、无失败任务和超长错误展示。
- 从快速提交跳转到运行详情。

## 19. 分阶段实施

### 第一阶段：数据模型与运行入口

1. 增加三张表和迁移。
2. 为 Provider 增加稳定唯一 Key。
3. 新增 `CrawlerRunService`。
4. 统一手动、Cron、启动、快速提交入口。
5. 接口返回 Run ID。

完成标准：所有新触发行为都能产生运行和根任务记录。

### 第二阶段：任务执行历史

1. 给 Task 增加 Run、Task、Parent、Type 元数据。
2. Worker 记录 Attempt。
3. Success/Error 事务化更新任务和运行计数。
4. `magnet_repo.Save` 返回新增/重复/错误结果。
5. 实现运行最终状态。

完成标准：一次运行的所有派生任务和重试都可追踪，统计准确。

### 第三阶段：恢复与重新运行

1. 队列改为传递 Task ID。
2. 增加数据库 feeder。
3. 实现启动恢复。
4. 实现失败任务和完整运行重试。
5. 增加历史保留清理任务。

完成标准：重启不会丢任务，也不会留下永久运行中状态。

### 第四阶段：前端历史页面

1. 增加历史列表和筛选。
2. 增加运行详情、任务详情和 Attempt 展示。
3. 快速提交增加查看任务入口。
4. 实时 Worker 增加 Run/Task 跳转。

完成标准：用户不查看日志即可定位一次采集的结果和失败原因。

## 20. 验收标准

第一版整体完成需满足：

1. 任意入口触发采集后都返回或生成唯一 Run ID。
2. 历史页面能查询最近 90 天运行。
3. 每个 Run 能看到所有根任务和派生任务。
4. 每个失败任务能看到每次执行错误和耗时。
5. 成功、跳过、失败、新增和重复统计与数据库实际结果一致。
6. 任务达到最大重试次数后运行能够正常结束。
7. 服务在任务执行中重启，恢复后任务继续执行或明确标记中断，不允许永久卡在 `running`。
8. 同一任务不会因多个 Worker 重复执行。
9. 快速提交成功后可以直接进入对应历史详情。
10. `go test ./...` 和前端生产构建通过。

## 21. 关键风险

### 21.1 Provider Handler 无法从数据库重建

当前 Handler 以函数值保存在 `TaskEntry` 中。实现恢复队列前必须建立 `provider_key + task_type -> TaskFactory` 注册机制，否则服务重启后只有 URL，无法知道应调用 `parseList` 还是 `parsePage`。

### 21.2 子任务登记与运行完成竞态

必须先在事务中登记所有子任务，再结束父任务并判断 `pending_tasks`。如果先结束父任务，运行可能短暂归零并被错误标记完成。

### 21.3 统计依赖 Save 返回值

当前 `magnet_repo.Save` 吞掉数据库错误，只记录日志。采集历史上线前必须让它返回结果和错误，否则“新增数”和“成功状态”不可信。

### 21.4 内存 EventBus 丢失关联信息

`bus.SubmitTask` 不应继续作为历史任务的主提交通道。可以保留为唤醒机制，但参数必须是 Task ID；运行和任务必须先持久化。

### 21.5 历史数据增长

列表任务可能派生大量详情任务和 Attempt。如果不设置保留期，三张表会持续增长。索引和定期清理需要与第一版一同设计。

## 22. 推荐结论

推荐将“采集任务历史”实现为可恢复的持久化任务体系，而不是入口日志：

- Run 负责描述一次业务触发和汇总结果。
- Task 负责描述父子任务图和逻辑终态。
- Attempt 负责描述每次真实执行与重试错误。
- 数据库是任务状态事实来源，内存队列只负责唤醒 Worker。
- 重试创建新 Run，原始历史保持不可变。

该设计会改动采集引擎的任务标识和提交路径，但能一次解决历史追踪、失败排查、重启恢复和重复执行问题，后续增加取消、告警或实时推送时也无需再次重构核心模型。
