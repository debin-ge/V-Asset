package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"youdlp/asset-service/internal/config"
	"youdlp/asset-service/internal/models"
	dynamicproxy "youdlp/asset-service/internal/proxy"
	"youdlp/asset-service/internal/repository"
)

// ProxyService 代理业务逻辑层
type ProxyService struct {
	repo        *repository.ProxyRepository
	policyRepo  *repository.ProxyPolicyRepository
	bindingRepo *repository.TaskProxyBindingRepository
	provider    *dynamicproxy.Provider
}

const (
	proxyUsageStageParse    = "parse"
	proxyUsageStageDownload = "download"
	proxyUsageMaxRange      = 31 * 24 * time.Hour
)

var proxyUsageMessageRedactors = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)([^/\s:@]+):([^/\s@]+)@`), `${1}${2}:***@`},
	{regexp.MustCompile(`(?i)((?:proxy-authorization|authorization|cookie|x-api-key|api-key):\s*)[^\r\n]+`), `${1}***`},
	{regexp.MustCompile(`(?i)((?:cookie|token|api[_-]?key|password)=)[^&\s]+`), `${1}***`},
	{regexp.MustCompile(`(?i)("(?:cookie|token|api[_-]?key|password)"\s*:\s*")[^"]*(")`), `${1}***${2}`},
}

type ProxySourceStatus struct {
	Healthy                   bool
	Mode                      string
	Message                   string
	AvailableManualProxyCount int64
	DynamicConfigured         bool
	CheckedAt                 time.Time
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

	proxy, err := s.repo.AcquireAvailableProxy(ctx, protocol, region)
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
		valid, reason, err := s.isBindingReusable(ctx, existing)
		if err != nil {
			return nil, err
		}
		if valid {
			return existing, nil
		}
		if err := s.releaseBinding(ctx, taskID, models.TaskProxyBindStatusExpired, reason); err != nil {
			return nil, err
		}
	}

	policy, err := s.policyRepo.GetEffectivePolicy(ctx, platform)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, errors.New("no active proxy source policy")
	}

	var excludedProxyID *int64
	if existing != nil && existing.ProxyID != nil {
		excludedProxyID = existing.ProxyID
	}

	result, degraded, degradeReason, err := s.acquireWithPolicy(ctx, policy, protocol, region, excludedProxyID)
	if err != nil {
		return nil, err
	}

	binding := s.buildBinding(taskID, policy, result, protocol, region, platform, degraded, degradeReason)
	if existing != nil {
		if err := s.bindingRepo.UpdateBinding(ctx, binding); err != nil {
			return nil, err
		}
		return s.bindingRepo.GetByTaskID(ctx, taskID)
	}

	if err := s.bindingRepo.CreateIfAbsent(ctx, binding); err != nil {
		if errors.Is(err, repository.ErrTaskProxyBindingAlreadyExists) {
			return s.bindingRepo.GetByTaskID(ctx, taskID)
		}
		return nil, err
	}

	return s.bindingRepo.GetByTaskID(ctx, taskID)
}

// ReportUsage 报告代理使用结果
func (s *ProxyService) ReportUsage(ctx context.Context, taskID, leaseID, stage string, success bool, errorCategory, errorMessage string) error {
	if errorCategory == "" && !success {
		errorCategory = models.ErrorCategoryUnknown
	}
	errorMessage = sanitizeProxyUsageErrorMessage(errorMessage)

	if taskID != "" {
		binding, err := s.bindingRepo.GetByTaskID(ctx, taskID)
		if err != nil {
			return err
		}
		if binding != nil {
			platform := ""
			if binding.Platform != nil {
				platform = *binding.Platform
			}
			sourceType := string(binding.SourceType)
			if err := s.repo.RecordUsageEvent(ctx, taskID, binding.ProxyID, leaseID, sourceType, stage, platform, success, errorCategory, errorMessage); err != nil {
				return err
			}
			if err := s.bindingRepo.UpdateReport(ctx, taskID, stage, success, errorCategory); err != nil {
				return err
			}
			if binding.SourceType == models.ProxySourceTypeManualPool && binding.ProxyID != nil {
				if err := s.repo.UpdateUsage(ctx, *binding.ProxyID, success, errorCategory, riskDelta(errorCategory), cooldownUntil(errorCategory)); err != nil {
					return err
				}
			}
			if err := s.repo.UpdatePlatformRisk(ctx, platform, errorCategory, timePtr(time.Now().Add(5*time.Minute))); err != nil {
				return err
			}
			if !success && stage == proxyUsageStageParse {
				if err := s.releaseBinding(ctx, taskID, models.TaskProxyBindStatusFailed, errorCategory); err != nil {
					return err
				}
				return s.bindingRepo.MarkFailed(ctx, taskID, errorCategory)
			}
			if stage == proxyUsageStageDownload {
				return s.releaseBinding(ctx, taskID, models.TaskProxyBindStatusReleased, "download reported")
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
	return s.repo.UpdateUsage(ctx, proxyID, success, errorCategory, riskDelta(errorCategory), cooldownUntil(errorCategory))
}

func (s *ProxyService) ReleaseProxyForTask(ctx context.Context, taskID, reason string) error {
	if taskID == "" {
		return errors.New("task id is required")
	}
	if reason == "" {
		reason = "released"
	}
	return s.releaseBinding(ctx, taskID, models.TaskProxyBindStatusReleased, reason)
}

func (s *ProxyService) ListUsageEvents(ctx context.Context, filter models.ProxyUsageEventFilter) (*models.ProxyUsageEventResult, error) {
	filter = normalizeProxyUsageEventFilter(filter)

	result, err := s.repo.ListUsageEvents(ctx, filter)
	if err != nil {
		return nil, err
	}

	for i := range result.Events {
		result.Events[i].ErrorMessage = sanitizeProxyUsageErrorMessage(result.Events[i].ErrorMessage)
	}
	return result, nil
}

func (s *ProxyService) CheckSourceStatus(ctx context.Context, protocol *models.ProxyProtocol, region *string) (*ProxySourceStatus, error) {
	count, err := s.repo.CountSelectableProxies(ctx, protocol, region)
	if err != nil {
		return nil, err
	}

	dynamicConfigured := s.provider != nil && s.provider.Enabled()
	healthy := dynamicConfigured || count > 0
	message := "manual proxy pool has available capacity"
	mode := string(models.ProxySourceTypeManualPool)
	if dynamicConfigured {
		mode = string(models.ProxySourceTypeDynamicAPI)
		message = "dynamic proxy source configured"
	} else if count == 0 {
		message = "no selectable proxy source"
	}

	return &ProxySourceStatus{
		Healthy:                   healthy,
		Mode:                      mode,
		Message:                   message,
		AvailableManualProxyCount: count,
		DynamicConfigured:         dynamicConfigured,
		CheckedAt:                 time.Now(),
	}, nil
}

func (s *ProxyService) acquireWithPolicy(
	ctx context.Context,
	policy *models.ProxySourcePolicy,
	protocol *models.ProxyProtocol,
	region *string,
	excludedProxyID *int64,
) (*models.ProxyAcquireResult, bool, *string, error) {
	result, err := s.trySource(ctx, policy.PrimarySource, policy, protocol, region, excludedProxyID)
	if err == nil {
		return result, false, nil, nil
	}

	if !policy.FallbackEnabled || policy.FallbackSource == nil || *policy.FallbackSource == "" {
		return nil, false, nil, err
	}

	degradeReason := err.Error()
	fallbackType := models.ProxySourceType(*policy.FallbackSource)
	result, err = s.trySource(ctx, fallbackType, policy, protocol, region, excludedProxyID)
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
	excludedProxyID *int64,
) (*models.ProxyAcquireResult, error) {
	switch sourceType {
	case models.ProxySourceTypeDynamicAPI:
		return s.tryDynamicSource(ctx, policy)
	case models.ProxySourceTypeManualPool:
		return s.tryManualSource(ctx, protocol, region, excludedProxyID)
	default:
		return nil, fmt.Errorf("unsupported proxy source type: %s", sourceType)
	}
}

func (s *ProxyService) tryDynamicSource(ctx context.Context, policy *models.ProxySourcePolicy) (*models.ProxyAcquireResult, error) {
	if s.provider == nil || !s.provider.Enabled() {
		return nil, errors.New("dynamic proxy API is not configured")
	}

	lease, err := s.provider.GetLeaseWithPolicy(ctx, policy.DynamicTimeoutMS, policy.DynamicRetryCount, policy.DynamicCircuitBreakerSec)
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
	excludedProxyID *int64,
) (*models.ProxyAcquireResult, error) {
	proxy, err := s.repo.AcquireTaskProxyExcluding(ctx, protocol, region, excludedProxyID)
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

func (s *ProxyService) isBindingReusable(ctx context.Context, binding *models.TaskProxyBinding) (bool, string, error) {
	if binding == nil {
		return false, "missing binding", nil
	}
	now := time.Now()
	if binding.ReleasedAt != nil {
		return false, "binding released", nil
	}
	if binding.ExpireAt != nil && !binding.ExpireAt.After(now) {
		return false, "binding expired", nil
	}
	if binding.SourceType == models.ProxySourceTypeManualPool && binding.ProxyID != nil {
		usable, err := s.repo.IsUsableForBoundTask(ctx, *binding.ProxyID)
		if err != nil {
			return false, "", err
		}
		if !usable {
			return false, "manual proxy no longer selectable", nil
		}
	}
	return true, "", nil
}

func (s *ProxyService) releaseBinding(ctx context.Context, taskID string, status models.TaskProxyBindStatus, reason string) error {
	released, changed, err := s.bindingRepo.ReleaseBound(ctx, taskID, status, reason)
	if err != nil {
		return err
	}
	if !changed || released == nil {
		return nil
	}
	if released.SourceType == models.ProxySourceTypeManualPool && released.ProxyID != nil {
		return s.repo.ReleaseActiveTask(ctx, *released.ProxyID)
	}
	return nil
}

func riskDelta(errorCategory string) int {
	switch errorCategory {
	case models.ErrorCategoryNetworkTimeout, models.ErrorCategoryProxyUnreachable:
		return 10
	case models.ErrorCategoryRateLimited:
		return 30
	case models.ErrorCategoryProxyAuth:
		return 40
	case models.ErrorCategoryBotDetected:
		return 50
	default:
		return 5
	}
}

func cooldownUntil(errorCategory string) *time.Time {
	var duration time.Duration
	switch errorCategory {
	case models.ErrorCategoryNetworkTimeout, models.ErrorCategoryProxyUnreachable:
		duration = 10 * time.Minute
	case models.ErrorCategoryProxyAuth:
		duration = 60 * time.Minute
	case models.ErrorCategoryRateLimited:
		duration = 30 * time.Minute
	case models.ErrorCategoryBotDetected:
		duration = 60 * time.Minute
	default:
		duration = 5 * time.Minute
	}
	return timePtr(time.Now().Add(duration))
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func normalizeProxyUsageEventFilter(filter models.ProxyUsageEventFilter) models.ProxyUsageEventFilter {
	now := time.Now()
	if filter.EndTime.IsZero() {
		filter.EndTime = now
	}
	if filter.StartTime.IsZero() {
		filter.StartTime = filter.EndTime.Add(-24 * time.Hour)
	}
	if filter.StartTime.After(filter.EndTime) {
		filter.StartTime = filter.EndTime.Add(-24 * time.Hour)
	}
	if filter.EndTime.Sub(filter.StartTime) > proxyUsageMaxRange {
		filter.StartTime = filter.EndTime.Add(-proxyUsageMaxRange)
	}
	if filter.Page <= 0 {
		filter.Page = models.ProxyUsageDefaultPage
	}
	if filter.PageSize <= 0 {
		filter.PageSize = models.ProxyUsageDefaultPerPage
	}
	if filter.PageSize > models.ProxyUsageMaxPerPage {
		filter.PageSize = models.ProxyUsageMaxPerPage
	}
	filter.SortOrder = strings.ToLower(filter.SortOrder)
	if filter.SortOrder != models.ProxyUsageSortOrderAsc {
		filter.SortOrder = models.ProxyUsageSortOrderDesc
	}
	filter.Success = strings.ToLower(filter.Success)
	if filter.Success != models.ProxyUsageSuccessOnly && filter.Success != models.ProxyUsageSuccessFailed {
		filter.Success = models.ProxyUsageSuccessAll
	}
	switch strings.ToLower(filter.SourceType) {
	case "manual":
		filter.SourceType = string(models.ProxySourceTypeManualPool)
	case "dynamic":
		filter.SourceType = string(models.ProxySourceTypeDynamicAPI)
	}
	return filter
}

func sanitizeProxyUsageErrorMessage(message string) string {
	if message == "" {
		return ""
	}
	sanitized := message
	for _, redactor := range proxyUsageMessageRedactors {
		sanitized = redactor.pattern.ReplaceAllString(sanitized, redactor.replacement)
	}
	return sanitized
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
func (s *ProxyService) ListProxies(ctx context.Context, filter models.ProxyListFilter) (*models.ProxyListResult, error) {
	if filter.Page < 1 {
		filter.Page = models.ProxyListDefaultPage
	}
	if filter.Page > models.ProxyListMaxPage {
		filter.Page = models.ProxyListMaxPage
	}
	if filter.PageSize < 1 {
		filter.PageSize = models.ProxyListDefaultPageSize
	}
	if filter.PageSize > models.ProxyListMaxPageSize {
		filter.PageSize = models.ProxyListMaxPageSize
	}
	return s.repo.ListProxies(ctx, filter)
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

// IsAlreadyExistsError 判断是否为唯一约束冲突
func (s *ProxyService) IsAlreadyExistsError(err error) bool {
	return errors.Is(err, repository.ErrProxyAlreadyExists)
}
