# get-magnet

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/Vue-3.4-green.svg)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

磁力链接下载管理系统 - 基于 Go + Vue3 的全栈应用，集成爬虫采集、aria2 下载器管理和 Web UI 界面。

## 功能特性

- **磁力链接爬虫采集**：支持从 javdb、sehuatang 等站点自动采集磁力链接
- **aria2 下载器集成**：通过 aria2 JSON-RPC 接口管理下载任务
- **Web UI 管理界面**：基于 Element Plus 的现代化管理后台
- **用户认证**：JWT 认证机制，支持登录/登出
- **定时任务调度**：可配置的爬虫定时执行
- **数据库持久化**：PostgreSQL 数据存储
- **DrissionRod 集成**：gRPC 客户端支持浏览器自动化操作
- **下载完成后自动移动**：支持将下载完成的文件移动到指定目录

## 技术栈

### 后端（Go 1.24）

- **Web 框架**: [gorilla/mux](https://github.com/gorilla/mux)
- **ORM**: [xorm](https://xorm.io/)
- **数据库**: PostgreSQL (lib/pq)
- **日志**: [logrus](https://github.com/sirupsen/logrus) + lumberjack
- **配置**: [viper](https://github.com/spf13/viper)
- **RPC**: gRPC + protobuf
- **下载器**: [aria2](https://aria2.github.io/) (arigo 库)
- **定时任务**: [robfig/cron](https://github.com/robfig/cron)
- **认证**: [cristalhq/jwt](https://github.com/cristalhq/jwt)

### 前端（Vue3）

- **框架**: [Vue 3.4.21](https://vuejs.org/)
- **UI 组件库**: [Element Plus 2.6.1](https://element-plus.org/)
- **状态管理**: [Pinia 2.1.7](https://pinia.vuejs.org/)
- **路由**: [Vue Router 4.3.0](https://router.vuejs.org/)
- **构建工具**: [Vite 5.1.6](https://vitejs.dev/)
- **HTTP 客户端**: [axios](https://axios-http.com/)
- **图表**: [ECharts](https://echarts.apache.org/)

## 快速开始

### 环境要求

- Go 1.24+
- Node.js 16+ / pnpm
- PostgreSQL 12+
- aria2 (可选，用于下载功能)

### 后端运行

```bash
# 1. 克隆项目
git clone https://github.com/nekoimi/get-magnet.git
cd get-magnet

# 2. 安装依赖
go mod download

# 3. 配置环境变量（可选）
export DB_DSN="postgres://user:password@localhost:5432/getmagnet?sslmode=disable"
export ARIA2_JSONRPC="http://localhost:6800/jsonrpc"
export ARIA2_SECRET="your_aria2_secret"

# 4. 运行
go run cmd/main.go
```

服务默认运行在 `http://localhost:8093`

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

### Docker 部署

```bash
# 构建镜像
docker build -t get-magnet:latest .

# 运行容器
docker run -d \
  -p 8093:8093 \
  -e DB_DSN="postgres://user:password@db:5432/getmagnet?sslmode=disable" \
  -e ARIA2_JSONRPC="http://aria2:6800/jsonrpc" \
  -v /path/to/logs:/workspace/logs \
  get-magnet:latest
```

## 配置说明

项目使用 viper 管理配置，支持环境变量覆盖。

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `PORT` | HTTP 服务端口 | `8093` |
| `LOG_LEVEL` | 日志级别 (debug/info/warning/error) | `debug` |
| `LOG_DIR` | 日志目录 | `logs` |
| `JWT_SECRET` | JWT 密钥 | `abc123456` |
| `DB_DSN` | PostgreSQL 连接字符串 | - |
| `ARIA2_JSONRPC` | aria2 JSON-RPC 地址 | - |
| `ARIA2_SECRET` | aria2 验证令牌 | - |
| `ARIA2_MOVE_TO_JAVDB_DIR` | javdb 文件移动目录 | - |
| `CRAWLER_EXEC_ON_STARTUP` | 启动时立即执行爬虫 | `false` |
| `CRAWLER_WORKER_NUM` | 爬虫工作线程数 | `4` |
| `CRAWLER_DRISSION_ROD_GRPC_IP` | DrissionRod gRPC IP | - |
| `CRAWLER_DRISSION_ROD_GRPC_PORT` | DrissionRod gRPC 端口 | - |

## API 接口

### 认证相关

- `POST /api/auth/login` - 用户登录
- `POST /api/auth/logout` - 用户登出

### 用户管理

- `GET /api/v1/me` - 获取当前用户信息
- `POST /api/v1/me/changePwd` - 修改密码

### 磁力链接管理

- `GET /api/v1/magnets/list` - 获取磁力链接列表
- `GET /api/v1/magnets/detail` - 获取磁力链接详情
- `POST /api/v1/magnets/create` - 创建磁力链接
- `POST /api/v1/magnets/update` - 更新磁力链接
- `POST /api/v1/magnets/delete` - 删除磁力链接

### 下载管理

- `POST /api/v1/download/submit` - 提交下载任务
- `POST /quick-api/download/submit/javdb` - 快速提交 javdb 下载
- `POST /quick-api/download/submit/javdb_page` - 提交 javdb 页面下载

### aria2 代理

- `POST /api/aria2/jsonrpc` - aria2 JSON-RPC 代理接口

## 项目结构

```
get-magnet/
├── cmd/
│   └── main.go                    # 应用入口
├── internal/
│   ├── api/                       # HTTP API 接口
│   │   ├── auth/                 # 认证相关
│   │   ├── download/             # 下载管理
│   │   ├── magnets/              # 磁力链接管理
│   │   ├── user/                 # 用户管理
│   │   └── middleware/           # 中间件
│   ├── bean/                      # 依赖注入容器
│   ├── bootstrap/                 # 应用启动引导
│   ├── config/                    # 配置管理
│   ├── crawler/                   # 爬虫模块
│   │   ├── providers/            # 爬虫实现
│   │   │   ├── javdb/           # JavDB 爬虫
│   │   │   └── sehuatang/       # 色花堂爬虫
│   │   └── download/             # 下载器
│   ├── db/                        # 数据库模块
│   │   ├── table/                # 数据表定义
│   │   └── migrate/              # 数据库迁移
│   ├── downloader/                # 下载器模块
│   │   └── aria2_downloader/    # aria2 实现
│   ├── drission_rod/              # DrissionRod gRPC 客户端
│   ├── job/                       # 定时任务
│   ├── logger/                    # 日志模块
│   ├── pkg/                       # 工具包
│   ├── repo/                      # 数据仓储层
│   └── server/                    # HTTP 服务器
├── ui/
│   ├── aria-ng/                   # aria-ng Web UI
│   └── get-magnet-ui/             # 主管理界面 (Vue3)
├── proto/                         # Protobuf 定义
├── deploy/                        # 部署配置
├── docker/                        # Docker 配置
└── logs/                          # 日志目录
```

## 爬虫开发

实现 `crawler.Crawler` 接口即可添加新的爬虫：

```go
type Crawler interface {
    Name() string          // 唯一名称
    CronSpec() string      // 定时表达式（cron 格式）
    Run()                  // 执行任务
}
```

注册新爬虫：

```go
crawlerManager.Register(yourCrawler.NewYourCrawler())
```

## 数据库迁移

项目使用 xorm 自动迁移，迁移脚本位于 `internal/db/migrate/`。

## 开发规范

### 依赖注入

项目使用自定义的依赖注入容器（bean 包）：

```go
// 注册组件
bean.MustRegisterPtr[config.Config](ctx, config.Load())

// 获取依赖
cfg := bean.PtrFromContext[config.Config](ctx)
```

### 日志规范

使用 logrus 进行日志记录：

```go
import log "github.com/sirupsen/logrus"

log.Info("信息日志")
log.Error("错误日志")
```

## 依赖说明

- 项目 fork 并修改了 `siku2/arigo`，替换为 `github.com/nekoimi/arigo`
- ui/aria-ng 是 git submodule

## 许可证

[MIT](LICENSE)

## 作者

**nekoimi** - [nekoimime@gmail.com](mailto:nekoimime@gmail.com)

---

如有问题或建议，欢迎提交 Issue 或 PR。
