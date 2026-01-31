.PHONY: proto build run clean docker-build test

# 生成 protobuf 代码
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/proxy.proto

# 编译
build:
	go build -o bin/proxy-service cmd/main.go

# 运行
run:
	go run cmd/main.go

# 清理
clean:
	rm -rf bin/
	rm -f proto/*.pb.go

# Docker 构建
docker-build:
	docker build -t vasset/proxy-service:latest .

# 测试
test:
	go test -v ./...

# 安装依赖
deps:
	go mod download
	go mod tidy
