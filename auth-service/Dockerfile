# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

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
