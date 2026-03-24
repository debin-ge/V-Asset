# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 复制 asset-service 完整源码（满足 replace 指令，auth-service 引用了 asset-service/proto）
COPY asset-service/ /asset-service/

# 复制 auth-service 依赖文件并下载
COPY auth-service/go.mod auth-service/go.sum ./
RUN go mod download

# 复制源代码
COPY auth-service/ .

# 更新依赖并编译
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth-service cmd/main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/auth-service .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 9001

CMD ["./auth-service"]
