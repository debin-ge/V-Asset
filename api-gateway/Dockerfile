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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway cmd/main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/api-gateway .
COPY --from=builder /app/config ./config

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 8080

CMD ["./api-gateway"]
