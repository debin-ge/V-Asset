package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

type AdminProxyHandler struct {
	adminClient pb.AdminServiceClient
	timeout     time.Duration
}

const (
	proxyListDefaultPage      = 1
	proxyListDefaultPageSize  = 20
	proxyListMaxPage          = 10000
	proxyListMaxPageSize      = 100
	proxyUsageDefaultPage     = 1
	proxyUsageDefaultPageSize = 20
	proxyUsageMaxPage         = 10000
	proxyUsageMaxPageSize     = 100
	proxyUsageMaxTimeRange    = 31 * 24 * time.Hour
)

var allowedProxyListSortFields = map[string]struct{}{
	"risk_score":        {},
	"priority":          {},
	"fail_count":        {},
	"active_task_count": {},
	"updated_at":        {},
	"last_used_at":      {},
}

var (
	errInvalidProxyUsageTimeRange  = errors.New("start_time must be before end_time")
	errProxyUsageTimeRangeTooLarge = errors.New("time range must not exceed 31 days")
)

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
		Healthy:                   resp.GetHealthy(),
		Mode:                      resp.GetMode(),
		Message:                   resp.GetMessage(),
		ProxyURL:                  resp.GetProxyUrl(),
		ProxyLeaseID:              resp.GetProxyLeaseId(),
		ProxyExpireAt:             resp.GetProxyExpireAt(),
		CheckedAt:                 resp.GetCheckedAt(),
		AvailableManualProxyCount: resp.GetAvailableManualProxyCount(),
		DynamicConfigured:         resp.GetDynamicConfigured(),
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
	page, err := parsePositiveInt32Query(c, "page", proxyListDefaultPage, proxyListMaxPage)
	if err != nil {
		models.BadRequest(c, "invalid page")
		return
	}
	pageSize, err := parsePositiveInt32Query(c, "page_size", proxyListDefaultPageSize, proxyListMaxPageSize)
	if err != nil {
		models.BadRequest(c, "invalid page_size")
		return
	}
	sortBy, sortOrder, err := parseProxyListSortQuery(c.Query("sort_by"), c.Query("sort_order"))
	if err != nil {
		models.BadRequest(c, err.Error())
		return
	}

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
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	items := make([]models.AdminProxyInfo, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, models.AdminProxyInfo{
			ID:                   item.GetId(),
			Host:                 item.GetHost(),
			Port:                 item.GetPort(),
			Protocol:             item.GetProtocol(),
			Username:             item.GetUsername(),
			Region:               item.GetRegion(),
			Priority:             item.GetPriority(),
			PlatformTags:         item.GetPlatformTags(),
			Remark:               item.GetRemark(),
			Status:               item.GetStatus(),
			LastUsedAt:           item.GetLastUsedAt(),
			SuccessCount:         item.GetSuccessCount(),
			FailCount:            item.GetFailCount(),
			CreatedAt:            item.GetCreatedAt(),
			UpdatedAt:            item.GetUpdatedAt(),
			CooldownUntil:        item.GetCooldownUntil(),
			ConsecutiveFailCount: item.GetConsecutiveFailCount(),
			RiskScore:            item.GetRiskScore(),
			LastErrorCategory:    item.GetLastErrorCategory(),
			LastFailAt:           item.GetLastFailAt(),
			MaxConcurrent:        item.GetMaxConcurrent(),
			ActiveTaskCount:      item.GetActiveTaskCount(),
		})
	}

	models.Success(c, models.AdminProxyListResponse{
		Items: items,
		Pagination: models.AdminProxyPagination{
			Page:     resp.GetPage(),
			PageSize: resp.GetPageSize(),
			Total:    resp.GetTotal(),
		},
	})
}

func (h *AdminProxyHandler) ListUsageEvents(c *gin.Context) {
	proxyID, err := parseOptionalInt64Query(c, "proxy_id")
	if err != nil {
		models.BadRequest(c, "invalid proxy_id")
		return
	}
	page, err := parsePositiveInt32Query(c, "page", proxyUsageDefaultPage, proxyUsageMaxPage)
	if err != nil {
		models.BadRequest(c, "invalid page")
		return
	}
	pageSize, err := parsePositiveInt32Query(c, "page_size", proxyUsageDefaultPageSize, proxyUsageMaxPageSize)
	if err != nil {
		models.BadRequest(c, "invalid page_size")
		return
	}

	success := strings.ToLower(c.DefaultQuery("success", "all"))
	if !isAllowedProxyUsageValue(success, "all", "success", "failed") {
		models.BadRequest(c, "invalid success")
		return
	}
	stage := strings.ToLower(c.Query("stage"))
	if stage != "" && !isAllowedProxyUsageValue(stage, "parse", "download") {
		models.BadRequest(c, "invalid stage")
		return
	}
	sourceType := strings.ToLower(c.Query("source_type"))
	if sourceType != "" && !isAllowedProxyUsageValue(sourceType, "manual", "dynamic", "manual_pool", "dynamic_api") {
		models.BadRequest(c, "invalid source_type")
		return
	}
	sortOrder := strings.ToLower(c.DefaultQuery("sort_order", "desc"))
	if !isAllowedProxyUsageValue(sortOrder, "asc", "desc") {
		models.BadRequest(c, "invalid sort_order")
		return
	}

	startTime, endTime, err := parseProxyUsageTimeRange(c.Query("start_time"), c.Query("end_time"))
	if err != nil {
		models.BadRequest(c, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.ListProxyUsageEvents(ctx, &pb.AdminListProxyUsageEventsRequest{
		TaskId:        c.Query("task_id"),
		ProxyId:       proxyID,
		ProxyLeaseId:  c.Query("proxy_lease_id"),
		SourceType:    sourceType,
		Stage:         stage,
		Platform:      c.Query("platform"),
		Success:       success,
		ErrorCategory: c.Query("error_category"),
		StartTimeUnix: startTime.Unix(),
		EndTimeUnix:   endTime.Unix(),
		Page:          page,
		PageSize:      pageSize,
		SortOrder:     sortOrder,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	events := make([]models.AdminProxyUsageEvent, 0, len(resp.GetEvents()))
	for _, item := range resp.GetEvents() {
		events = append(events, models.AdminProxyUsageEvent{
			ID:                   item.GetId(),
			TaskID:               item.GetTaskId(),
			ProxyID:              item.GetProxyId(),
			ProxyLeaseID:         item.GetProxyLeaseId(),
			SourceType:           publicProxyUsageSourceType(item.GetSourceType()),
			Stage:                item.GetStage(),
			Platform:             item.GetPlatform(),
			Success:              item.GetSuccess(),
			ErrorCategory:        item.GetErrorCategory(),
			ErrorMessage:         item.GetErrorMessage(),
			CreatedAt:            item.GetCreatedAt(),
			ProxyHost:            item.GetProxyHost(),
			ProxyPort:            item.GetProxyPort(),
			ProxyProtocol:        item.GetProxyProtocol(),
			ProxyRegion:          item.GetProxyRegion(),
			ProxyRiskScore:       item.GetProxyRiskScore(),
			ProxyCooldownUntil:   item.GetProxyCooldownUntil(),
			ProxyActiveTaskCount: item.GetProxyActiveTaskCount(),
			ProxyMaxConcurrent:   item.GetProxyMaxConcurrent(),
		})
	}

	models.Success(c, models.AdminProxyUsageEventListResponse{
		Events: events,
		Pagination: models.AdminProxyUsageEventPagination{
			Page:     resp.GetPage(),
			PageSize: resp.GetPageSize(),
			Total:    resp.GetTotal(),
		},
		Summary: proxyUsageSummaryResponse(resp.GetSummary()),
	})
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

func parseOptionalInt64Query(c *gin.Context, key string) (int64, error) {
	value := c.Query(key)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func parsePositiveInt32Query(c *gin.Context, key string, defaultValue int32, maxValue int32) (int32, error) {
	value := c.Query(key)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if parsed < 1 {
		return defaultValue, nil
	}
	if maxValue > 0 && parsed > int(maxValue) {
		return maxValue, nil
	}
	return int32(parsed), nil
}

func parseProxyListSortQuery(sortByValue, sortOrderValue string) (string, string, error) {
	sortBy := strings.ToLower(strings.TrimSpace(sortByValue))
	sortOrder := strings.ToLower(strings.TrimSpace(sortOrderValue))

	if sortBy == "" {
		if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
			return "", "", errors.New("invalid sort_order")
		}
		return "", "", nil
	}

	if _, ok := allowedProxyListSortFields[sortBy]; !ok {
		return "", "", errors.New("invalid sort_by")
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return "", "", errors.New("invalid sort_order")
	}
	return sortBy, sortOrder, nil
}

func parseProxyUsageTimeRange(startValue, endValue string) (time.Time, time.Time, error) {
	endTime := time.Now()
	var err error
	if endValue != "" {
		endTime, err = time.Parse(time.RFC3339, endValue)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	startTime := endTime.Add(-24 * time.Hour)
	if startValue != "" {
		startTime, err = time.Parse(time.RFC3339, startValue)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if startTime.After(endTime) {
		return time.Time{}, time.Time{}, errInvalidProxyUsageTimeRange
	}
	if endTime.Sub(startTime) > proxyUsageMaxTimeRange {
		return time.Time{}, time.Time{}, errProxyUsageTimeRangeTooLarge
	}
	return startTime, endTime, nil
}

func isAllowedProxyUsageValue(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func publicProxyUsageSourceType(value string) string {
	switch value {
	case "manual_pool":
		return "manual"
	case "dynamic_api":
		return "dynamic"
	default:
		return value
	}
}

func proxyUsageSummaryResponse(summary *pb.AdminProxyUsageEventSummary) models.AdminProxyUsageEventSummary {
	if summary == nil {
		return models.AdminProxyUsageEventSummary{}
	}
	return models.AdminProxyUsageEventSummary{
		SuccessCount:   summary.GetSuccessCount(),
		FailureCount:   summary.GetFailureCount(),
		FailureRate:    summary.GetFailureRate(),
		CategoryCounts: proxyUsageCountResponse(summary.GetCategoryCounts()),
		StageCounts:    proxyUsageCountResponse(summary.GetStageCounts()),
		PlatformCounts: proxyUsageCountResponse(summary.GetPlatformCounts()),
	}
}

func proxyUsageCountResponse(items []*pb.AdminProxyUsageEventCount) []models.AdminProxyUsageEventCount {
	result := make([]models.AdminProxyUsageEventCount, 0, len(items))
	for _, item := range items {
		result = append(result, models.AdminProxyUsageEventCount{
			Key:   item.GetKey(),
			Count: item.GetCount(),
		})
	}
	return result
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
