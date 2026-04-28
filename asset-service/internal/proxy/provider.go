package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"youdlp/asset-service/internal/config"
)

// Lease 表示一次动态代理租约。
type Lease struct {
	URL      string
	LeaseID  string
	ExpireAt string
}

// APIResponse 是第三方代理 API 的默认响应格式。
type APIResponse struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	ExpireAt string `json:"expire_at"`
}

// Provider 负责实时拉取动态代理。
type Provider struct {
	apiKey       string
	endpoint     string
	client       *http.Client
	retryCount   int
	mu           sync.Mutex
	circuitUntil time.Time
}

// NewProvider 创建动态代理提供者。
func NewProvider(cfg *config.ProxyConfig) *Provider {
	return &Provider{
		apiKey:   cfg.APIKey,
		endpoint: cfg.APIEndpoint,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		retryCount: cfg.RetryCount,
	}
}

// Enabled 表示是否已配置动态代理 API。
func (p *Provider) Enabled() bool {
	return p != nil && p.endpoint != ""
}

// GetLeaseWithRetry 获取动态代理租约并带重试。
func (p *Provider) GetLeaseWithRetry(ctx context.Context) (*Lease, error) {
	return p.GetLeaseWithPolicy(ctx, 0, p.retryCount, 0)
}

// GetLeaseWithPolicy 获取动态代理租约，并使用策略中的超时、重试和熔断参数。
func (p *Provider) GetLeaseWithPolicy(ctx context.Context, timeoutMS, retryCount, circuitBreakerSec int) (*Lease, error) {
	if !p.Enabled() {
		return nil, fmt.Errorf("dynamic proxy API is not configured")
	}
	if retryCount <= 0 {
		retryCount = p.retryCount
	}
	if retryCount <= 0 {
		retryCount = 1
	}

	p.mu.Lock()
	if !p.circuitUntil.IsZero() && time.Now().Before(p.circuitUntil) {
		until := p.circuitUntil
		p.mu.Unlock()
		return nil, fmt.Errorf("dynamic proxy API circuit open until %s", until.Format(time.RFC3339))
	}
	p.mu.Unlock()

	var lastErr error
	for i := 0; i < retryCount; i++ {
		lease, err := p.getLeaseWithTimeout(ctx, timeoutMS)
		if err == nil {
			return lease, nil
		}

		lastErr = err
		if i == retryCount-1 {
			break
		}

		wait := time.Duration(math.Pow(2, float64(i))) * time.Second
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if circuitBreakerSec > 0 {
		p.mu.Lock()
		p.circuitUntil = time.Now().Add(time.Duration(circuitBreakerSec) * time.Second)
		p.mu.Unlock()
	}
	return nil, fmt.Errorf("failed to get proxy lease after %d attempts: %w", retryCount, lastErr)
}

func (p *Provider) getLeaseWithTimeout(ctx context.Context, timeoutMS int) (*Lease, error) {
	if timeoutMS <= 0 {
		return p.GetLease(ctx)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()
	return p.GetLease(timeoutCtx)
}

// GetLease 获取一次动态代理租约。
func (p *Provider) GetLease(ctx context.Context) (*Lease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("proxy API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy API returned %d", resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode proxy response: %w", err)
	}
	if apiResp.IP == "" || apiResp.Port == 0 {
		return nil, fmt.Errorf("proxy API returned empty endpoint")
	}

	proxyURL := fmt.Sprintf("http://%s:%d", apiResp.IP, apiResp.Port)
	if apiResp.Username != "" && apiResp.Password != "" {
		proxyURL = fmt.Sprintf("http://%s:%s@%s:%d", apiResp.Username, apiResp.Password, apiResp.IP, apiResp.Port)
	}

	hash := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%s|%d", apiResp.IP, apiResp.Username, apiResp.Port, apiResp.ExpireAt, time.Now().UTC().UnixNano())))

	return &Lease{
		URL:      proxyURL,
		LeaseID:  hex.EncodeToString(hash[:16]),
		ExpireAt: apiResp.ExpireAt,
	}, nil
}
