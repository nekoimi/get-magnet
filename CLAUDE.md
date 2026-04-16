# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 常用命令

### 后端开发

```bash
# 运行服务
go run cmd/main/main.go

# 运行服务（当前目录）
go run cmd/main.go

# 测试（Makefile 中定义）
make test

# 下载依赖
go mod download

# 构建
go build -o get-magnet cmd/main.go
```

### 前端开发

```bash
cd ui/get-magnet-ui

# 安装依赖
pnpm install

# 开发模式
pnpm dev

# 生产构建
pnpm build
```

### 测试单个文件

```bash
# 运行特定包的测试
go test ./internal/pkg/util -v

# 运行特定测试函数
go test ./internal/pkg/util -run TestBcrypt -v
```

## 架构概览

这是一个基于 Go + Vue3 的磁力链接下载管理系统，核心架构特点：

### 依赖注入系统（bean 包）

项目使用自定义的依赖注入容器，而非标准 Go 模式：

- `bean.MustRegister[T]` / `bean.MustRegisterPtr[T]`：注册服务
- `bean.FromContext[T]` / `bean.PtrFromContext[T]`：获取依赖
- 上下文中包含 `Registry`，通过类型安全的方式存储/获取服务

示例：
```go
// 注册
bean.MustRegisterPtr[config.Config](ctx, config.Load())

// 获取
cfg := bean.PtrFromContext[config.Config](ctx)
```

### 生命周期管理

所有需要启动/停止的组件都实现 `bean.Lifecycle` 接口：

```go
type Lifecycle interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

LifecycleManager 负责统一管理：
- 启动时并发启动所有组件
- 收到系统信号（SIGINT/SIGTERM）时顺序关闭所有组件

### 爬虫引擎架构

爬虫系统采用 Worker 池 + 任务队列模式：

1. **Crawler 接口**：`internal/crawler/crawler.go`
   - 实现此接口即可添加新爬虫
   - 方法：`Name()`, `CronSpec()`, `Run()`

2. **CrawlerManager**：管理多个爬虫实例
   - 注册新爬虫：`manager.Register(crawler.BuilderFunc)`
   - 支持定时调度和立即执行

3. **CrawlerEngine**：任务处理引擎
   - 管理多个 Worker
   - 任务通过 `TaskDispatcher` 分发
   - 处理成功/失败回调

4. **Worker**：实际执行任务的协程
   - 从任务队列获取任务
   - 调用任务处理器
   - 失败自动重试（最多 5 次）

5. **事件总线（bus）**：组件间解耦通信
   - 爬虫发布事件，Engine 订阅处理
   - 主要事件：`SubmitTask`, `SubmitJavDB`, `SubmitJavDBPage`

6. **DrDownloader 接口**：爬虫下载器
   - `Get(url string) (string, error)`：下载页面内容
   - 支持普通 HTTP 和 DrissionRod gRPC 两种实现

### 数据库管理

- 使用 xorm ORM 和 PostgreSQL
- 自动迁移：`internal/db/migrate/` 目录包含迁移脚本
- 版本号递增，失败记录到 `Migrates` 表
- 核心表定义：`internal/db/table/`

### 下载器集成

通过 aria2 JSON-RPC 管理下载任务：

- `Aria2Downloader` 实现了 `downloader.DownloadService` 接口
- 支持下载完成/失败回调
- 定时任务：更新 tracker 服务器、检查完成任务并移动文件
- 支持 JavDB 文件自动移动到指定目录

### 配置管理

使用 viper 管理配置：

- 环境变量自动映射，点号替换为下划线（如 `aria2.jsonrpc` → `ARIA2_JSONRPC`）
- 配置定义：`internal/config/config.go`
- 关键环境变量：
  - `DB_DSN`：PostgreSQL 连接字符串
  - `ARIA2_JSONRPC`：aria2 RPC 地址
  - `CRAWLER_DRISSION_ROD_GRPC_IP/PORT`：DrissionRod gRPC 配置

## 关键文件路径

- **应用入口**：`cmd/main.go`
- **依赖注入/生命周期**：`internal/bean/`
- **爬虫引擎**：`internal/crawler/`
  - 爬虫实现：`internal/crawler/providers/javdb/`, `internal/crawler/providers/sehuatang/`
- **下载器**：`internal/downloader/aria2_downloader/`
- **API 路由**：`internal/server/router.go`
- **数据库**：`internal/db/`
- **配置**：`internal/config/config.go`

## 添加新爬虫

1. 创建新包，实现 `crawler.Crawler` 接口
2. 在 `internal/bootstrap/bootstrap.go` 中注册：
   ```go
   crawlerManager.Register(yourcrawler.NewYourCrawler())
   ```
3. 如需 DrissionRod 支持，实现 `DrDownloader` 接口
4. 通过事件总线提交任务

## 前端项目

- 位置：`ui/get-magnet-ui/`
- 技术栈：Vue 3.4 + Element Plus + Pinia + Vite
- 默认端口：开发模式由 Vite 配置，生产构建后由 Go 后端服务
