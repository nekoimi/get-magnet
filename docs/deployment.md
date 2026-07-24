# 在线管理系统部署说明

## 必填环境变量

- `POSTGRES_PASSWORD`：PostgreSQL 密码。
- `JWT_SECRET`：生产环境强随机 JWT 密钥。
- `APP_EXTERNAL_BASE_URL`：外部可访问的应用根地址，例如 `https://magnet.example.com`，用于生成 STRM 播放地址。

网盘中间服务不由本仓库构建。通过 `CLOUD_DRIVER_BASE_URL` 指向已部署实例；若它与 Compose 位于同一网络，可使用 `http://cloud-driver:8080`。

## 管理端静态资源

根目录 `Dockerfile` 使用独立 Node 构建阶段执行 `pnpm build`，并将产物复制到镜像内的 `/workspace/ui`。Go 后端在 `/` 提供这些静态资源，因此线上无需单独部署前端服务。

镜像内保留 BusyBox `wget`，容器健康检查请求 `http://127.0.0.1:8093/healthz`。

根目录的 `docker-compose.example.yaml` 提供 PostgreSQL 与 get-magnet 示例，并分别使用 `pg_isready` 与 `/healthz` 进行健康检查。

## 日志查询安全边界

后续增加日志查询接口时必须遵守以下约束：

- 仅允许读取运行时 `log_dir` 解析后的目录内文件。
- 使用服务端枚举产生的文件标识，拒绝客户端传入任意绝对路径和 `..`。
- 只允许读取常规日志文件，拒绝符号链接。
- 限制单次读取行数、字节数与查询时间，不提供下载整个日志目录的能力。
- 对日志内容进行敏感信息过滤，并保留接口鉴权。
