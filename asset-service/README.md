# Asset Service - 资产管理服务

Asset Service 是 V-Asset 视频下载平台的资产管理微服务，负责用户下载历史管理、配额控制和数据统计。

## 功能特性

- **下载历史管理**: 查询、删除历史记录，支持分页、过滤、排序
- **配额管理**: 用户每日下载配额的检查和消费
- **数据统计**: 多维度用户统计，包括平台分布、日活动等
- **文件信息**: 文件权限验证和元数据获取

## 技术栈

- Go 1.24
- gRPC
- PostgreSQL

## 快速开始

### 1. 生成 Protobuf 代码

```bash
make proto
```

### 2. 编译

```bash
make build
```

### 3. 运行

```bash
make run
```

## API 接口

| 方法 | 描述 |
|:---|:---|
| `GetHistory` | 获取下载历史(分页/过滤/排序) |
| `DeleteHistory` | 删除历史记录 |
| `CheckQuota` | 检查用户配额 |
| `ConsumeQuota` | 消费配额 |
| `GetUserStats` | 获取用户统计 |
| `GetFileInfo` | 获取文件信息 |

## 配置

配置文件位于 `config/dev.yaml`，主要配置项：

- `server.port`: 服务端口(默认9004)
- `database`: PostgreSQL连接配置
- `quota`: 配额策略配置
- `pagination`: 分页配置

## 目录结构

```
asset-service/
├── cmd/main.go           # 应用入口
├── config/               # 配置文件
├── internal/
│   ├── config/           # 配置加载
│   ├── handler/          # gRPC处理器
│   ├── models/           # 数据模型
│   ├── repository/       # 数据访问层
│   └── service/          # 业务逻辑层
├── migrations/           # 数据库迁移
└── proto/                # Protobuf定义
```

## Docker

```bash
# 构建镜像
make docker-build

# 启动容器
make docker-up
```
