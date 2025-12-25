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
	log.Printf("[Proxy] Requesting proxy from API: %s", p.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint, nil)
	if err != nil {
		log.Printf("[Proxy] ❌ Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	log.Printf("[Proxy] Sending request to proxy API...")
	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[Proxy] ❌ API request failed: %v", err)
		return "", fmt.Errorf("proxy API request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[Proxy] API response status: %d", resp.StatusCode)

	if resp.StatusCode != 200 {
		log.Printf("[Proxy] ❌ API returned non-200 status: %d", resp.StatusCode)
		return "", fmt.Errorf("proxy API returned %d", resp.StatusCode)
	}

	var proxyResp ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&proxyResp); err != nil {
		log.Printf("[Proxy] ❌ Failed to decode response: %v", err)
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	log.Printf("[Proxy] ✓ Decoded proxy response: IP=%s, Port=%d", proxyResp.IP, proxyResp.Port)

	// 格式化代理 URL
	var proxyURL string
	if proxyResp.Username != "" && proxyResp.Password != "" {
		proxyURL = fmt.Sprintf("http://%s:%s@%s:%d",
			proxyResp.Username,
			proxyResp.Password,
			proxyResp.IP,
			proxyResp.Port,
		)
		log.Printf("[Proxy] ✓ Formatted proxy URL with auth: %s:***@%s:%d", proxyResp.Username, proxyResp.IP, proxyResp.Port)
	} else {
		proxyURL = fmt.Sprintf("http://%s:%d", proxyResp.IP, proxyResp.Port)
		log.Printf("[Proxy] ✓ Formatted proxy URL without auth: %s:%d", proxyResp.IP, proxyResp.Port)
	}

	log.Printf("[Proxy] ✓ Got proxy IP: %s", proxyResp.IP)
	return proxyURL, nil
}

// GetProxyWithRetry 带重试的获取代理
func (p *Provider) GetProxyWithRetry(ctx context.Context) (string, error) {
	log.Printf("[Proxy] Starting proxy acquisition with retry (max attempts: %d)", p.retryCount)
	var lastErr error

	for i := 0; i < p.retryCount; i++ {
		log.Printf("[Proxy] Attempt %d/%d to get proxy...", i+1, p.retryCount)
		proxy, err := p.GetProxy(ctx)
		if err == nil {
			log.Printf("[Proxy] ✓ Successfully got proxy on attempt %d/%d", i+1, p.retryCount)
			return proxy, nil
		}

		lastErr = err
		log.Printf("[Proxy] ❌ Failed to get proxy (attempt %d/%d): %v", i+1, p.retryCount, err)

		// 指数退避
		if i < p.retryCount-1 {
			waitTime := time.Duration(math.Pow(2, float64(i))) * time.Second
			log.Printf("[Proxy] Waiting %v before next retry...", waitTime)
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				log.Printf("[Proxy] ❌ Context cancelled during retry wait")
				return "", ctx.Err()
			}
		}
	}

	log.Printf("[Proxy] ❌ Failed to get proxy after %d attempts", p.retryCount)
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
