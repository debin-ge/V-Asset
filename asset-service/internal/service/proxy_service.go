package service

import (
	"context"
	"errors"
	"fmt"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/models"
	dynamicproxy "vasset/asset-service/internal/proxy"
	"vasset/asset-service/internal/repository"
)

// ProxyService 代理业务逻辑层
type ProxyService struct {
	repo     *repository.ProxyRepository
	provider *dynamicproxy.Provider
}

// NewProxyService 创建代理服务
func NewProxyService(repo *repository.ProxyRepository, cfg *config.Config) *ProxyService {
	return &ProxyService{
		repo:     repo,
		provider: dynamicproxy.NewProvider(&cfg.Proxy),
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

// ReportUsage 报告代理使用结果
func (s *ProxyService) ReportUsage(ctx context.Context, leaseID string, success bool) error {
	if leaseID == "" {
		return errors.New("proxy lease id is required")
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
