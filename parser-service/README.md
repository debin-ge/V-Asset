# Parser Service

Parser Service 是 V-Asset 视频资产管理平台的元数据解析服务,负责解析视频URL并提取视频元数据信息。

## 功能特性

- **多平台支持**: YouTube、TikTok、Bilibili、Twitter/X、Instagram 等1000+视频平台
- **智能缓存**: Redis缓存解析结果,提高响应速度
- **并发控制**: 限制并发解析数量,防止资源耗尽
- **格式标准化**: 自动标准化视频格式信息,提供质量标签
- **错误处理**: 完善的错误分类和处理机制

## 技术栈

- **语言**: Go 1.21
- **协议**: gRPC
- **缓存**: Redis
- **解析工具**: yt-dlp

## 快速开始

### 前置要求

- Go 1.21+
- Redis
- yt-dlp (本地运行时需要)
- protoc (如需重新生成proto代码)

### 本地开发

1. **安装yt-dlp**
```bash
# macOS
brew install yt-dlp

# 或使用pip
pip install yt-dlp
```

2. **安装依赖**
```bash
make deps
```

3. **生成proto代码**
```bash
make proto
```

4. **启动Redis**
```bash
docker run -d -p 6379:6379 redis:7-alpine
```

5. **运行服务**
```bash
make run
```

### Docker部署

```bash
# 构建并启动服务
make docker-up

# 停止服务
make docker-down
```

## API接口

### ParseURL

解析视频URL并获取元数据。

**请求**:
```protobuf
message ParseURLRequest {
  string url = 1;
  bool skip_cache = 2;  // 可选:跳过缓存
}
```

**响应**:
```protobuf
message ParseURLResponse {
  string video_id = 1;
  string platform = 2;
  string title = 3;
  string description = 4;
  int64 duration = 5;
  string thumbnail = 6;
  string author = 7;
  string upload_date = 8;
  int64 view_count = 9;
  repeated VideoFormat formats = 10;
}
```

### ValidateURL

验证URL是否有效。

**请求**:
```protobuf
message ValidateURLRequest {
  string url = 1;
}
```

**响应**:
```protobuf
message ValidateURLResponse {
  bool valid = 1;
  string platform = 2;
  string message = 3;
}
```

## 配置说明

配置文件位于 `config/dev.yaml`:

```yaml
server:
  port: 9002              # gRPC端口

redis:
  addr: localhost:6379    # Redis地址
  
ytdlp:
  timeout: 30             # 解析超时(秒)
  max_concurrent: 10      # 最大并发数
  
cache:
  ttl: 3600              # 缓存TTL(秒)
```

## 使用示例

使用grpcurl测试:

```bash
# 解析YouTube视频
grpcurl -plaintext -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}' \
  localhost:9002 parser.ParserService/ParseURL

# 验证URL
grpcurl -plaintext -d '{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}' \
  localhost:9002 parser.ParserService/ValidateURL
```

## 项目结构

```
parser-service/
├── cmd/
│   └── main.go                 # 服务入口
├── internal/
│   ├── adapter/                # 平台适配器
│   ├── cache/                  # 缓存服务
│   ├── config/                 # 配置管理
│   ├── detector/               # 平台检测
│   ├── handler/                # gRPC处理器
│   ├── service/                # 业务逻辑
│   ├── utils/                  # 工具函数
│   └── ytdlp/                  # yt-dlp封装
├── proto/                      # Proto定义
├── config/                     # 配置文件
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 支持的平台

- YouTube (youtube.com, youtu.be)
- TikTok (tiktok.com)
- Bilibili (bilibili.com)
- Twitter/X (twitter.com, x.com)
- Instagram (instagram.com)
- 更多通过yt-dlp支持的平台

## Cookie配置

某些平台(如Bilibili)可能需要Cookie才能访问完整信息:

1. 导出浏览器Cookie到文件
2. 将Cookie文件放到 `/etc/vasset/cookies/` 目录
3. 在配置文件中指定Cookie文件路径

## 监控指标

服务提供以下监控指标:

- 解析请求总数(按平台分类)
- 解析成功/失败数
- 缓存命中/未命中数
- 解析耗时(P50/P95/P99)

## 故障排查

### yt-dlp未找到
```bash
# 确认yt-dlp已安装
which yt-dlp

# 或在配置文件中指定完整路径
ytdlp:
  binary_path: "/usr/local/bin/yt-dlp"
```

### Redis连接失败
服务会在Redis不可用时跳过缓存,但仍能正常工作。

### 解析超时
增加timeout配置值或检查网络连接。

## 许可证

MIT
