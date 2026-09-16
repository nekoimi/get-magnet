# 在线管理系统部署说明

## 必填环境变量

- `POSTGRES_PASSWORD`：PostgreSQL 密码。
- `JWT_SECRET`：生产环境强随机 JWT 密钥。
- `APP_EXTERNAL_BASE_URL`：外部可访问的应用根地址，例如 `https://magnet.example.com`，用于生成 STRM 播放地址。

网盘中间服务不由本仓库构建。通过 `CLOUD_DRIVER_BASE_URL` 指向已部署实例；若它与 Compose 位于同一网络，可使用 `http://cloud-driver:8080`。

## 管理端静态资源

根目录 `Dockerfile` 生成一个同时包含 Go 后端和管理端的镜像：

1. `ui-builder` 使用 Node 22 和 pnpm 构建 `ui/get-magnet-ui`。
2. `ariang-builder` 使用 Node 22 和 npm 构建 `ui/aria-ng`。
3. `go-builder` 使用 Go 1.25 构建静态后端二进制，并注入版本与 Git commit。
4. 最终 Alpine 镜像将管理端产物复制到 `/workspace/ui`，将 AriaNg 产物复制到 `/workspace/ui/aria-ng`。
5. Go 后端在 `/` 提供管理端静态资源，在 `/ui/aria-ng/` 提供 AriaNg；后台“下载管理 / Aria2 控制台”通过 iframe 加载该地址，因此线上无需单独部署前端服务。

生产构建固定使用同源地址：

- `VITE_PUBLIC_PATH=/`
- `VITE_API_URL=/`

镜像内保留 BusyBox `wget`，容器健康检查请求 `http://127.0.0.1:8093/healthz`。

根目录的 `docker-compose.example.yaml` 提供 PostgreSQL 与 get-magnet 示例，并分别使用 `pg_isready` 与 `/healthz` 进行健康检查。

## GitHub Actions 镜像构建

`.github/workflows/cr-image.yml` 在以下场景运行：

- 推送到 `main`、`master` 或 `feature/**` 分支。
- 推送 `v*` 标签。
- Pull Request 和手动触发。

工作流 checkout 时使用 `submodules: recursive`，这是构建管理端和 AriaNg 的必要条件。测试任务先执行全量 Go 测试和前端生产构建；镜像任务随后使用 Buildx 构建 `linux/amd64`、`linux/arm64`，非 Pull Request 构建发布至：

```text
ghcr.io/<owner>/<repository>
```

标签构建会生成语义化版本标签和 `latest`，分支构建会生成分支标签及 `sha-*` 标签。

## 日志查询安全边界

后续增加日志查询接口时必须遵守以下约束：

- 仅允许读取运行时 `log_dir` 解析后的目录内文件。
- 使用服务端枚举产生的文件标识，拒绝客户端传入任意绝对路径和 `..`。
- 只允许读取常规日志文件，拒绝符号链接。
- 限制单次读取行数、字节数与查询时间，不提供下载整个日志目录的能力。
- 对日志内容进行敏感信息过滤，并保留接口鉴权。
