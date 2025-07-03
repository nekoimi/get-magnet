FROM golang:1.22-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /build
COPY . .
RUN go install cmd
RUN go build --ldflags "-extldflags -static" -o get-magnet cmd/main.go

FROM alpine:3.20

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

# Installs latest Chromium package.
# # rod support version: Chromium 128.0.6568.0
RUN apk upgrade --no-cache --available \
    && apk add --no-cache \
      chromium-swiftshader \
      ttf-freefont \
      font-noto-emoji \
    && apk add --no-cache \
      --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community \
      font-wqy-zenhei \
    && apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
# Add Chrome as a user
RUN mkdir -p /usr/src/app \
    && adduser -D chrome \
    && chown -R chrome:chrome /usr/src/app

COPY local.conf /etc/fonts/local.conf

# Autorun chrome headless
ENV CHROMIUM_FLAGS="--disable-software-rasterizer --disable-dev-shm-usage"
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/lib/chromium/

# 设置 Chromium 启动路径给 Rod 用
ENV ROD_BROWSER_PATH=/usr/bin/chromium-browser

COPY --from=builder /build/get-magnet   /usr/bin/get-magnet

WORKDIR /workspace

EXPOSE 8093

ENTRYPOINT ["get-magnet"]
