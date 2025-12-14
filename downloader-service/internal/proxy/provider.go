package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"vasset/downloader-service/internal/config"
)

// ProxyResponse 代理 API 响应
type ProxyResponse struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	ExpireAt string `json:"expire_at"`
}

// Provider 代理提供者
type Provider struct {
	apiKey     string
	endpoint   string
	client     *http.Client
	retryCount int
}

// NewProvider 创建代理提供者
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

// GetProxy 获取代理 IP
func (p *Provider) GetProxy(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("proxy API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("proxy API returned %d", resp.StatusCode)
	}

	var proxyResp ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&proxyResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// 格式化代理 URL
	var proxyURL string
	if proxyResp.Username != "" && proxyResp.Password != "" {
		proxyURL = fmt.Sprintf("http://%s:%s@%s:%d",
			proxyResp.Username,
			proxyResp.Password,
			proxyResp.IP,
			proxyResp.Port,
		)
	} else {
		proxyURL = fmt.Sprintf("http://%s:%d", proxyResp.IP, proxyResp.Port)
	}

	log.Printf("[Proxy] Got proxy IP: %s", proxyResp.IP)
	return proxyURL, nil
}

// GetProxyWithRetry 带重试的获取代理
func (p *Provider) GetProxyWithRetry(ctx context.Context) (string, error) {
	var lastErr error

	for i := 0; i < p.retryCount; i++ {
		proxy, err := p.GetProxy(ctx)
		if err == nil {
			return proxy, nil
		}

		lastErr = err
		log.Printf("[Proxy] Failed to get proxy (attempt %d/%d): %v", i+1, p.retryCount, err)

		// 指数退避
		waitTime := time.Duration(math.Pow(2, float64(i))) * time.Second
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return "", fmt.Errorf("failed to get proxy after %d attempts: %w", p.retryCount, lastErr)
}

// MockProvider 模拟代理提供者(用于测试或不需要代理的场景)
type MockProvider struct{}

// NewMockProvider 创建模拟代理提供者
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// GetProxy 返回空代理(直连)
func (p *MockProvider) GetProxy(ctx context.Context) (string, error) {
	return "", nil
}

// GetProxyWithRetry 返回空代理(直连)
func (p *MockProvider) GetProxyWithRetry(ctx context.Context) (string, error) {
	return "", nil
}
