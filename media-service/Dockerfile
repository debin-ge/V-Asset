# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 安装依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o media-service cmd/main.go

# 运行阶段
FROM python:3.11-slim-bookworm

# 安装yt-dlp、ffmpeg和Node.js (yt-dlp解析YouTube需要JS运行时，impersonation 依赖 curl-cffi)
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata ffmpeg nodejs libjemalloc2 && \
    jemalloc_path="$(find /usr/lib -name libjemalloc.so.2 | head -n 1)" && \
    ln -sf "${jemalloc_path}" /usr/local/lib/libjemalloc.so.2 && \
    pip install --no-cache-dir "yt-dlp[curl-cffi]" && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/media-service .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations

# 创建运行目录
RUN mkdir -p /etc/youdlp/cookies /data/youdlp/tmp /data/youdlp/archive

# 设置时区
ENV TZ=Asia/Shanghai
ENV LD_PRELOAD=/usr/local/lib/libjemalloc.so.2

EXPOSE 9002

CMD ["./media-service"]
