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

	"vasset/proxy-service/internal/client"
	"vasset/proxy-service/internal/config"
	"vasset/proxy-service/internal/handler"
	"vasset/proxy-service/internal/health"
	pb "vasset/proxy-service/proto"
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

	logger.Info("Starting Proxy Service", zap.Int("port", cfg.Server.Port))

	// 3. 连接 Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Failed to connect to Redis, cache will be disabled", zap.Error(err))
	} else {
		logger.Info("✓ Connected to Redis")
	}

	// 4. 初始化 yt-dlp API 客户端
	ytdlpClient := client.NewYtDLPClient(&cfg.YtDLPAPI, logger)
	logger.Info("✓ YtDLP API client initialized", zap.String("base_url", cfg.YtDLPAPI.BaseURL))

	// 5. 启动健康检查器（如果启用）
	var healthChecker *health.Checker
	if cfg.HealthCheck.Enabled {
		healthChecker = health.NewChecker(&cfg.YtDLPAPI, &cfg.HealthCheck, logger)
		healthChecker.Start()
		defer healthChecker.Stop()
	} else {
		logger.Info("Health checker is disabled")
	}

	// 6. 初始化 gRPC Handler
	grpcHandler := handler.NewGRPCHandler(
		ytdlpClient,
		redisClient,
		cfg.Cache.GetCacheTTL(),
		logger,
	)

	// 7. 启动 gRPC 服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	pb.RegisterProxyServiceServer(grpcServer, grpcHandler)

	go func() {
		logger.Info("✓ gRPC server listening", zap.Int("port", cfg.Server.Port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// 8. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	grpcServer.GracefulStop()
	logger.Info("Server stopped")
}
