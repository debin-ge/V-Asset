package handler

import (
	"context"
	"strconv"
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

// NewProxyHandler 创建代理管理处理器
func NewProxyHandler(assetClient pb.AssetServiceClient, timeout time.Duration) *ProxyHandler {
	return &ProxyHandler{
		assetClient: assetClient,
		timeout:     timeout,
	}
}

// CreateProxy 创建代理
func (h *ProxyHandler) CreateProxy(c *gin.Context) {
	var req models.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.CreateProxy(ctx, &pb.CreateProxyRequest{
		Ip:          req.IP,
		Port:        req.Port,
		Username:    req.Username,
		Password:    req.Password,
		Protocol:    req.Protocol,
		Region:      req.Region,
		CheckHealth: req.CheckHealth,
	})
	if err != nil {
		models.InternalError(c, "failed to create proxy: "+err.Error())
		return
	}

	models.Success(c, models.CreateProxyResponse{
		ID:                resp.Id,
		HealthCheckPassed: resp.HealthCheckPassed,
		HealthCheckError:  resp.HealthCheckError,
	})
}

// UpdateProxy 更新代理
func (h *ProxyHandler) UpdateProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	var req models.UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	_, err = h.assetClient.UpdateProxy(ctx, &pb.UpdateProxyRequest{
		Id:       id,
		Username: req.Username,
		Password: req.Password,
		Protocol: req.Protocol,
		Region:   req.Region,
	})
	if err != nil {
		models.InternalError(c, "failed to update proxy: "+err.Error())
		return
	}

	models.Success(c, nil)
}

// DeleteProxy 删除代理
func (h *ProxyHandler) DeleteProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	_, err = h.assetClient.DeleteProxy(ctx, &pb.DeleteProxyRequest{
		Id: id,
	})
	if err != nil {
		models.InternalError(c, "failed to delete proxy: "+err.Error())
		return
	}

	models.Success(c, nil)
}

// GetProxy 获取代理详情
func (h *ProxyHandler) GetProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.GetProxy(ctx, &pb.GetProxyRequest{
		Id: id,
	})
	if err != nil {
		models.InternalError(c, "failed to get proxy: "+err.Error())
		return
	}

	models.Success(c, models.ProxyInfo{
		ID:              resp.Proxy.Id,
		IP:              resp.Proxy.Ip,
		Port:            resp.Proxy.Port,
		Username:        resp.Proxy.Username,
		Password:        resp.Proxy.Password,
		Protocol:        resp.Proxy.Protocol,
		Region:          resp.Proxy.Region,
		Status:          resp.Proxy.Status,
		LastCheckAt:     resp.Proxy.LastCheckAt,
		LastCheckResult: resp.Proxy.LastCheckResult,
		SuccessCount:    int64(resp.Proxy.SuccessCount),
		FailCount:       int64(resp.Proxy.FailCount),
		LastUsedAt:      resp.Proxy.LastUsedAt,
		CreatedAt:       resp.Proxy.CreatedAt,
		UpdatedAt:       resp.Proxy.UpdatedAt,
	})
}

// ListProxies 列出代理
func (h *ProxyHandler) ListProxies(c *gin.Context) {
	var req models.ListProxiesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.ListProxies(ctx, &pb.ListProxiesRequest{
		Status:   req.Status,
		Protocol: req.Protocol,
		Region:   req.Region,
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
	})
	if err != nil {
		models.InternalError(c, "failed to list proxies: "+err.Error())
		return
	}

	// 转换结果
	items := make([]models.ProxyInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, models.ProxyInfo{
			ID:              item.Id,
			IP:              item.Ip,
			Port:            item.Port,
			Username:        item.Username,
			Password:        item.Password,
			Protocol:        item.Protocol,
			Region:          item.Region,
			Status:          item.Status,
			LastCheckAt:     item.LastCheckAt,
			LastCheckResult: item.LastCheckResult,
			SuccessCount:    int64(item.SuccessCount),
			FailCount:       int64(item.FailCount),
			LastUsedAt:      item.LastUsedAt,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}

	models.Success(c, models.PagedResponse{
		Total:    resp.Total,
		Page:     int(resp.Page),
		PageSize: int(resp.PageSize),
		Items:    items,
	})
}

// CheckProxyHealth 检查代理健康
func (h *ProxyHandler) CheckProxyHealth(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.CheckProxyHealth(ctx, &pb.CheckProxyHealthRequest{
		Id: id,
	})
	if err != nil {
		models.InternalError(c, "failed to check proxy health: "+err.Error())
		return
	}

	models.Success(c, models.ProxyHealthCheckResponse{
		Healthy:   resp.Healthy,
		Error:     resp.Error,
		LatencyMs: resp.LatencyMs,
	})
}
