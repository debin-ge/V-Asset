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
	"google.golang.org/grpc"

	"vasset/auth-service/internal/config"
	"vasset/auth-service/internal/database"
	"vasset/auth-service/internal/handler"
	"vasset/auth-service/internal/repository"
	"vasset/auth-service/internal/service"
	"vasset/auth-service/internal/utils"
	pb "vasset/auth-service/proto"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("config/dev.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Auth Service on port %d...", cfg.Server.Port)

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

	// 测试 Redis 连接
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		log.Println("✓ Connected to Redis")
	}

	// 4. 初始化 Repository
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// 5. 初始化工具
	jwtUtil := utils.NewJWTUtil(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
	)

	// 6. 初始化 Service
	tokenService := service.NewTokenService(jwtUtil, redisClient, sessionRepo, userRepo)
	userService := service.NewUserService(userRepo, &cfg.Password)
	authService := service.NewAuthService(userService, tokenService, sessionRepo, redisClient, &cfg.Session, &cfg.Password)

	// 7. 初始化 gRPC 服务器
	grpcServer := handler.NewGRPCServer(authService, userService, tokenService)

	// 8. 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, grpcServer)

	// 9. 启动会话清理任务
	go startSessionCleanup(ctx, sessionRepo, cfg.Session.CleanupInterval)

	// 10. 优雅关闭
	go func() {
		log.Printf("✓ gRPC server listening on :%d", cfg.Server.Port)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	s.GracefulStop()
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

// startSessionCleanup 启动会话清理任务
func startSessionCleanup(ctx context.Context, sessionRepo *repository.SessionRepository, intervalSeconds int) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sessionRepo.DeleteExpiredSessions(ctx); err != nil {
				log.Printf("Failed to cleanup expired sessions: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
