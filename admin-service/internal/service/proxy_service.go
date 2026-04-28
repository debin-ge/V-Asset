package service

import (
	"context"
	"net/url"
	"time"

	"youdlp/admin-service/internal/models"
	pb "youdlp/admin-service/proto"
)

type ProxyService struct {
	assetClient pb.AssetServiceClient
}

func NewProxyService(assetClient pb.AssetServiceClient) *ProxyService {
	return &ProxyService{assetClient: assetClient}
}

func (s *ProxyService) GetSourceStatus(ctx context.Context) (*models.ProxySourceStatusResponse, error) {
	checkedAt := time.Now().Format(time.RFC3339)
	resp, err := s.assetClient.CheckProxySourceStatus(ctx, &pb.CheckProxySourceStatusRequest{})
	if err != nil {
		return &models.ProxySourceStatusResponse{
			Healthy:   false,
			Mode:      "read-only",
			Message:   "Failed to check proxy source: " + err.Error(),
			CheckedAt: checkedAt,
		}, nil
	}

	return &models.ProxySourceStatusResponse{
		Healthy:                   resp.Healthy,
		Mode:                      resp.Mode,
		Message:                   resp.Message,
		CheckedAt:                 resp.CheckedAt,
		AvailableManualProxyCount: resp.AvailableManualProxyCount,
		DynamicConfigured:         resp.DynamicConfigured,
	}, nil
}

func (s *ProxyService) GetSourcePolicy(ctx context.Context) (*models.ProxySourcePolicy, error) {
	resp, err := s.assetClient.GetProxySourcePolicy(ctx, &pb.GetProxySourcePolicyRequest{})
	if err != nil {
		return nil, err
	}

	return &models.ProxySourcePolicy{
		ID:                       resp.Id,
		ScopeType:                resp.ScopeType,
		ScopeValue:               resp.ScopeValue,
		PrimarySource:            resp.PrimarySource,
		FallbackSource:           resp.FallbackSource,
		FallbackEnabled:          resp.FallbackEnabled,
		DynamicTimeoutMS:         resp.DynamicTimeoutMs,
		DynamicRetryCount:        resp.DynamicRetryCount,
		DynamicCircuitBreakerSec: resp.DynamicCircuitBreakerSec,
		MinLeaseTTLSec:           resp.MinLeaseTtlSec,
		ManualSelectionStrategy:  resp.ManualSelectionStrategy,
	}, nil
}

func (s *ProxyService) UpdateSourcePolicy(ctx context.Context, id int64, req models.UpdateProxySourcePolicyRequest) error {
	_, err := s.assetClient.UpdateProxySourcePolicy(ctx, &pb.UpdateProxySourcePolicyRequest{
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
	return err
}

func (s *ProxyService) List(ctx context.Context, req models.ListProxiesRequest) (*models.ProxyListResponse, error) {
	status := int32(-1)
	if req.Status != nil {
		status = *req.Status
	}

	resp, err := s.assetClient.ListProxies(ctx, &pb.ListProxiesRequest{
		Search:   req.Search,
		Protocol: req.Protocol,
		Region:   req.Region,
		Status:   status,
	})
	if err != nil {
		return nil, err
	}

	items := make([]models.ProxyInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, models.ProxyInfo{
			ID:                   item.Id,
			Host:                 item.Host,
			Port:                 item.Port,
			Protocol:             item.Protocol,
			Username:             item.Username,
			Region:               item.Region,
			Priority:             item.Priority,
			PlatformTags:         item.PlatformTags,
			Remark:               item.Remark,
			Status:               item.Status,
			LastUsedAt:           item.LastUsedAt,
			SuccessCount:         item.SuccessCount,
			FailCount:            item.FailCount,
			CreatedAt:            item.CreatedAt,
			UpdatedAt:            item.UpdatedAt,
			CooldownUntil:        item.CooldownUntil,
			ConsecutiveFailCount: item.ConsecutiveFailCount,
			RiskScore:            item.RiskScore,
			LastErrorCategory:    item.LastErrorCategory,
			LastFailAt:           item.LastFailAt,
			MaxConcurrent:        item.MaxConcurrent,
			ActiveTaskCount:      item.ActiveTaskCount,
		})
	}

	return &models.ProxyListResponse{Items: items}, nil
}

func (s *ProxyService) ListUsageEvents(ctx context.Context, req models.ProxyUsageEventFilter) (*models.ProxyUsageEventListResponse, error) {
	resp, err := s.assetClient.ListProxyUsageEvents(ctx, &pb.ListProxyUsageEventsRequest{
		TaskId:        req.TaskID,
		ProxyId:       req.ProxyID,
		ProxyLeaseId:  req.ProxyLeaseID,
		SourceType:    req.SourceType,
		Stage:         req.Stage,
		Platform:      req.Platform,
		Success:       req.Success,
		ErrorCategory: req.ErrorCategory,
		StartTimeUnix: req.StartTimeUnix,
		EndTimeUnix:   req.EndTimeUnix,
		Page:          req.Page,
		PageSize:      req.PageSize,
		SortOrder:     req.SortOrder,
	})
	if err != nil {
		return nil, err
	}

	events := make([]models.ProxyUsageEventInfo, 0, len(resp.Events))
	for _, event := range resp.Events {
		events = append(events, models.ProxyUsageEventInfo{
			ID:                   event.Id,
			TaskID:               event.TaskId,
			ProxyID:              event.ProxyId,
			ProxyLeaseID:         event.ProxyLeaseId,
			SourceType:           event.SourceType,
			Stage:                event.Stage,
			Platform:             event.Platform,
			Success:              event.Success,
			ErrorCategory:        event.ErrorCategory,
			ErrorMessage:         event.ErrorMessage,
			CreatedAt:            event.CreatedAt,
			ProxyHost:            event.ProxyHost,
			ProxyPort:            event.ProxyPort,
			ProxyProtocol:        event.ProxyProtocol,
			ProxyRegion:          event.ProxyRegion,
			ProxyRiskScore:       event.ProxyRiskScore,
			ProxyCooldownUntil:   event.ProxyCooldownUntil,
			ProxyActiveTaskCount: event.ProxyActiveTaskCount,
			ProxyMaxConcurrent:   event.ProxyMaxConcurrent,
		})
	}

	return &models.ProxyUsageEventListResponse{
		Events:   events,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
		Summary:  proxyUsageSummaryFromProto(resp.Summary),
	}, nil
}

func (s *ProxyService) Create(ctx context.Context, req models.CreateProxyRequest) (int64, error) {
	resp, err := s.assetClient.CreateProxy(ctx, &pb.CreateProxyRequest{
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
		return 0, err
	}
	return resp.Id, nil
}

func proxyUsageSummaryFromProto(summary *pb.ProxyUsageEventSummary) models.ProxyUsageEventSummary {
	if summary == nil {
		return models.ProxyUsageEventSummary{}
	}
	return models.ProxyUsageEventSummary{
		SuccessCount:   summary.SuccessCount,
		FailureCount:   summary.FailureCount,
		FailureRate:    summary.FailureRate,
		CategoryCounts: proxyUsageCountsFromProto(summary.CategoryCounts),
		StageCounts:    proxyUsageCountsFromProto(summary.StageCounts),
		PlatformCounts: proxyUsageCountsFromProto(summary.PlatformCounts),
	}
}

func proxyUsageCountsFromProto(items []*pb.ProxyUsageEventCount) []models.ProxyUsageEventCount {
	result := make([]models.ProxyUsageEventCount, 0, len(items))
	for _, item := range items {
		result = append(result, models.ProxyUsageEventCount{
			Key:   item.Key,
			Count: item.Count,
		})
	}
	return result
}

func (s *ProxyService) Update(ctx context.Context, id int64, req models.UpdateProxyRequest) error {
	_, err := s.assetClient.UpdateProxy(ctx, &pb.UpdateProxyRequest{
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
	return err
}

func (s *ProxyService) UpdateStatus(ctx context.Context, id int64, status int32) error {
	_, err := s.assetClient.UpdateProxyStatus(ctx, &pb.UpdateProxyStatusRequest{
		Id:     id,
		Status: status,
	})
	return err
}

func (s *ProxyService) Delete(ctx context.Context, id int64) error {
	_, err := s.assetClient.DeleteProxy(ctx, &pb.DeleteProxyRequest{Id: id})
	return err
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
