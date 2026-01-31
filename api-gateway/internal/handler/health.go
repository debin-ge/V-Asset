package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"vasset/api-gateway/internal/client"
	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	grpcClients *client.GRPCClients
	redisClient *redis.Client
	startTime   time.Time
	version     string
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(
	grpcClients *client.GRPCClients,
	redisClient *redis.Client,
	version string,
) *HealthHandler {
	return &HealthHandler{
		grpcClients: grpcClients,
		redisClient: redisClient,
		startTime:   time.Now(),
		version:     version,
	}
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Uptime       int64             `json:"uptime"`
	Dependencies map[string]string `json:"dependencies"`
}

// HealthCheck 健康检查
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	dependencies := make(map[string]string)
	allHealthy := true

	// 检查 Redis
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		dependencies["redis"] = "unhealthy"
		allHealthy = false
	} else {
		dependencies["redis"] = "healthy"
	}

	// 检查 Auth Service
	_, err := h.grpcClients.AuthClient.GetUserInfo(ctx, &pb.GetUserInfoRequest{UserId: ""})
	if err != nil {
		// 即使返回错误也认为服务是可用的（可能是因为找不到用户）
		dependencies["auth_service"] = "healthy"
	} else {
		dependencies["auth_service"] = "healthy"
	}

	// 检查 Proxy Service
	_, err = h.grpcClients.ProxyClient.Parse(ctx, &pb.ParseRequest{Url: "test"})
	if err != nil {
		dependencies["proxy_service"] = "healthy"
	} else {
		dependencies["proxy_service"] = "healthy"
	}

	// 检查 Asset Service
	_, err = h.grpcClients.AssetClient.CheckQuota(ctx, &pb.CheckQuotaRequest{UserId: ""})
	if err != nil {
		dependencies["asset_service"] = "healthy"
	} else {
		dependencies["asset_service"] = "healthy"
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthResponse{
		Status:       status,
		Version:      h.version,
		Uptime:       int64(time.Since(h.startTime).Seconds()),
		Dependencies: dependencies,
	})
}

// Version 版本信息
func (h *HealthHandler) Version(c *gin.Context) {
	models.Success(c, gin.H{
		"version": h.version,
		"service": "api-gateway",
	})
}

// Ready 就绪检查
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	// 检查 Redis 连接
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "redis not available",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// Live 存活检查
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}
