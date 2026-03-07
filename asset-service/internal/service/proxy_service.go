package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/models"
	dynamicproxy "vasset/asset-service/internal/proxy"
	"vasset/asset-service/internal/repository"
)

// ProxyService 代理业务逻辑层
type ProxyService struct {
	repo        *repository.ProxyRepository
	policyRepo  *repository.ProxyPolicyRepository
	bindingRepo *repository.TaskProxyBindingRepository
	provider    *dynamicproxy.Provider
}

// NewProxyService 创建代理服务
func NewProxyService(
	repo *repository.ProxyRepository,
	policyRepo *repository.ProxyPolicyRepository,
	bindingRepo *repository.TaskProxyBindingRepository,
	cfg *config.Config,
) *ProxyService {
	return &ProxyService{
		repo:        repo,
		policyRepo:  policyRepo,
		bindingRepo: bindingRepo,
		provider:    dynamicproxy.NewProvider(&cfg.Proxy),
	}
}

// GetAvailableProxy 获取可用代理
func (s *ProxyService) GetAvailableProxy(ctx context.Context, protocol *models.ProxyProtocol, region *string) (string, string, string, error) {
	if s.provider != nil && s.provider.Enabled() {
		lease, err := s.provider.GetLeaseWithRetry(ctx)
		if err != nil {
			return "", "", "", err
		}
		return lease.URL, lease.LeaseID, lease.ExpireAt, nil
	}

	proxy, err := s.repo.GetAvailableProxy(ctx, protocol, region)
	if err != nil {
		return "", "", "", err
	}
	if proxy == nil {
		return "", "", "", errors.New("no available proxy")
	}

	leaseID := fmt.Sprintf("static-%d", proxy.ID)
	return proxy.GetURL(), leaseID, "", nil
}

// AcquireProxyForTask 为任务分配或复用代理
func (s *ProxyService) AcquireProxyForTask(
	ctx context.Context,
	taskID string,
	protocol *models.ProxyProtocol,
	region *string,
	platform *string,
) (*models.TaskProxyBinding, error) {
	if taskID == "" {
		return nil, errors.New("task id is required")
	}

	existing, err := s.bindingRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.BindStatus == models.TaskProxyBindStatusBound {
		return existing, nil
	}

	policy, err := s.policyRepo.GetEffectivePolicy(ctx, platform)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, errors.New("no active proxy source policy")
	}

	result, degraded, degradeReason, err := s.acquireWithPolicy(ctx, policy, protocol, region)
	if err != nil {
		return nil, err
	}

	binding := s.buildBinding(taskID, policy, result, protocol, region, platform, degraded, degradeReason)
	if err := s.bindingRepo.CreateIfAbsent(ctx, binding); err != nil {
		if errors.Is(err, repository.ErrTaskProxyBindingAlreadyExists) {
			return s.bindingRepo.GetByTaskID(ctx, taskID)
		}
		return nil, err
	}

	return s.bindingRepo.GetByTaskID(ctx, taskID)
}

// ReportUsage 报告代理使用结果
func (s *ProxyService) ReportUsage(ctx context.Context, taskID, leaseID, stage string, success bool) error {
	if taskID != "" {
		binding, err := s.bindingRepo.GetByTaskID(ctx, taskID)
		if err != nil {
			return err
		}
		if binding != nil {
			if err := s.bindingRepo.UpdateReport(ctx, taskID, stage, success); err != nil {
				return err
			}
			if binding.SourceType == models.ProxySourceTypeManualPool && binding.ProxyID != nil {
				return s.repo.UpdateUsage(ctx, *binding.ProxyID, success)
			}
			return nil
		}
	}

	if leaseID == "" {
		return errors.New("task id or proxy lease id is required")
	}
	if s.provider != nil && s.provider.Enabled() {
		return nil
	}

	var proxyID int64
	if _, err := fmt.Sscanf(leaseID, "static-%d", &proxyID); err != nil {
		return nil
	}
	return s.repo.UpdateUsage(ctx, proxyID, success)
}

func (s *ProxyService) acquireWithPolicy(
	ctx context.Context,
	policy *models.ProxySourcePolicy,
	protocol *models.ProxyProtocol,
	region *string,
) (*models.ProxyAcquireResult, bool, *string, error) {
	result, err := s.trySource(ctx, policy.PrimarySource, policy, protocol, region)
	if err == nil {
		return result, false, nil, nil
	}

	if !policy.FallbackEnabled || policy.FallbackSource == nil || *policy.FallbackSource == "" {
		return nil, false, nil, err
	}

	degradeReason := err.Error()
	fallbackType := models.ProxySourceType(*policy.FallbackSource)
	result, err = s.trySource(ctx, fallbackType, policy, protocol, region)
	if err != nil {
		return nil, false, nil, err
	}

	return result, true, &degradeReason, nil
}

func (s *ProxyService) trySource(
	ctx context.Context,
	sourceType models.ProxySourceType,
	policy *models.ProxySourcePolicy,
	protocol *models.ProxyProtocol,
	region *string,
) (*models.ProxyAcquireResult, error) {
	switch sourceType {
	case models.ProxySourceTypeDynamicAPI:
		return s.tryDynamicSource(ctx, policy)
	case models.ProxySourceTypeManualPool:
		return s.tryManualSource(ctx, protocol, region)
	default:
		return nil, fmt.Errorf("unsupported proxy source type: %s", sourceType)
	}
}

func (s *ProxyService) tryDynamicSource(ctx context.Context, policy *models.ProxySourcePolicy) (*models.ProxyAcquireResult, error) {
	if s.provider == nil || !s.provider.Enabled() {
		return nil, errors.New("dynamic proxy API is not configured")
	}

	lease, err := s.provider.GetLeaseWithRetry(ctx)
	if err != nil {
		return nil, err
	}

	var expireAt *time.Time
	if lease.ExpireAt != "" {
		parsed, err := time.Parse(time.RFC3339, lease.ExpireAt)
		if err != nil {
			return nil, fmt.Errorf("invalid dynamic proxy expire_at: %w", err)
		}
		if policy.MinLeaseTTLSec > 0 && time.Until(parsed) < time.Duration(policy.MinLeaseTTLSec)*time.Second {
			return nil, errors.New("dynamic proxy ttl below minimum threshold")
		}
		expireAt = &parsed
	}

	leaseID := lease.LeaseID
	return &models.ProxyAcquireResult{
		SourceType:   models.ProxySourceTypeDynamicAPI,
		ProxyLeaseID: &leaseID,
		ProxyURL:     lease.URL,
		ExpireAt:     expireAt,
	}, nil
}

func (s *ProxyService) tryManualSource(
	ctx context.Context,
	protocol *models.ProxyProtocol,
	region *string,
) (*models.ProxyAcquireResult, error) {
	proxy, err := s.repo.GetAvailableProxy(ctx, protocol, region)
	if err != nil {
		return nil, err
	}
	if proxy == nil {
		return nil, errors.New("no available manual proxy")
	}

	proxyID := proxy.ID
	leaseID := fmt.Sprintf("static-%d", proxy.ID)
	return &models.ProxyAcquireResult{
		SourceType:   models.ProxySourceTypeManualPool,
		ProxyID:      &proxyID,
		ProxyLeaseID: &leaseID,
		ProxyURL:     proxy.GetURL(),
	}, nil
}

func (s *ProxyService) buildBinding(
	taskID string,
	policy *models.ProxySourcePolicy,
	result *models.ProxyAcquireResult,
	protocol *models.ProxyProtocol,
	region *string,
	platform *string,
	degraded bool,
	degradeReason *string,
) *models.TaskProxyBinding {
	protocolValue := string(models.ProxyProtocolHTTP)
	if protocol != nil && *protocol != "" {
		protocolValue = string(*protocol)
	}

	policyID := policy.ID
	return &models.TaskProxyBinding{
		TaskID:           taskID,
		SourceType:       result.SourceType,
		SourcePolicyID:   &policyID,
		ProxyID:          result.ProxyID,
		ProxyLeaseID:     result.ProxyLeaseID,
		ProxyURLSnapshot: result.ProxyURL,
		Protocol:         protocolValue,
		Region:           region,
		Platform:         platform,
		ExpireAt:         result.ExpireAt,
		BindStatus:       models.TaskProxyBindStatusBound,
		IsDegraded:       degraded,
		DegradeReason:    degradeReason,
	}
}

// GetSourcePolicy 获取当前生效的全局策略
func (s *ProxyService) GetSourcePolicy(ctx context.Context) (*models.ProxySourcePolicy, error) {
	return s.policyRepo.GetEffectivePolicy(ctx, nil)
}

// UpdateSourcePolicy 更新全局策略
func (s *ProxyService) UpdateSourcePolicy(
	ctx context.Context,
	id int64,
	primarySource string,
	fallbackSource *string,
	fallbackEnabled bool,
	dynamicTimeoutMS int,
	dynamicRetryCount int,
	dynamicCircuitBreakerSec int,
	minLeaseTTLSec int,
	manualSelectionStrategy string,
) error {
	return s.policyRepo.UpdatePolicy(
		ctx,
		id,
		primarySource,
		fallbackSource,
		fallbackEnabled,
		dynamicTimeoutMS,
		dynamicRetryCount,
		dynamicCircuitBreakerSec,
		minLeaseTTLSec,
		manualSelectionStrategy,
	)
}

// ListProxies 列出手动代理池
func (s *ProxyService) ListProxies(
	ctx context.Context,
	search, protocol, region *string,
	status *models.ProxyStatus,
) ([]*models.Proxy, error) {
	return s.repo.ListProxies(ctx, search, protocol, region, status)
}

// CreateProxy 创建手动代理
func (s *ProxyService) CreateProxy(ctx context.Context, proxy *models.Proxy) (int64, error) {
	if proxy == nil {
		return 0, errors.New("proxy is required")
	}
	if proxy.Host == nil || *proxy.Host == "" {
		if proxy.IP == "" {
			return 0, errors.New("host is required")
		}
		host := proxy.IP
		proxy.Host = &host
	}
	if proxy.Port <= 0 {
		return 0, errors.New("port must be greater than 0")
	}
	if proxy.Protocol == "" {
		proxy.Protocol = models.ProxyProtocolHTTP
	}
	return s.repo.CreateProxy(ctx, proxy)
}

// UpdateProxy 更新手动代理
func (s *ProxyService) UpdateProxy(ctx context.Context, proxy *models.Proxy) error {
	if proxy == nil {
		return errors.New("proxy is required")
	}
	if proxy.ID == 0 {
		return errors.New("proxy id is required")
	}
	if proxy.Host == nil || *proxy.Host == "" {
		if proxy.IP == "" {
			return errors.New("host is required")
		}
		host := proxy.IP
		proxy.Host = &host
	}
	if proxy.Port <= 0 {
		return errors.New("port must be greater than 0")
	}
	if proxy.Protocol == "" {
		proxy.Protocol = models.ProxyProtocolHTTP
	}
	return s.repo.UpdateProxy(ctx, proxy)
}

// UpdateProxyStatus 更新手动代理状态
func (s *ProxyService) UpdateProxyStatus(ctx context.Context, id int64, status models.ProxyStatus) error {
	if id == 0 {
		return errors.New("proxy id is required")
	}
	return s.repo.UpdateProxyStatus(ctx, id, status)
}

// DeleteProxy 软删除手动代理
func (s *ProxyService) DeleteProxy(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("proxy id is required")
	}
	return s.repo.DeleteProxy(ctx, id)
}

// IsNotFoundError 判断是否为记录不存在
func (s *ProxyService) IsNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
