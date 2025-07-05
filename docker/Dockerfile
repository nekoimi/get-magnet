# Base image of runtime
FROM alpine:3.20

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

ARG OCR_DOWNLOAD_URL=https://github.com/86maid/ddddocr/releases/download/v4.0.1/x86_64-unknown-linux-gnu-inline.zip

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
ENV ROD_DATA_DIR=/var/lib/rod-data

# 安装OCR服务
RUN apk add --no-cache curl \
    && curl -L -o /tmp/release.zip $OCR_DOWNLOAD_URL \
    && unzip /tmp/release.zip -d /tmp/ocr \
    && ls -l /tmp && ls -l /tmp/ocr \
    && mv /tmp/ocr/ddddocr /usr/bin/ocr \
    && chmod +x /usr/bin/ocr

# 设置OCR服务启动路径
ENV OCR_BIN_PATH=/usr/bin/ocr

# Autorun chrome headless
ENV CHROMIUM_FLAGS="--disable-software-rasterizer --disable-dev-shm-usage"

VOLUME /var/lib/rod-data

ENTRYPOINT ["chromium-browser", "--headless"]
