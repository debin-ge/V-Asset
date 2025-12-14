package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"vasset/parser-service/internal/cache"
	"vasset/parser-service/internal/config"
	"vasset/parser-service/internal/handler"
	"vasset/parser-service/internal/service"
	pb "vasset/parser-service/proto"
)

func main() {
	// 1. 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// 2. 加载配置
	cfg, err := config.LoadConfig("config/dev.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	logger.Info("Starting Parser Service", zap.Int("port", cfg.Server.Port))

	// 3. 连接 Redis
	redisClient := initRedis(&cfg.Redis, logger)
	defer redisClient.Close()

	// 测试 Redis 连接
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Failed to connect to Redis, cache will be disabled", zap.Error(err))
	} else {
		logger.Info("✓ Connected to Redis")
	}

	// 4. 初始化缓存服务
	cacheService := cache.NewService(redisClient, cfg.Cache.GetCacheTTL())

	// 5. 初始化解析服务
	parserService := service.NewParserService(cfg, cacheService, logger)

	// 6. 初始化 gRPC 服务器
	grpcServer := handler.NewGRPCServer(parserService, logger)

	// 7. 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	s := grpc.NewServer()
	pb.RegisterParserServiceServer(s, grpcServer)

	// 8. 优雅关闭
	go func() {
		logger.Info("✓ gRPC server listening", zap.Int("port", cfg.Server.Port))
		if err := s.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	s.GracefulStop()
	logger.Info("Server stopped")
}

// initRedis 初始化 Redis 连接
func initRedis(cfg *config.RedisConfig, logger *zap.Logger) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	return client
}
