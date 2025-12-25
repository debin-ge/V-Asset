package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/repository"
)

// ProxyService 代理业务逻辑层
type ProxyService struct {
	repo   *repository.ProxyRepository
	cfg    *config.Config
	client *http.Client
}

// NewProxyService 创建代理服务
func NewProxyService(repo *repository.ProxyRepository, cfg *config.Config) *ProxyService {
	return &ProxyService{
		repo: repo,
		cfg:  cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.Proxy.HealthCheckTimeout) * time.Second,
		},
	}
}

// Create 创建代理
func (s *ProxyService) Create(ctx context.Context, proxy *models.Proxy, checkHealth bool) (int64, bool, string, error) {
	// 设置默认协议
	if proxy.Protocol == "" {
		proxy.Protocol = models.ProxyProtocolHTTP
	}

	// 初始状态
	proxy.Status = models.ProxyStatusActive

	// 如果需要健康检查
	var healthCheckPassed bool
	var healthCheckError string
	if checkHealth {
		healthy, err := s.CheckHealth(ctx, proxy)
		if err != nil {
			healthCheckError = err.Error()
			proxy.Status = models.ProxyStatusInactive
		} else {
			healthCheckPassed = healthy
			if !healthy {
				healthCheckError = "health check failed"
				proxy.Status = models.ProxyStatusInactive
			}
		}
	}

	id, err := s.repo.Create(ctx, proxy)
	if err != nil {
		return 0, false, "", err
	}

	return id, healthCheckPassed, healthCheckError, nil
}

// GetByID 获取代理
func (s *ProxyService) GetByID(ctx context.Context, id int64) (*models.Proxy, error) {
	return s.repo.GetByID(ctx, id)
}

// Update 更新代理
func (s *ProxyService) Update(ctx context.Context, proxy *models.Proxy) error {
	return s.repo.Update(ctx, proxy)
}

// Delete 删除代理
func (s *ProxyService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// List 列表查询
func (s *ProxyService) List(ctx context.Context, filter *models.ProxyFilter) (*models.ProxyResult, error) {
	// 设置分页默认值
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = s.cfg.Pagination.DefaultPageSize
	}
	if filter.PageSize > s.cfg.Pagination.MaxPageSize {
		filter.PageSize = s.cfg.Pagination.MaxPageSize
	}

	return s.repo.List(ctx, filter)
}

// CheckHealth 检查代理健康状态
func (s *ProxyService) CheckHealth(ctx context.Context, proxy *models.Proxy) (bool, error) {
	proxyURL := proxy.GetURL()

	parsedProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		return false, fmt.Errorf("invalid proxy URL: %w", err)
	}

	// 创建使用代理的 HTTP 客户端
	transport := &http.Transport{
		Proxy: http.ProxyURL(parsedProxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(s.cfg.Proxy.HealthCheckTimeout) * time.Second,
	}

	// 发起测试请求
	req, err := http.NewRequestWithContext(ctx, "HEAD", s.cfg.Proxy.TestURL, nil)
	if err != nil {
		return false, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	// HTTP 2xx 或 3xx 认为健康
	return resp.StatusCode >= 200 && resp.StatusCode < 400, nil
}

// CheckHealthByID 根据 ID 检查代理健康
func (s *ProxyService) CheckHealthByID(ctx context.Context, id int64) (bool, int64, error) {
	proxy, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return false, 0, err
	}
	if proxy == nil {
		return false, 0, errors.New("proxy not found")
	}

	startTime := time.Now()
	healthy, checkErr := s.CheckHealth(ctx, proxy)
	latency := time.Since(startTime).Milliseconds()

	// 更新健康检查结果
	var status models.ProxyStatus
	var result string
	if checkErr != nil {
		status = models.ProxyStatusInactive
		result = checkErr.Error()
	} else if healthy {
		status = models.ProxyStatusActive
		result = "ok"
	} else {
		status = models.ProxyStatusInactive
		result = "unhealthy"
	}

	if err := s.repo.UpdateHealthCheck(ctx, id, status, result); err != nil {
		return false, 0, err
	}

	return healthy, latency, checkErr
}

// GetAvailableProxy 获取可用代理
func (s *ProxyService) GetAvailableProxy(ctx context.Context, protocol *models.ProxyProtocol, region *string) (string, int64, error) {
	proxy, err := s.repo.GetAvailableProxy(ctx, protocol, region)
	if err != nil {
		return "", 0, err
	}
	if proxy == nil {
		return "", 0, errors.New("no available proxy")
	}

	return proxy.GetURL(), proxy.ID, nil
}

// ReportUsage 报告代理使用结果
func (s *ProxyService) ReportUsage(ctx context.Context, id int64, success bool) error {
	return s.repo.UpdateUsage(ctx, id, success)
}
