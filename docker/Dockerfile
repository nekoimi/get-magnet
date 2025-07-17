# Base image of runtime
FROM debian:bookworm-slim

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# 安装依赖：Chromium、字体、中文字体、时区
# rod support version: Chromium 128.0.6568.0
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl \
        unzip \
        chromium \
        fonts-freefont-ttf \
        fonts-noto-color-emoji \
        fonts-wqy-zenhei \
        tzdata \
        ca-certificates \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
        dpkg-reconfigure -f noninteractive tzdata && \
        update-ca-certificates && \
        apt-get clean && \
        rm -rf /var/lib/apt/lists/*

COPY local.conf /etc/fonts/local.conf

# 创建工作目录并添加用户
RUN mkdir -p /workspace && mkdir -p /var/lib/rod-data && \
    groupadd -g 1000 appuser && \
    useradd -u 1000 -g 1000 -ms /bin/bash appuser && \
    chown -R appuser:appuser /workspace && \
    chown -R appuser:appuser /var/lib/rod-data

# Autorun chrome headless
ENV CHROMIUM_FLAGS="--disable-software-rasterizer --disable-dev-shm-usage"
ENV CHROME_BIN=/usr/bin/chromium

# 设置 Chromium 启动路径给 Rod 用
ENV ROD_BROWSER_PATH=/usr/bin/chromium
ENV ROD_DATA_DIR=/var/lib/rod-data

# 安装OCR服务
RUN curl -L -o /tmp/release.zip https://github.com/86maid/ddddocr/releases/download/v4.0.1/x86_64-unknown-linux-gnu-inline.zip \
    && unzip /tmp/release.zip -d /tmp/ocr \
    && ls -l /tmp && ls -l /tmp/ocr \
    && mv /tmp/ocr/ddddocr /usr/bin/ocr \
    && chmod +x /usr/bin/ocr \
        \
    && rm -rf /tmp/*

# 设置OCR服务启动路径
ENV OCR_BIN_PATH=/usr/bin/ocr

# Run as non-privileged
USER appuser

WORKDIR /workspace

VOLUME /var/lib/rod-data

ENTRYPOINT ["chromium", "--headless"]
