package handler

import (
	"context"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
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
	resp, err := h.assetClient.GetAvailableProxy(ctx, &pb.GetAvailableProxyRequest{})
	if err != nil {
		models.Success(c, models.ProxySourceStatusResponse{
			Healthy:   false,
			Mode:      "dynamic-lease",
			Message:   "Failed to acquire proxy lease: " + err.Error(),
			CheckedAt: checkedAt,
		})
		return
	}

	if resp.ProxyUrl == "" {
		models.Success(c, models.ProxySourceStatusResponse{
			Healthy:   false,
			Mode:      "dynamic-lease",
			Message:   "Proxy source returned an empty lease",
			CheckedAt: checkedAt,
		})
		return
	}

	models.Success(c, models.ProxySourceStatusResponse{
		Healthy:       true,
		Mode:          "dynamic-lease",
		Message:       "Dynamic proxy source is ready",
		ProxyURL:      maskProxyURL(resp.ProxyUrl),
		ProxyLeaseID:  resp.ProxyLeaseId,
		ProxyExpireAt: resp.ExpireAt,
		CheckedAt:     checkedAt,
	})
}

// NewProxyHandler 创建代理管理处理器
func NewProxyHandler(assetClient pb.AssetServiceClient, timeout time.Duration) *ProxyHandler {
	return &ProxyHandler{
		assetClient: assetClient,
		timeout:     timeout,
	}
}

func maskProxyURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if username != "" {
			parsed.User = url.UserPassword(username, "***")
		}
	}
	return parsed.String()
}
