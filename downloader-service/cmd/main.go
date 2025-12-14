package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"vasset/downloader-service/internal/cleanup"
	"vasset/downloader-service/internal/config"
	"vasset/downloader-service/internal/database"
	"vasset/downloader-service/internal/handler"
	"vasset/downloader-service/internal/proxy"
	"vasset/downloader-service/internal/repository"
	"vasset/downloader-service/internal/service"
	"vasset/downloader-service/internal/storage"
	"vasset/downloader-service/internal/worker"
	"vasset/downloader-service/internal/ytdlp"
	pb "vasset/downloader-service/proto"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("config/dev.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Downloader Service on port %d...", cfg.Server.Port)

	log.Printf("Starting Downloader Service on port %d...", cfg.Server.Port)

	// 2. 运行数据库迁移
	if err := database.RunMigrations(cfg.Database.GetURL()); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// 3. 连接 PostgreSQL
	db, err := initDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Connected to PostgreSQL")

	// 3. 连接 Redis
	redisClient := initRedis(&cfg.Redis)
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		log.Println("✓ Connected to Redis")
	}

	// 4. 初始化 Repository
	downloadRepo := repository.NewDownloadRepository(db)

	// 5. 初始化代理提供者
	var proxyProvider worker.ProxyProviderInterface
	if cfg.Proxy.APIKey != "" && cfg.Proxy.APIKey != "your-api-key" {
		proxyProvider = proxy.NewProvider(&cfg.Proxy)
		log.Println("✓ Using proxy provider")
	} else {
		proxyProvider = proxy.NewMockProvider()
		log.Println("✓ Using direct connection (no proxy)")
	}

	// 6. 初始化 yt-dlp 执行器
	executor := ytdlp.NewExecutor(&cfg.YtDLP)

	// 7. 初始化存储管理
	pathGenerator := storage.NewPathGenerator(&cfg.Storage)
	fileManager := storage.NewFileManager(cfg.Storage.BasePath)

	// 确保存储目录存在
	if err := fileManager.EnsureDir(cfg.Storage.BasePath); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// 8. 初始化进度发布器
	progressPublisher := worker.NewProgressPublisher(redisClient)

	// 9. 初始化 Worker 池
	workerPool := worker.NewPool(
		&cfg.Worker,
		&cfg.Storage,
		&cfg.Retry,
		downloadRepo,
		proxyProvider,
		executor,
		pathGenerator,
		fileManager,
		progressPublisher,
	)

	// 10. 启动 Worker 池
	workerPool.Start()
	log.Printf("✓ Worker pool started with %d workers", cfg.Worker.PoolSize)

	// 11. 初始化并启动任务消费者
	taskConsumer, err := worker.NewTaskConsumer(&cfg.RabbitMQ, workerPool)
	if err != nil {
		log.Printf("Warning: Failed to connect to RabbitMQ: %v", err)
		log.Println("  Service will run without MQ consumer")
	} else {
		go func() {
			if err := taskConsumer.Start(ctx); err != nil {
				log.Printf("Task consumer stopped: %v", err)
			}
		}()
		log.Println("✓ Task consumer started")
	}

	// 12. 启动清理调度器
	cleanupScheduler := cleanup.NewScheduler(&cfg.Cleanup, downloadRepo, fileManager)
	go cleanupScheduler.Start(ctx)
	log.Println("✓ Cleanup scheduler started")

	// 13. 初始化 Service 和 Handler
	downloaderService := service.NewDownloaderService(downloadRepo)
	grpcHandler := handler.NewGRPCHandler(downloaderService)

	// 14. 启动 gRPC 服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDownloaderServiceServer(grpcServer, grpcHandler)

	go func() {
		log.Printf("✓ gRPC server listening on :%d", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 15. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// 停止 gRPC 服务器
	grpcServer.GracefulStop()

	// 停止任务消费者
	if taskConsumer != nil {
		taskConsumer.Stop()
	}

	// 停止 Worker 池
	workerPool.Stop()

	log.Println("Server stopped")
}

// initDatabase 初始化数据库连接
func initDatabase(cfg *config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		return nil, err
	}

	// 配置连接池
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// initRedis 初始化 Redis 连接
func initRedis(cfg *config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}
