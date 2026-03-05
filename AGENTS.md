# get-magnet 项目上下文

## 项目概述

**get-magnet** 是一个基于 Go 后端和 Vue3 前端的全栈应用，主要功能是磁力链接下载管理系统。系统集成了爬虫采集、aria2 下载器管理和 Web UI 界面，支持从多个源（javdb、sehuatang）自动采集磁力链接并通过 aria2 进行下载。

### 核心功能
- 磁力链接爬虫采集（支持 javdb、sehuatang）
- aria2 下载器集成和管理
- Web UI 管理界面（基于 Element Plus）
- 用户认证（JWT）
- 定时任务调度
- 数据库持久化（PostgreSQL）
- DrissionRod gRPC 客户端集成

## 技术栈

### 后端（Go 1.24.0）
- **Web 框架**: gorilla/mux
- **ORM**: xorm
- **数据库**: PostgreSQL (lib/pq)
- **日志**: logrus + lumberjack（日志轮转）
- **配置**: viper
- **RPC**: gRPC + protobuf
- **下载器**: aria2（通过 arigo 库）
- **定时任务**: robfig/cron
- **认证**: cristalhq/jwt
- **HTML 解析**: goquery
- **缓存**: go-cache

### 前端（Vue3）
- **框架**: Vue 3.4.21
- **UI 组件库**: Element Plus 2.6.1
- **状态管理**: Pinia 2.1.7
- **路由**: Vue Router 4.3.0
- **构建工具**: Vite 5.1.6
- **语言**: TypeScript 5.4.2
- **HTTP 客户端**: axios
- **富文本编辑器**: @wangeditor/editor
- **图表**: echarts + echarts-gl
- **其他**: nprogress、js-cookie、qrcodejs2-fixes

## 项目结构

```
get-magnet/
├── cmd/
│   └── main.go                    # 应用入口
├── internal/
│   ├── api/                       # HTTP API 接口
│   │   ├── v1_*.go               # API 端点（auth、download、file、reverse、setting、ui、user）
│   │   └── middleware/           # 中间件（auth、logging）
│   ├── bean/                      # 依赖注入和生命周期管理
│   │   ├── context.go            # 上下文管理
│   │   ├── lifecycle.go          # 生命周期接口
│   │   ├── manager.go            # Bean 管理器
│   │   └── registry.go           # Bean 注册表
│   ├── bootstrap/                 # 应用启动引导
│   │   └── bootstrap.go          # Bean 生命周期初始化
│   ├── bus/                       # 事件总线
│   ├── config/                    # 配置管理
│   │   └── config.go             # 配置结构定义和加载
│   ├── crawler/                   # 爬虫模块
│   │   ├── crawler.go            # 爬虫接口定义
│   │   ├── engine.go             # 爬虫引擎
│   │   ├── manager.go            # 爬虫管理器
│   │   ├── task_queue.go         # 任务队列
│   │   ├── task.go               # 任务定义
│   │   ├── worker.go             # 工作线程
│   │   ├── download/             # 下载器
│   │   └── providers/            # 爬虫实现（javdb、sehuatang）
│   ├── db/                        # 数据库模块
│   │   ├── database.go           # 数据库初始化
│   │   ├── db.sql                # 数据库初始化脚本
│   │   ├── migrate/              # 数据库迁移
│   │   └── table/                # 数据表定义
│   ├── downloader/                # 下载器模块
│   │   └── aria2_downloader/     # aria2 下载器实现
│   ├── drission_rod/              # DrissionRod gRPC 客户端
│   ├── job/                       # 定时任务
│   ├── logger/                    # 日志模块
│   ├── pkg/                       # 工具包
│   │   ├── apptools/             # 应用工具（自动重启、环境）
│   │   ├── error_ext/            # 错误扩展
│   │   ├── files/                # 文件工具
│   │   ├── jwt/                  # JWT 工具
│   │   ├── queue/                # 队列工具
│   │   ├── request/              # 请求处理
│   │   ├── respond/              # 响应处理
│   │   ├── singleton/            # 单例模式
│   │   └── util/                 # 通用工具（bcrypt、json、sha256、sort、time、url）
│   ├── repo/                      # 数据仓储层
│   └── server/                    # HTTP 服务器
│       ├── router.go             # 路由定义
│       └── server.go             # 服务器实现
├── ui/
│   ├── aria-ng/                   # aria-ng Web UI（第三方）
│   └── get-magnet-ui/             # 主 UI 项目（Vue3）
│       ├── src/                   # 源代码
│       ├── package.json           # 前端依赖和脚本
│       └── vite.config.ts         # Vite 配置
├── proto/                         # Protobuf 定义
├── deploy/                        # 部署配置
├── docker/                        # Docker 相关
├── logs/                          # 日志文件目录
├── tests/                         # 测试文件
├── go.mod                         # Go 模块定义
├── go.sum                         # Go 依赖锁定
├── Makefile                       # 构建脚本
├── Dockerfile                     # Docker 镜像构建
└── README.md                      # 项目说明
```

## 构建和运行

### 后端

**运行开发服务器：**
```bash
go run cmd/main.go
```

**运行测试：**
```bash
go test ./...
# 或使用 Makefile
make test
```

**构建可执行文件：**
```bash
go build -o get-magnet cmd/main.go
```

### 前端（get-magnet-ui）

**安装依赖：**
```bash
cd ui/get-magnet-ui
pnpm install
```

**开发模式：**
```bash
pnpm dev
```

**生产构建：**
```bash
pnpm build
```

**代码检查和修复：**
```bash
pnpm lint-fix
```

## 配置管理

配置通过 viper 管理，支持默认值和环境变量覆盖。主要配置项：

- **Port**: HTTP 服务端口（默认：8093）
- **LogLevel**: 日志级别（默认：debug）
- **LogDir**: 日志目录（默认：logs）
- **JwtSecret**: JWT 密钥（默认：abc123456）
- **Aria2**: aria2 配置
  - `jsonrpc`: aria2 JSON-RPC 地址（环境变量：ARIA2_JSONRPC）
  - `secret`: aria2 验证令牌（环境变量：ARIA2_SECRET）
  - `move_to.javdb_dir`: javdb 下载文件移动目录
- **Crawler**: 爬虫配置
  - `exec_on_startup`: 启动时是否立即执行（默认：false）
  - `worker_num`: 工作线程数量（默认：4）
  - `drission_rod_grpc_ip`: DrissionRod gRPC IP
  - `drission_rod_grpc_port`: DrissionRod gRPC 端口
- **DB**: 数据库配置
  - `dsn`: 数据库连接字符串（环境变量：DB_DSN）

## 开发约定

### 依赖注入模式

项目使用自定义的依赖注入容器（bean 包）：
- 所有需要管理的组件必须实现 `bean.Lifecycle` 接口（可选）
- 使用 `bean.MustRegister` 或 `bean.MustRegisterPtr` 注册组件
- 通过 `bean.PtrFromContext[T]` 或 `bean.FromContext[T]` 获取依赖
- 启动流程在 `bootstrap.BeanLifecycle()` 中定义

### 日志规范

- 使用 logrus 进行日志记录
- 日志文件按日期轮转存储在 logs/ 目录
- 支持多级别日志：debug、info、warning、error、trace

### API 路由

- 所有 API 路由定义在 `internal/api/` 目录
- 使用 gorilla/mux 作为路由器
- 支持 JWT 认证中间件
- 支持 日志记录中间件

### 爬虫开发

爬虫需要实现 `crawler.Crawler` 接口：
```go
type Crawler interface {
    Name() string          // 唯一名称
    CronSpec() string      // 定时表达式（cron 格式）
    Run()                  // 执行任务
}
```

新爬虫需要通过 `crawlerManager.Register()` 注册。

### 数据库迁移

- 使用 xorm 作为 ORM
- 迁移脚本位于 `internal/db/migrate/`
- 迁移版本格式：migrate_vX_X_X.go

## 启动流程

1. 加载配置（`config.Load()`）
2. 初始化数据库（`db.NewDBLifecycle()`）
3. 初始化定时任务调度器（`job.NewCronScheduler()`）
4. 初始化 DrissionRod 客户端
5. 初始化 aria2 下载服务
6. 注册爬虫（javdb、sehuatang）
7. 初始化爬虫管理器和引擎
8. 启动 HTTP 服务器

## 数据库

- 使用 PostgreSQL
- ORM 框架：xorm
- 主要表：
  - admin：管理员
  - config：配置
  - magnets：磁力链接
  - migrate：迁移版本记录

## 依赖关系说明

- 项目 fork 了 `siku2/arigo` 并替换为 `github.com/nekoimi/arigo`（在 go.mod 中定义）
- ui/aria-ng 是 git submodule（在 .gitmodules 中定义）