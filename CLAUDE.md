# scrapio 开发上下文

- Go module：`github.com/nekoimi/scrapio`，入口 `cmd/main.go`。
- 前端：`web/`，Vue 3 + Vite；生产镜像从此目录构建。
- 浏览器服务：独立项目 `scrapio-browser`，通过 gRPC 连接。
- 数据库：PostgreSQL。
- v2.0 文档是历史技术改造记录；v2.1 文档描述后续产品方向。

```bash
go run ./cmd/main.go
go test ./...
go build -o scrapio ./cmd/main.go
cd web && pnpm install --frozen-lockfile && pnpm build
```

部署与环境变量参见 `README.md` 和 `docs/项目文档v1.0/deployment.md`。浏览器连接的环境变量仍使用兼容名称 `CRAWLER_DRISSION_ROD_GRPC_IP/PORT`。
