package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

// ProxyHandler 代理管理处理器
type ProxyHandler struct {
	assetClient pb.AssetServiceClient
	timeout     time.Duration
}

// GetProxySourceStatus 探测动态代理源状态。
func (h *ProxyHandler) GetProxySourceStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	checkedAt := time.Now().Format(time.RFC3339)
	resp, err := h.assetClient.CheckProxySourceStatus(ctx, &pb.CheckProxySourceStatusRequest{})
	if err != nil {
		models.Success(c, models.ProxySourceStatusResponse{
			Healthy:   false,
			Mode:      "read-only",
			Message:   "Failed to check proxy source: " + err.Error(),
			CheckedAt: checkedAt,
		})
		return
	}

	models.Success(c, models.ProxySourceStatusResponse{
		Healthy:                   resp.GetHealthy(),
		Mode:                      resp.GetMode(),
		Message:                   resp.GetMessage(),
		CheckedAt:                 resp.GetCheckedAt(),
		AvailableManualProxyCount: resp.GetAvailableManualProxyCount(),
		DynamicConfigured:         resp.GetDynamicConfigured(),
	})
}

// NewProxyHandler 创建代理管理处理器
func NewProxyHandler(assetClient pb.AssetServiceClient, timeout time.Duration) *ProxyHandler {
	return &ProxyHandler{
		assetClient: assetClient,
		timeout:     timeout,
	}
}
