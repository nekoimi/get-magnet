# Changelog

所有项目的显著变更都将记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### Added
- 添加 CORS 中间件解决跨域问题
- 实现磁力链接管理 CRUD 功能
- 添加 AGENTS.md 项目上下文文档
- 扩展 page 页面提交接口
- 添加关键词过滤规则

### Fixed
- 修复文件重复移动的 bug
- 新增定时扫描已经完成的任务
- 修复链接不符合条件的情况

## [1.0.0] - 2024-03

### Added
- 添加 DrissionRod gRPC 客户端集成，支持浏览器自动化
- 实现依赖注入容器（bean 包），优化组件生命周期管理
- 添加 aria2 下载器集成和管理
- 支持从 javdb、sehuatang 等站点自动采集磁力链接
- 实现 JWT 用户认证机制
- 添加定时任务调度器（robfig/cron）
- 集成 PostgreSQL 数据库（xorm ORM）
- 添加日志文件分级别分割轮转（logrus + lumberjack）
- 实现文件自动移动功能（下载完成后自动移动到指定目录）
- 添加下载速度监控和统计
- 集成 tracker 服务器自动更新
- 添加文件优选删除机制
- 实现任务关联功能（followedBy）
- 添加 OCR 识别机制
- 实现 Cloudflare 人机验证绕过
- 添加文件删除功能
- 支持配置热加载（viper）
- 添加 Vue3 + Element Plus 前端管理界面
- 集成 aria-ng Web UI

### Changed
- 更新 Go 版本至 1.24
- 重构 API 模块，按功能拆分避免函数名冲突
- 优化依赖注入方式，添加 registry + context 依赖解耦
- 优化 crawler 注册机制
- 优化 job 执行机制
- 优化文件移动路径处理
- 优化日志输出格式
- 优化 aria2 客户端结构
- 优化任务处理时机和机制
- 优化文件下载选择策略
- 优化下载任务检测状态
- 优化 worker goroutine 性能

### Fixed
- 修复 alpine 下命令错误
- 修复权限问题
- 修复文件名过长导致的 bug
- 修复下载完成文件不删除的问题
- 修复 crawler 注册重名覆盖问题
- 修复配置名称错误的 bug
- 修复文件移动兼容性问题
- 修复 URL 路径错误的问题
- 修复 xorm column name 错误
- 修复文件选择限制问题
- 修复下载速度统计更新不及时的问题
- 修复 rod 读取 cookies 解析错误
- 修复查询路径错误
- 修复 glibc 支持问题
- 修复下载任务文件名显示
- 修复任务提交方式
- 修改 Dockerfile 时区配置

### Removed
- 移除本地浏览器操作，改用 DrissionRod gRPC
- 移除基础容器镜像依赖
- 移除 flaresolverr 依赖
- 移除 Python 扩展代码

## [0.9.0] - 2024-02

### Added
- 初始化项目架构
- 添加 gorilla/mux 路由框架
- 实现基础 HTTP API 接口
- 添加数据库连接和迁移
- 实现基础爬虫框架
- 添加 Docker 支持
- 添加 CI/CD 工作流
- 添加 Makefile 构建脚本

### Changed
- 优化项目结构
- 调整模块包名
- 更新依赖库版本

### Fixed
- 修复 Docker 构建问题
- 修复时区配置
- 修复构建日志输出

## [0.1.0] - 2024-01

### Added
- 项目初始版本
- 基础磁力链接下载功能
- 简单 Web 界面

---

## 提交类型说明

- **Added**: 新增功能
- **Changed**: 变更现有功能
- **Deprecated**: 标记即将废弃的功能
- **Removed**: 移除功能
- **Fixed**: 修复 bug
- **Security**: 安全相关修复

## 标签说明

- `feat`: 新功能
- `fix`: 修复
- `docs`: 文档
- `style`: 格式（不影响代码运行的变动）
- `refactor`: 重构
- `perf`: 性能优化
- `test`: 测试
- `chore`: 构建过程或辅助工具的变动
