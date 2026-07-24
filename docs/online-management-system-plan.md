# 在线管理系统功能计划

## 背景

当前 `get-magnet` 后端已经具备磁力链接采集、资源入库、下载调度、网盘离线下载、播放地址代理、JWT 登录等核心能力。`ui/get-magnet-ui` 是基于 Vue3、Element Plus、Pinia、Vue Router 的管理后台模板，已经接入登录、当前用户、磁力链接列表和磁力链接 CRUD 的基础接口。

本计划用于把现有项目整理成可在线使用的管理系统。目标不是重写模板，而是在保留现有布局、登录态、请求封装和菜单机制的基础上，逐步替换模板演示页，补齐后端管理 API，并形成围绕“采集 -> 资源库 -> 下载调度 -> 后处理/播放 -> 运维配置”的完整后台。

## 当前能力盘点

### 后端已具备

- HTTP 服务：`internal/server/router.go`，默认端口 `8093`。
- 健康检查：`GET /healthz`，用于容器和反向代理探活。
- 登录认证：`POST /api/auth/login`、`POST /api/auth/logout`。
- 当前用户：`/api/v1/me`、`/api/v1/me/changePwd`。
- 磁力资源管理：`/api/v1/magnets/list`、`detail`、`create`、`update`、`delete`。
- 快速采集入口：`/quick-api/download/submit/javdb`、`/quick-api/download/submit/javdb_page`。
- 播放入口：`GET /api/play/{number}`，按番号解析网盘媒体地址并跳转。
- 爬虫引擎：`CrawlerEngine`、任务队列、worker、JavDB provider。
- 下载调度：`DownloadScheduler` 按 cron 扫描待下载资源并提交到 `DownloadService`。
- 网盘离线下载：`CloudDownloader` 对接 `cloud_driver` 中间服务，支持任务查询、完成后处理、STRM 生成。
- 数据表：`admin`、`config`、`magnets`、`migrate`。

### 前端已具备

- 管理后台模板基础：登录页、布局、侧边栏、标签页、主题配置、请求拦截器。
- 接口地址配置：`.env.development` 中 `VITE_API_URL = http://localhost:8093/`。
- 登录接口：`src/api/login/index.ts`。
- 当前用户接口：`src/api/user/index.ts`。
- 磁力接口：`src/api/magnet/index.ts`。
- 磁力链接页面：`src/views/magnets/index.vue`，支持分页、关键词、状态筛选、新增、编辑、删除、批量删除。
- 磁力编辑弹窗：`src/views/magnets/dialog.vue`。
- 前端路由中已加入 `/magnets` 菜单。

### 需要校准的问题

- 前端磁力状态只有 `0 待处理`、`1 处理中`、`2 已完成`，后端实际状态为：
  - `0` 已采集，待提交下载
  - `1` 正在提交下载
  - `2` 已提交外部下载任务
  - `3` 下载完成且后处理完成
  - `4` 下载提交失败或外部下载失败
- `POST /api/v1/download/submit` 当前为空实现，无法从管理端直接提交指定资源下载。
- 快速采集接口目前在 `/quick-api` 下且免鉴权，适合脚本调用，但不适合作为在线后台的主入口。
- `config` 表存在，但还没有配置管理 API。
- 模板里大量系统、示例、功能演示页面仍未替换为当前业务页面。
- 后端目前只有单管理员模型，没有角色、审计、操作日志、登录会话管理。

## 产品目标

在线管理系统应服务于三个核心场景：

- 资源运营：查看、搜索、修正、补录磁力资源，跟踪资源从采集到下载完成的全过程。
- 自动化调度：在线触发采集、提交下载、重试失败任务、查看任务状态和错误原因。
- 系统运维：配置爬虫、下载器、网盘中间服务、STRM、数据库和应用运行参数，查看健康状态和关键日志。

第一阶段以单管理员自用系统为目标，不引入复杂 RBAC。当前已确认只做单管理，后续如果需要多人协作，再扩展角色、权限、审计和数据隔离。

## 信息架构

建议将前端菜单整理为以下结构，逐步移除模板演示菜单：

```text
首页
资源管理
  磁力资源
  资源详情
  失败任务
采集管理
  采集任务
  快速提交
  采集源配置
下载管理
  下载队列
  网盘任务
  aria2 管理
播放与媒体
  播放信息
  STRM 文件
系统设置
  基础配置
  下载配置
  爬虫配置
  用户安全
运维监控
  健康检查
  调度任务
  日志查看
```

## 页面规划

### 首页

用途：提供系统状态总览。

核心卡片：

- 今日采集数量、总资源数、待提交下载、下载中、已完成、失败数。
- 下载调度状态：是否启用、cron、最近一轮执行时间、下一轮执行时间。
- 网盘中间服务状态：健康、平台、profile、最近错误。
- 爬虫状态：worker 数、队列长度、最近采集源。

后端需要补齐：

- `GET /api/v1/dashboard/summary`
- `GET /api/v1/system/health`
- `GET /api/v1/jobs/status`

### 磁力资源

基于当前 `src/views/magnets` 继续增强。

列表字段建议：

- 编号、标题、女优、来源、状态、优选链接、任务 ID、重试次数、最后提交时间、完成时间、创建时间。
- 对 `download_error` 使用弹窗或抽屉展示完整错误。
- 对 `links` 提供复制、展开查看、设为优选链接。
- 对 `play_file_path`、`strm_path` 提供复制和跳转。

筛选条件建议：

- 关键词：标题、编号。
- 状态：使用后端 0-4 完整状态。
- 来源：JavDB、SeHuaTang、手动。
- 是否有优选链接。
- 是否已生成 STRM。
- 创建时间、最后提交时间、完成时间。

操作建议：

- 新增、编辑、删除、批量删除。
- 提交下载。
- 重试下载。
- 标记失败、标记已完成。
- 重新生成 STRM。
- 打开播放地址。
- 复制优选链接、复制所有链接。

后端需要补齐：

- `POST /api/v1/magnets/submitDownload`
- `POST /api/v1/magnets/retryDownload`
- `POST /api/v1/magnets/markStatus`
- `POST /api/v1/magnets/rebuildSTRM`
- `GET /api/v1/magnets/statusOptions`
- `GET /api/v1/magnets/sourceOptions`

### 资源详情

详情页或右侧抽屉展示单条资源完整生命周期。

内容建议：

- 基础信息：标题、编号、女优、来源、原始 URL、创建/更新时间。
- 磁力信息：优选链接、全部链接、链接数量。
- 下载信息：状态、任务 ID、重试次数、错误原因、最后提交、完成时间。
- 后处理信息：是否完成、播放文件 ID、播放文件路径、文件大小、STRM 路径。
- 操作记录：采集、提交、失败、完成、手动修改。

后端需要补齐：

- `GET /api/v1/magnets/detail` 已存在，但建议扩展为返回状态文案、链接统计、播放 URL。
- `GET /api/v1/magnets/events?id=xxx`，需要新增操作事件表或日志索引。

### 采集管理

目标是把 `/quick-api` 的脚本入口沉淀为后台可操作功能。

页面一：快速提交

- 输入 JavDB 详情页 URL 或列表页 URL。
- 选择采集类型：详情页、列表页、按番号。
- 提交后展示任务提交结果。

页面二：采集任务

- 查看任务队列长度、worker 数、执行中任务、失败任务。
- 手动触发全部爬虫。
- 手动触发指定爬虫。
- 查看最近采集结果。

页面三：采集源配置

- JavDB：cron、是否启用、入口 URL、Cookie/Profile、限速。
- SeHuaTang：先规划，后续启用 provider 后接入。

后端需要补齐：

- 将 `/quick-api/download/submit/javdb` 映射为鉴权 API：`POST /api/v1/crawler/submit/javdb`。
- 将 `/quick-api/download/submit/javdb_page` 映射为鉴权 API：`POST /api/v1/crawler/submit/javdbPage`。
- `POST /api/v1/crawler/run`
- `GET /api/v1/crawler/status`
- `GET /api/v1/crawler/tasks`
- `GET /api/v1/crawler/providers`
- `POST /api/v1/crawler/providers/update`

### 下载管理

目标是把自动调度过程可视化。

页面一：下载队列

- 待提交、提交中、下载中、失败、完成五个 tab。
- 批量提交、批量重试、批量忽略。
- 展示调度配置：enabled、submit_cron、batch_size、max_retry。

页面二：网盘任务

- 查询中间服务健康状态。
- 按 `followed_by` 查询网盘离线任务状态。
- 展示文件列表、warnings、error_message。
- 支持重新轮询、清理多余文件、重新生成 STRM。

页面三：aria2 管理

- 当前后端仍保留 aria2 配置和 `/api/aria2/jsonrpc` 代理。
- 如果继续支持 aria2，可内嵌 `/ui/aria-ng/`。
- 如果主路径改为网盘离线下载，可将 aria2 菜单放到“高级/兼容”分组。

后端需要补齐：

- `GET /api/v1/download/queue`
- `POST /api/v1/download/submit`
- `POST /api/v1/download/retry`
- `POST /api/v1/download/runSchedulerOnce`
- `GET /api/v1/download/scheduler`
- `GET /api/v1/cloud-driver/health`
- `GET /api/v1/cloud-driver/tasks/:taskID`

### 播放与 STRM

目标是让资源完成后可以确认播放链路。

页面建议：

- 已完成资源列表。
- 播放文件 ID、播放文件路径、文件大小、STRM 路径。
- 打开播放地址：`/api/play/{number}`。
- 复制 STRM 内容或路径。
- 批量重新生成 STRM。

后端需要补齐：

- `GET /api/v1/media/list`
- `GET /api/v1/media/playURL?number=xxx`
- `POST /api/v1/media/rebuildSTRM`
- `POST /api/v1/media/rebuildSTRMBatch`

### 系统设置

目标是把环境变量和 YAML 配置中的关键项在后台只读展示，方便线上排查配置是否符合预期。

配置分组：

- 应用配置：`port`、`app.external_base_url`。
- 下载配置：`download.enabled`、`download.submit_cron`、`download.batch_size`、`download.max_retry`。
- 网盘配置：`cloud_driver.base_url`、`platform`、`profile_id`、`save_root`、`timeout`、`poll_cron`。
- STRM 配置：`strm.enabled`、`strm.root_dir`、`strm.overwrite`。
- 爬虫配置：`crawler.exec_on_startup`、`crawler.worker_num`、`drission_rod_grpc_ip`、`drission_rod_grpc_port`。
- aria2 配置：`aria2.jsonrpc`、`aria2.secret`、`aria2.move_to.javdb_dir`。

实现建议：

- 第一阶段只读展示运行时配置，避免在线修改配置却无法生效。
- 当前阶段环境变量和配置项只在后台展示，不支持后台直接修改。
- 后续如确实需要在线变更，再评估写入 `config` 表、动态生效、配置版本、变更记录和回滚。

后端需要补齐：

- `GET /api/v1/settings`
- `POST /api/v1/settings/testCloudDriver`
- `POST /api/v1/settings/testAria2`
- `POST /api/v1/settings/testDrissionRod`

### 用户安全

当前只有 admin 表和修改当前密码。

第一阶段：

- 当前用户信息。
- 修改密码。
- 登录超时后的统一跳转。

第二阶段：

- 管理员列表。
- 新增管理员。
- 禁用管理员。
- 重置密码。

第三阶段：

- 角色权限。
- 操作审计。
- 登录日志。

后端需要补齐：

- `GET /api/v1/admins/list`
- `POST /api/v1/admins/create`
- `POST /api/v1/admins/update`
- `POST /api/v1/admins/disable`
- `GET /api/v1/audit/list`

### 运维监控

目标是排查线上问题时不必进容器看日志。

页面建议：

- 服务健康：应用、数据库、网盘中间服务、DrissionRod、aria2。
- 定时任务：cron 列表、最近执行时间、最近错误。
- 日志查看：按日期、级别、关键词读取最近日志。
- 版本信息：构建版本、Git commit、启动时间、运行时长。

后端需要补齐：

- `GET /api/v1/ops/health`
- `GET /api/v1/ops/jobs`
- `GET /api/v1/ops/logs`
- `GET /api/v1/ops/version`

## 后端实施计划

### 第一阶段：补齐管理 API 基座

- 新增 `internal/api/dashboard`，提供首页统计。
- 新增 `internal/api/crawler`，把快速提交和手动触发采集纳入鉴权 API。
- 新增 `internal/api/download` 的真实提交、重试、运行一次调度能力。
- 新增 `internal/api/settings`，先做运行时配置只读展示。
- 调整磁力状态返回，统一后端状态枚举和文案。
- 为关键操作补充参数校验和错误响应。

### 第二阶段：完善资源生命周期

- 磁力详情返回下载、后处理、播放、STRM 的完整信息。
- 下载失败支持手动重试，并清理 `download_error`。
- 支持批量提交待下载资源。
- 支持下载完成资源重新生成 STRM。
- 增加资源操作记录表，例如 `magnet_events`。

### 第三阶段：配置与调度在线化

- 使用 `config` 表承载可在线修改的配置。
- 配置变更后支持部分组件热更新，例如下载调度 cron、batch size、max retry。
- 定时任务调度器暴露任务列表和最近执行结果。
- 爬虫 provider 支持启用/禁用。

### 第四阶段：运维能力

- 健康检查聚合应用、数据库、网盘中间服务、DrissionRod、aria2。
- 日志查询 API，按 log 文件 tail 最近内容。
- 增加版本信息和启动时间。
- Docker Compose 中补齐 PostgreSQL、get-magnet、cloud-driver、aria2 的健康检查。

## 前端实施计划

### 第一阶段：收敛模板菜单

- 保留登录、首页、磁力资源、个人中心。
- 暂时隐藏模板演示菜单：`fun`、`pages`、`make`、`params`、`visualizing`、示例 iframe。
- 将 `/magnets` 改为“资源管理/磁力资源”。
- 建立业务 API 目录：`api/dashboard`、`api/crawler`、`api/download`、`api/settings`、`api/ops`。

### 第二阶段：增强磁力资源页

- 更新状态枚举为后端 0-4。
- 增加下载任务字段、错误字段、后处理字段。
- 增加资源详情抽屉。
- 增加提交下载、重试下载、播放、复制链接、重新生成 STRM 操作。
- 表单补齐 `play_file_id`、`play_file_path`、`strm_path` 等只读展示字段。

### 第三阶段：新增业务页面

- 首页 Dashboard。
- 采集管理：快速提交、任务状态、采集源。
- 下载管理：队列、调度器、网盘任务。
- 系统设置：运行配置展示、连接测试。
- 运维监控：健康状态、调度任务、日志。

### 第四阶段：交互体验优化

- 长任务操作增加 loading、禁用重复提交。
- 下载错误和日志使用抽屉展示，避免表格过宽。
- 批量操作前给出确认和影响数量。
- 状态标签颜色统一。
- 常用复制操作使用 `DocumentCopy` 图标按钮。

## 数据模型规划

### 现有表增强

`magnets` 已经包含大部分生命周期字段，建议继续使用：

- 基础资源：`origin`、`title`、`number`、`optimal_link`、`links`、`raw_url_host`、`raw_url_path`、`actress0`。
- 下载状态：`status`、`followed_by`、`download_error`、`download_retry_count`、`last_submit_at`、`download_completed_at`。
- 后处理：`post_process_done`、`play_file_id`、`play_file_path`、`play_file_size`、`strm_path`。

建议新增索引：

- `idx_magnets_number`
- `idx_magnets_status`
- `idx_magnets_origin`
- `idx_magnets_followed_by`
- `idx_magnets_created_at`
- `idx_magnets_last_submit_at`

### 建议新增表

`magnet_events`

- `id`
- `magnet_id`
- `event_type`
- `message`
- `payload`
- `operator`
- `created_at`

用途：记录采集、编辑、提交、重试、失败、完成、生成 STRM 等生命周期事件。

`job_runs`

- `id`
- `job_name`
- `status`
- `started_at`
- `finished_at`
- `duration_ms`
- `error`

用途：支持调度任务监控。

`operation_logs`

- `id`
- `admin_id`
- `action`
- `resource_type`
- `resource_id`
- `request_payload`
- `created_at`

用途：后续多人管理和审计。

## API 设计约定

- 新增管理 API 统一放在 `/api/v1` 下，并走 JWT 鉴权。
- 继续使用项目现有 `respond.Ok` 和 `respond.Error` 格式。
- 列表接口统一使用 `page_num`、`page_size`、`keyword`。
- 批量操作统一使用 `ids: []`。
- 状态枚举由后端提供 options，前端不硬编码文案。
- 高风险操作必须记录操作日志。
- `/quick-api` 保留为调试 API，在管理端不完善时可手动使用；当前继续免鉴权，但管理端不直接依赖它。

## 权限策略

第一阶段采用单管理员模型，当前不需要 RBAC：

- 登录后可访问全部管理功能。
- `/api/auth/login`、`/api/play`、`/healthz` 保持免鉴权。
- `/quick-api` 当前作为调试 API 继续免鉴权；公网暴露时再评估 token 或网络隔离。

第二阶段再扩展：

- `admin` 超级管理员。
- `operator` 可管理资源和下载任务。
- `viewer` 只读查看。

## 部署与在线化要求

- 线上管理端由后端直接托管静态 UI 文件；开发环境可以前后端分开运行。
- Compose 中使用 `/healthz` 探测 get-magnet。
- PostgreSQL 使用 `pg_isready`。
- cloud-driver 使用 `/health`。
- 线上必须配置强 `JWT_SECRET`。
- `APP_EXTERNAL_BASE_URL` 必须配置为公网可访问地址，用于 STRM 播放 URL。
- 管理后台公网暴露时建议放在反向代理后，启用 HTTPS。

## 里程碑

### M1：可用后台

- 登录可用。
- 首页统计可用。
- 磁力资源列表状态正确。
- 资源新增、编辑、删除可用。
- 手动提交 JavDB 链接可用。
- 手动提交下载、重试失败可用。

### M2：闭环管理

- 下载队列可视化。
- 资源详情展示完整生命周期。
- 网盘任务状态可查。
- 播放地址和 STRM 信息可查。
- 配置只读展示和连接测试可用。

### M3：线上运维

- 调度任务状态可查。
- 日志可查。
- 健康检查聚合可查。
- Docker Compose 健康检查完善。
- 模板演示页面清理完成。

### M4：多人协作

- 管理员管理。
- 角色权限。
- 操作审计。
- 登录日志。

## 验收标准

- 前端菜单只展示当前业务需要的管理页面。
- 用户能在后台完成从提交采集 URL 到资源下载完成的完整操作链路。
- 下载失败时能看到明确原因，并能一键重试。
- 资源完成后能看到播放地址、播放文件、STRM 路径。
- 系统配置和依赖服务状态可见。
- 应用、数据库、cloud-driver 均有健康检查。
- 所有新增管理接口默认走 JWT 鉴权。

## 建议优先级

优先做：

- 状态枚举校准。
- 磁力列表增强。
- 手动采集 API。
- 手动提交/重试下载 API。
- Dashboard 统计。

随后做：

- 下载队列页面。
- 资源详情抽屉。
- 配置只读展示。
- cloud-driver 健康检查和任务查询。

最后做：

- 在线配置写入。
- 操作审计。
- 角色权限。
- 日志查看。

## 阶段 TODO

### P0：基础整理

- [x] 确认管理后台第一阶段只支持单管理员模型，不做 RBAC。
- [x] 确认线上管理端入口路径：线上由后端托管静态 UI 文件，开发环境前后端分开运行。
- [x] 确认 `/quick-api` 作为调试 API 继续免鉴权，管理端完善前可手动使用。
- [x] 统一 API 响应约定，继续使用现有 `respond.Ok`、`respond.Error`。
- [x] 确认前端不需要的菜单可以隐藏，模板代码仅作为实现参考。
- [x] 新增前端业务 API 目录：`dashboard`、`crawler`、`download`、`settings`、`ops`。
- [x] 确认环境变量和配置项目前只做后台展示，不在后台直接更改。

### P1：可用后台

#### 后端

- [x] 新增 `internal/api/dashboard` 模块。
- [x] 实现 `GET /api/v1/dashboard/summary`，返回总资源数、各状态数量、今日采集数、失败数。
- [x] 新增磁力状态 options 接口：`GET /api/v1/magnets/statusOptions`。
- [x] 将磁力状态枚举统一为后端实际 0-4 状态。
- [x] 增强 `magnets/list` 查询条件，支持来源、是否有优选链接、创建时间、最后提交时间、完成时间。
- [x] 实现 `POST /api/v1/crawler/submit/javdb`，替代管理端直接调用 `/quick-api/download/submit/javdb`。
- [x] 实现 `POST /api/v1/crawler/submit/javdbPage`，替代管理端直接调用 `/quick-api/download/submit/javdb_page`。
- [x] 实现 `POST /api/v1/download/submit`，支持按磁力资源 ID 手动提交下载。
- [x] 实现 `POST /api/v1/download/retry`，支持失败资源重试下载。
- [x] 为手动提交和重试下载补充参数校验、状态校验、错误返回。
- [x] 为 `magnets` 表补充常用索引迁移：`number`、`status`、`origin`、`followed_by`、`created_at`。

#### 前端

- [x] 新增首页 Dashboard 页面并接入 `dashboard/summary`。
- [x] 更新磁力资源页面状态筛选，使用后端 0-4 状态。
- [x] 更新磁力资源表格状态标签文案和颜色。
- [x] 在磁力列表增加任务 ID、重试次数、最后提交时间、完成时间、错误提示字段。
- [x] 在磁力列表增加“提交下载”操作。
- [x] 在失败资源上增加“重试下载”操作。
- [x] 新增快速提交页面，支持提交 JavDB 详情页 URL。
- [x] 新增 JavDB 列表页提交入口。
- [x] 对新增、编辑、删除、提交、重试操作统一增加 loading 和错误提示。
- [x] 校准登录态失效跳转逻辑，确保 401 和业务认证错误都能回登录页。

#### 验收

- [x] 登录后可进入后台。
- [x] 首页能看到基础统计。
- [x] 磁力列表状态显示与后端一致。
- [x] 可手动新增、编辑、删除磁力资源。
- [x] 可在后台提交 JavDB 采集任务。
- [x] 可对待下载资源手动提交下载。
- [x] 可对失败资源一键重试。

### P2：资源生命周期闭环

#### 后端

- [x] 扩展 `GET /api/v1/magnets/detail`，返回状态文案、链接统计、播放 URL、后处理信息。
- [x] 新增 `POST /api/v1/magnets/markStatus`，支持手动标记状态。
- [x] 新增 `POST /api/v1/magnets/rebuildSTRM`，支持单资源重新生成 STRM。
- [x] 新增 `POST /api/v1/magnets/rebuildSTRMBatch`，支持批量重新生成 STRM。
- [x] 新增 `GET /api/v1/download/queue`，按状态返回下载队列。
- [x] 新增 `POST /api/v1/download/runSchedulerOnce`，支持手动运行一轮下载调度。
- [x] 新增 `GET /api/v1/download/scheduler`，返回调度器配置和运行状态。
- [x] 新增 `GET /api/v1/cloud-driver/health`，聚合网盘中间服务健康检查。
- [x] 新增 `GET /api/v1/cloud-driver/tasks/{taskID}`，查询网盘离线任务详情。
- [x] 设计并新增 `magnet_events` 表。
- [x] 在采集、编辑、提交下载、重试、失败、完成、生成 STRM 时写入 `magnet_events`。

#### 前端

- [x] 新增资源详情抽屉或详情页。
- [x] 详情中展示基础信息、全部磁力链接、下载信息、后处理信息、播放信息。
- [x] 详情中展示资源事件时间线。
- [x] 新增下载队列页面，按待提交、提交中、下载中、完成、失败分 tab。
- [x] 下载队列支持批量提交、批量重试。
- [x] 新增网盘任务详情弹窗，展示状态、文件列表、warnings、错误信息。
- [x] 在完成资源上增加“打开播放地址”操作。
- [x] 在完成资源上增加“重新生成 STRM”操作。
- [x] 对 `download_error` 使用弹窗或抽屉展示完整内容。
- [x] 增加复制链接、复制任务 ID、复制 STRM 路径等快捷操作。

#### 验收

- [x] 单条资源能完整展示从采集到下载完成的生命周期。
- [x] 下载失败原因在后台可见。
- [x] 下载队列能按状态查看和批量处理。
- [x] 网盘任务能通过后台查询。
- [x] 完成资源能打开播放地址。
- [x] STRM 信息可见并能重新生成。

### P3：配置与运维在线化

#### 后端

- [ ] 新增 `internal/api/settings` 模块。
- [ ] 实现 `GET /api/v1/settings`，只读返回运行时配置。
- [ ] 实现 `POST /api/v1/settings/testCloudDriver`。
- [ ] 实现 `POST /api/v1/settings/testAria2`。
- [ ] 实现 `POST /api/v1/settings/testDrissionRod`。
- [ ] 新增 `internal/api/ops` 模块。
- [ ] 实现 `GET /api/v1/ops/health`，聚合应用、数据库、cloud-driver、DrissionRod、aria2 状态。
- [ ] 实现 `GET /api/v1/ops/jobs`，返回 cron 任务列表和最近执行结果。
- [ ] 实现 `GET /api/v1/ops/version`，返回版本、commit、启动时间、运行时长。
- [ ] 设计 `job_runs` 表，用于记录调度任务执行结果。
- [ ] 在 `CronScheduler` 中记录最近执行时间、耗时、状态、错误信息。
- [ ] 评估日志查询 API 的安全边界，只允许读取 `LogDir` 下的日志文件。

#### 前端

- [ ] 新增系统设置页面，分组展示应用、下载、网盘、STRM、爬虫、aria2 配置。
- [ ] 新增连接测试按钮：cloud-driver、aria2、DrissionRod。
- [ ] 新增运维健康页面，展示依赖服务状态。
- [ ] 新增调度任务页面，展示任务名、cron、最近执行时间、最近结果。
- [ ] 新增版本信息展示。
- [ ] 对配置中的敏感字段做脱敏展示，例如 `jwt_secret`、`aria2.secret`。

#### 部署

- [ ] 为 get-magnet compose 服务加入 `/healthz` healthcheck。
- [ ] 为 PostgreSQL compose 服务加入 `pg_isready` healthcheck。
- [ ] 为 cloud-driver compose 服务加入 `/health` healthcheck。
- [ ] 明确 `APP_EXTERNAL_BASE_URL` 的线上配置方式。
- [ ] 明确前端静态资源构建和部署路径。
- [ ] 检查 Docker 镜像内是否具备健康检查命令，确认使用 `wget` 或安装 `curl`。

#### 验收

- [ ] 后台能看到运行时配置。
- [ ] 后台能测试外部依赖连通性。
- [ ] 后台能看到应用和依赖健康状态。
- [ ] 调度任务状态可见。
- [ ] Compose 中服务健康状态正确。

### P4：多人协作与审计

#### 后端

- [ ] 扩展 `admin` 表，增加启用状态、最后登录时间、备注字段。
- [ ] 新增管理员列表接口：`GET /api/v1/admins/list`。
- [ ] 新增管理员创建接口：`POST /api/v1/admins/create`。
- [ ] 新增管理员更新接口：`POST /api/v1/admins/update`。
- [ ] 新增管理员禁用接口：`POST /api/v1/admins/disable`。
- [ ] 设计角色模型：`admin`、`operator`、`viewer`。
- [ ] 设计并新增 `operation_logs` 表。
- [ ] 在关键写操作中记录操作日志。
- [ ] 新增登录日志记录。
- [ ] 新增审计列表接口：`GET /api/v1/audit/list`。

#### 前端

- [ ] 新增管理员管理页面。
- [ ] 新增角色选择和权限展示。
- [ ] 新增操作审计页面。
- [ ] 新增登录日志页面。
- [ ] 根据角色隐藏不可用菜单和操作按钮。

#### 验收

- [ ] 可新增、禁用、更新管理员。
- [ ] 不同角色看到的菜单和操作不同。
- [ ] 关键操作可在审计日志中追踪。
- [ ] 登录记录可查。

### 技术债清理

- [ ] 移除或隐藏模板演示页面和无业务意义路由。
- [ ] 统一前端接口命名风格，避免 `pageNum` 与 `page_num` 混用。
- [ ] 为前端业务实体补充 TypeScript 类型定义。
- [ ] 统一时间格式展示。
- [ ] 统一错误提示和空状态展示。
- [ ] 后端补充 API 单元测试或 handler 级测试。
- [ ] 后端补充下载状态转换测试。
- [ ] 后端补充配置解析和默认值测试。
- [ ] 更新 README 的在线管理系统说明。
- [ ] 更新 Docker Compose 示例。
