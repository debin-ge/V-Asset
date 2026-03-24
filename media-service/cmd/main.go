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
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"youdlp/media-service/internal/cache"
	"youdlp/media-service/internal/config"
	dlcleanup "youdlp/media-service/internal/download/cleanup"
	dlclient "youdlp/media-service/internal/download/client"
	dlconfig "youdlp/media-service/internal/download/config"
	dldatabase "youdlp/media-service/internal/download/database"
	dlrepo "youdlp/media-service/internal/download/repository"
	dlscheduler "youdlp/media-service/internal/download/scheduler"
	dlstorage "youdlp/media-service/internal/download/storage"
	dlworker "youdlp/media-service/internal/download/worker"
	dlytdlp "youdlp/media-service/internal/download/ytdlp"
	"youdlp/media-service/internal/handler"
	"youdlp/media-service/internal/observability"
	"youdlp/media-service/internal/service"
	pb "youdlp/media-service/proto"
)

func main() {
	// 1. 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// 2. 加载配置（解析配置 + 下载配置）
	parseCfg, err := config.LoadConfig("config/dev.yaml")
	if err != nil {
		logger.Fatal("Failed to load parser config", zap.Error(err))
	}
	downloadCfg, err := dlconfig.LoadConfig("config/dev.yaml")
	if err != nil {
		logger.Fatal("Failed to load downloader config", zap.Error(err))
	}

	logger.Info("Starting Media Service", zap.Int("port", parseCfg.Server.Port))

	// 3. 连接 Redis
	redisClient := initRedis(&parseCfg.Redis)
	defer redisClient.Close()
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()
	if err := redisClient.Ping(appCtx).Err(); err != nil {
		logger.Warn("Failed to connect to Redis", zap.Error(err))
	} else {
		logger.Info("✓ Connected to Redis")
	}

	// 4. 初始化 PostgreSQL 和下载子系统
	if err := dldatabase.RunMigrations(downloadCfg.Database.GetURL()); err != nil {
		logger.Fatal("Failed to run database migrations", zap.Error(err))
	}
	db, err := initDatabase(&downloadCfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	downloadRepo := dlrepo.NewDownloadRepository(db)
	executor := dlytdlp.NewExecutor(&downloadCfg.YtDLP)
	pathGenerator := dlstorage.NewPathGenerator(&downloadCfg.Storage)
	fileManager := dlstorage.NewFileManager(downloadCfg.Storage.BasePath)
	if err := fileManager.EnsureDir(downloadCfg.Storage.BasePath); err != nil {
		logger.Fatal("Failed to create storage directory", zap.Error(err))
	}
	progressPublisher := dlworker.NewProgressPublisher(redisClient)

	var assetClient *dlclient.AssetClient
	if downloadCfg.AssetService.Addr != "" {
		timeout := time.Duration(downloadCfg.AssetService.Timeout) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		ac, err := dlclient.NewAssetClient(downloadCfg.AssetService.Addr, timeout, downloadCfg.AssetService.CookieTempDir)
		if err != nil {
			log.Printf("Warning: Failed to connect to Asset Service: %v", err)
		} else {
			assetClient = ac
			defer assetClient.Close()
		}
	}

	workerPool := dlworker.NewPool(
		&downloadCfg.Worker,
		&downloadCfg.Storage,
		&downloadCfg.Retry,
		downloadRepo,
		executor,
		pathGenerator,
		fileManager,
		progressPublisher,
		assetClient,
		downloadCfg.YtDLP.YouTube,
	)
	workerPool.Start()

	var taskConsumer *dlworker.TaskConsumer
	taskConsumer, err = dlworker.NewTaskConsumer(&downloadCfg.RabbitMQ, workerPool)
	if err != nil {
		log.Printf("Warning: Failed to connect to RabbitMQ: %v", err)
	} else {
		go func() {
			if err := taskConsumer.Start(appCtx); err != nil && err != context.Canceled {
				log.Printf("[TaskConsumer] stopped: %v", err)
			}
		}()
	}

	cleanupScheduler := dlcleanup.NewScheduler(&downloadCfg.Cleanup, downloadRepo, fileManager)
	go cleanupScheduler.Start(appCtx)

	ytDLPUpdater := dlscheduler.NewYtDLPUpdater(&downloadCfg.YtDLP, &downloadCfg.YtDLPUpdate)
	go ytDLPUpdater.Start(appCtx)

	// 5. 初始化解析服务
	cacheService := cache.NewService(redisClient, parseCfg.Cache.GetCacheTTL())
	parserService := service.NewParserService(parseCfg, cacheService, logger)
	grpcHandler := handler.NewGRPCServer(parserService, logger)

	// 6. 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", parseCfg.Server.Port))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(observability.UnaryServerInterceptor("media-service")),
	)
	pb.RegisterMediaServiceServer(grpcServer, grpcHandler)

	go func() {
		logger.Info("✓ gRPC server listening", zap.Int("port", parseCfg.Server.Port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// 7. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	appCancel()
	grpcServer.GracefulStop()
	if taskConsumer != nil {
		taskConsumer.Stop()
	}
	workerPool.Stop()
	logger.Info("Server stopped")
}

func initDatabase(cfg *dlconfig.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func initRedis(cfg *config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}
