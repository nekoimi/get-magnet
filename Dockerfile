# syntax=docker/dockerfile:1.7

FROM node:22-alpine AS ui-builder

WORKDIR /build/ui/get-magnet-ui
RUN corepack enable

COPY ui/get-magnet-ui/package.json ui/get-magnet-ui/pnpm-lock.yaml ./
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

COPY ui/get-magnet-ui/ ./
ARG VITE_PUBLIC_PATH=/
ARG VITE_API_URL=/
ENV VITE_PUBLIC_PATH=${VITE_PUBLIC_PATH}
ENV VITE_API_URL=${VITE_API_URL}
RUN pnpm build

FROM golang:1.25-alpine AS go-builder

ENV CGO_ENABLED=0
WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath \
      -ldflags="-s -w -extldflags=-static -X github.com/nekoimi/get-magnet/internal/api/ops.BuildVersion=${VERSION} -X github.com/nekoimi/get-magnet/internal/api/ops.BuildCommit=${COMMIT}" \
      -o /out/get-magnet ./cmd/main.go

FROM alpine:3.22

LABEL org.opencontainers.image.title="get-magnet" \
      org.opencontainers.image.description="Magnet crawler and download management system" \
      org.opencontainers.image.source="https://github.com/nekoimi/get-magnet"

RUN apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && addgroup -g 1000 appuser \
    && adduser -u 1000 -G appuser -s /bin/sh -D appuser \
    && mkdir -p /workspace/logs /workspace/ui \
    && chown -R appuser:appuser /workspace

COPY --from=go-builder /out/get-magnet /usr/bin/get-magnet
COPY --from=ui-builder /build/ui/get-magnet-ui/dist/ /workspace/ui/
COPY ui/aria-ng/ /workspace/ui/aria-ng/

ENV TZ=Asia/Shanghai \
    LOG_DIR=/workspace/logs

WORKDIR /workspace
USER appuser

VOLUME ["/workspace/logs"]
EXPOSE 8093

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8093/healthz || exit 1

ENTRYPOINT ["/usr/bin/get-magnet"]
