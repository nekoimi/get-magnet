FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /build
COPY . .
RUN go install cmd
RUN go build --ldflags "-extldflags -static" -o get-magnet cmd/main.go

# FROM ghcr.io/nekoimi/get-magnet-runtime:latest
FROM alpine:latest

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

COPY --from=builder /build/get-magnet   /usr/bin/get-magnet

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

ENTRYPOINT ["get-magnet"]
