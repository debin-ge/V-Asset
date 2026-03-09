package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

type AdminProxyHandler struct {
	adminClient pb.AdminServiceClient
	timeout     time.Duration
}

func NewAdminProxyHandler(adminClient pb.AdminServiceClient, timeout time.Duration) *AdminProxyHandler {
	return &AdminProxyHandler{adminClient: adminClient, timeout: timeout}
}

func (h *AdminProxyHandler) GetSourceStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetProxySourceStatus(ctx, &pb.AdminEmpty{})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, models.ProxySourceStatusResponse{
		Healthy:       resp.GetHealthy(),
		Mode:          resp.GetMode(),
		Message:       resp.GetMessage(),
		ProxyURL:      resp.GetProxyUrl(),
		ProxyLeaseID:  resp.GetProxyLeaseId(),
		ProxyExpireAt: resp.GetProxyExpireAt(),
		CheckedAt:     resp.GetCheckedAt(),
	})
}

func (h *AdminProxyHandler) GetSourcePolicy(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetProxySourcePolicy(ctx, &pb.AdminEmpty{})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, models.AdminProxySourcePolicy{
		ID:                       resp.GetId(),
		ScopeType:                resp.GetScopeType(),
		ScopeValue:               resp.GetScopeValue(),
		PrimarySource:            resp.GetPrimarySource(),
		FallbackSource:           resp.GetFallbackSource(),
		FallbackEnabled:          resp.GetFallbackEnabled(),
		DynamicTimeoutMS:         resp.GetDynamicTimeoutMs(),
		DynamicRetryCount:        resp.GetDynamicRetryCount(),
		DynamicCircuitBreakerSec: resp.GetDynamicCircuitBreakerSec(),
		MinLeaseTTLSec:           resp.GetMinLeaseTtlSec(),
		ManualSelectionStrategy:  resp.GetManualSelectionStrategy(),
	})
}

func (h *AdminProxyHandler) UpdateSourcePolicy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid policy id")
		return
	}

	var req models.AdminUpdateProxySourcePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	_, err = h.adminClient.UpdateProxySourcePolicy(ctx, &pb.AdminUpdateProxySourcePolicyRequest{
		Id:                       id,
		PrimarySource:            req.PrimarySource,
		FallbackSource:           req.FallbackSource,
		FallbackEnabled:          req.FallbackEnabled,
		DynamicTimeoutMs:         req.DynamicTimeoutMS,
		DynamicRetryCount:        req.DynamicRetryCount,
		DynamicCircuitBreakerSec: req.DynamicCircuitBreakerSec,
		MinLeaseTtlSec:           req.MinLeaseTTLSec,
		ManualSelectionStrategy:  req.ManualSelectionStrategy,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *AdminProxyHandler) List(c *gin.Context) {
	var statusValue int32
	hasStatus := false
	if value := c.Query("status"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			models.BadRequest(c, "invalid status")
			return
		}
		statusValue = int32(parsed)
		hasStatus = true
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.ListProxies(ctx, &pb.AdminListProxiesRequest{
		Search:    c.Query("search"),
		Protocol:  c.Query("protocol"),
		Region:    c.Query("region"),
		Status:    statusValue,
		HasStatus: hasStatus,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	items := make([]models.AdminProxyInfo, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, models.AdminProxyInfo{
			ID:           item.GetId(),
			Host:         item.GetHost(),
			Port:         item.GetPort(),
			Protocol:     item.GetProtocol(),
			Username:     item.GetUsername(),
			Region:       item.GetRegion(),
			Priority:     item.GetPriority(),
			PlatformTags: item.GetPlatformTags(),
			Remark:       item.GetRemark(),
			Status:       item.GetStatus(),
			LastUsedAt:   item.GetLastUsedAt(),
			SuccessCount: item.GetSuccessCount(),
			FailCount:    item.GetFailCount(),
			CreatedAt:    item.GetCreatedAt(),
			UpdatedAt:    item.GetUpdatedAt(),
		})
	}

	models.Success(c, models.AdminProxyListResponse{Items: items})
}

func (h *AdminProxyHandler) Create(c *gin.Context) {
	var req models.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.CreateProxy(ctx, &pb.AdminCreateProxyRequest{
		Host:         req.Host,
		Port:         req.Port,
		Protocol:     req.Protocol,
		Username:     req.Username,
		Password:     req.Password,
		Region:       req.Region,
		Priority:     req.Priority,
		PlatformTags: req.PlatformTags,
		Remark:       req.Remark,
		Status:       req.Status,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, gin.H{"id": resp.GetId()})
}

func (h *AdminProxyHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
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

	_, err = h.adminClient.UpdateProxy(ctx, &pb.AdminUpdateProxyRequest{
		Id:           id,
		Host:         req.Host,
		Port:         req.Port,
		Protocol:     req.Protocol,
		Username:     req.Username,
		Password:     req.Password,
		Region:       req.Region,
		Priority:     req.Priority,
		PlatformTags: req.PlatformTags,
		Remark:       req.Remark,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *AdminProxyHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	var req struct {
		Status *int32 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if req.Status == nil {
		models.BadRequest(c, "invalid request: status is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	_, err = h.adminClient.UpdateProxyStatus(ctx, &pb.AdminUpdateProxyStatusRequest{Id: id, Status: *req.Status})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *AdminProxyHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	_, err = h.adminClient.DeleteProxy(ctx, &pb.AdminDeleteRequest{Id: id})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, gin.H{"success": true})
}
