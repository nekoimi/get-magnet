# scrapio 部署说明

## 组件与镜像

基础部署包含 PostgreSQL、scrapio 和 scrapio-browser。浏览器服务通过 CloakBrowser Manager 获取远程浏览器；下载类插件依赖按需另行部署。

根目录 `Dockerfile` 使用 Node 22 / pnpm 从 `web/` 构建管理界面，再用 Go 1.26 构建 `scrapio`，最终镜像将前端产物放在 `/workspace/ui`。Go 服务在 `/` 提供管理界面，同时提供 `/healthz`。前端构建参数 `VITE_PUBLIC_PATH` 和 `VITE_API_URL` 在镜像工作流中均为 `/`。已删除的 AriaNg 页面及其构建步骤不再参与部署。

`scrapio-browser` 仓库目前通过手动工作流发布 `ghcr.io/nekoimi/scrapio-browser:test`。根目录示例 Compose 默认使用此标签；实际发布时可以通过 `SCRAPIO_BROWSER_IMAGE` 选择自己验证过的镜像。浏览器镜像内提供 `grpc_health_probe`，Compose 等待其健康后启动主服务。

## 使用 Compose

复制根目录的 `docker-compose.example.yaml` 为 `docker-compose.yaml`，在 `.env` 中至少配置：

```dotenv
POSTGRES_PASSWORD=replace-with-a-strong-password
JWT_SECRET=replace-with-a-long-random-secret
CLOAK_MANAGER_URL=https://your-cloak-manager.example
CLOAK_PROFILE_ID=your-profile-id
```

可选项：`CLOAK_AUTH_TOKEN`、`SCRAPIO_BROWSER_IMAGE`、`APP_PORT`、`POSTGRES_DB`、`POSTGRES_USER`。启动：

```bash
docker compose up -d
```

浏览器服务必须能访问所填的 CloakBrowser Manager。主服务通过现有兼容配置键 `CRAWLER_DRISSION_ROD_GRPC_IP/PORT` 连接 `scrapio-browser:8191`；这些键名目前是 Go 配置契约，不因产品改名而改动。

对于已有 PostgreSQL volume，保持原来的数据库名、用户、密码和卷名。Compose 中 `scrapio` 是新安装默认值；PostgreSQL 初始化变量不会给已有数据卷重命名或迁移数据。升级前备份数据库并核对 `DB_DSN`。

## GitHub Actions

`.github/workflows/cr-image.yml` 在 `feature/dev` 分支、`v*` 标签、Pull Request 或手动触发。它先运行 Go 测试与 `web/` 生产构建，再构建多架构镜像 `ghcr.io/<owner>/<repository>`。标签发布生成版本号与 `latest`，分支发布生成分支和提交 SHA 标签。手动的 `test.yml` 构建测试标签。前端已经是仓库内目录，无需递归 checkout 子模块。

## 运行配置

- `DB_DSN`：PostgreSQL 连接串。
- `JWT_SECRET`：生产环境强随机密钥。
- `APP_EXTERNAL_BASE_URL`：需要生成可从外部访问的 STRM 播放 URL 时设置。
- `QUICK_API_TOKEN`：启用对应快速接口时设置。
- `CRAWLER_DRISSION_ROD_GRPC_IP/PORT`：scrapio-browser 地址，Compose 已设置。

本地配置样例位于 `config/dev.yaml.example` 和 `config/prod.yaml.example`。
