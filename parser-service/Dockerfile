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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o parser-service cmd/main.go

# 运行阶段
FROM python:3.11-alpine

# 安装yt-dlp、ffmpeg和Node.js (yt-dlp解析YouTube需要JS运行时)
RUN apk --no-cache add ca-certificates tzdata ffmpeg nodejs && \
    pip install --no-cache-dir yt-dlp

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/parser-service .
COPY --from=builder /app/config ./config

# 创建Cookie目录
RUN mkdir -p /etc/vasset/cookies

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 9002

CMD ["./parser-service"]
