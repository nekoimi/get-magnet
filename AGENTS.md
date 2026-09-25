# scrapio 仓库说明

本仓库是 scrapio 主项目，Go module 为 `github.com/nekoimi/scrapio`。浏览器执行服务是独立仓库 `scrapio-browser`，前端管理界面位于 `web/`。PostgreSQL 保存任务与结果；旧磁力采集与下载插件仍保留兼容能力。v2.0 文档记录历史改造，v2.1 文档是后续产品规划。

## 常用命令

```bash
go test ./...
go run ./cmd/main.go
cd web && pnpm install --frozen-lockfile && pnpm build
docker compose -f docker-compose.example.yaml config
```

根目录 `Dockerfile` 构建 Go 服务和 `web/` 管理界面。不再构建 AriaNg 静态页面。部署参考 `README.md` 与 `docs/项目文档v1.0/deployment.md`。

当前浏览器连接配置键仍为 `CRAWLER_DRISSION_ROD_GRPC_IP` 和 `CRAWLER_DRISSION_ROD_GRPC_PORT`；它们对应 scrapio-browser，改名时需考虑现有部署兼容性。旧代码中的下载任务幂等前缀也属于持久化兼容数据。
