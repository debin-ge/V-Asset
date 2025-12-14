package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"vasset/api-gateway/internal/client"
	"vasset/api-gateway/internal/config"
	"vasset/api-gateway/internal/mq"
	"vasset/api-gateway/internal/router"
	"vasset/api-gateway/internal/ws"
)

func main() {
	log.Println("Starting API Gateway...")

	// 1. 加载配置
	cfg, err := config.LoadConfig("config/dev.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("✓ Configuration loaded (port: %d, mode: %s)", cfg.Server.Port, cfg.Server.Mode)

	// 2. 连接 Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		log.Printf("✓ Connected to Redis: %s", cfg.Redis.Addr)
	}

	// 3. 连接 gRPC 服务
	grpcClients, err := client.NewGRPCClients(&cfg.GRPC)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC services: %v", err)
	}
	defer grpcClients.Close()

	// 4. 连接 RabbitMQ
	mqPublisher, err := mq.NewPublisher(&cfg.RabbitMQ)
	if err != nil {
		log.Printf("Warning: Failed to connect to RabbitMQ: %v", err)
		// 不退出，允许降级运行
	} else {
		defer mqPublisher.Close()
	}

	// 5. 创建 WebSocket 管理器
	wsManager := ws.NewManager(redisClient)
	log.Println("✓ WebSocket manager initialized")

	// 6. 设置路由
	deps := &router.Dependencies{
		Config:      cfg,
		GRPCClients: grpcClients,
		RedisClient: redisClient,
		MQPublisher: mqPublisher,
		WSManager:   wsManager,
	}
	r := router.SetupRouter(deps)

	// 7. 创建 HTTP 服务器
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        r,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// 8. 启动服务器
	go func() {
		log.Printf("✓ HTTP server listening on :%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 9. 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// 10. 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
