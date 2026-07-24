FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /build
COPY . .
RUN go install cmd
RUN go build --ldflags "-extldflags -static" -o get-magnet cmd/main.go

FROM node:22-alpine AS ui-builder
WORKDIR /build/ui/get-magnet-ui
RUN corepack enable
COPY ui/get-magnet-ui/package.json ui/get-magnet-ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY ui/get-magnet-ui/ ./
RUN pnpm build

# FROM ghcr.io/nekoimi/get-magnet-runtime:latest
FROM alpine:latest

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

COPY --from=builder /build/get-magnet   /usr/bin/get-magnet
COPY --from=ui-builder /build/ui/get-magnet-ui/dist /workspace/ui
COPY ui/aria-ng /workspace/ui/aria-ng

ENV LOG_PATH=/workspace/logs

RUN apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /workspace

# 添加用户
RUN addgroup -g 1000 appuser && \
    adduser -u 1000 -G appuser -s /bin/sh -D appuser && \
    chown -R appuser:appuser /workspace

# Run as non-privileged
USER appuser

VOLUME /workspace/logs

EXPOSE 8093

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8093/healthz || exit 1

ENTRYPOINT ["get-magnet"]
