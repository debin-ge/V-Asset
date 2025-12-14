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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o downloader-service cmd/main.go

# 运行阶段
FROM ubuntu:22.04

# 配置环境变量
ENV DEBIAN_FRONTEND=noninteractive \
    TZ=Asia/Shanghai

# 使用清华镜像源加速（针对 ARM64）
RUN sed -i 's|ports.ubuntu.com|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list && \
    sed -i 's|archive.ubuntu.com|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list

# 安装 yt-dlp 和 ffmpeg
RUN apt-get update && apt-get install -y \
    curl \
    ffmpeg \
    ca-certificates \
    tzdata \
    python3 \
    && curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp \
    && chmod +x /usr/local/bin/yt-dlp \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/downloader-service .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations

# 创建数据目录
RUN mkdir -p /data/vasset/tmp /data/vasset/archive

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 9003

CMD ["./downloader-service"]
