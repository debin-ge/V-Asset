package health

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"vasset/proxy-service/internal/config"
)

// Status 健康状态
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusUnknown   Status = "unknown"
)

// Checker 健康检查器
type Checker struct {
	baseURL    string
	apiKey     string
	interval   time.Duration
	logger     *zap.Logger
	httpClient *http.Client

	mu           sync.RWMutex
	status       Status
	lastCheck    time.Time
	lastError    string
	responseTime time.Duration

	stopCh chan struct{}
}

// NewChecker 创建健康检查器
func NewChecker(cfg *config.YtDLPAPIConfig, healthCfg *config.HealthCheckConfig, logger *zap.Logger) *Checker {
	return &Checker{
		baseURL:  cfg.BaseURL,
		apiKey:   cfg.APIKey,
		interval: healthCfg.GetInterval(),
		logger:   logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		status: StatusUnknown,
		stopCh: make(chan struct{}),
	}
}

// Start 启动定时检查
func (c *Checker) Start() {
	c.logger.Info("Starting health checker",
		zap.String("base_url", c.baseURL),
		zap.Duration("interval", c.interval))

	// 立即执行一次检查
	c.check()

	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.check()
			case <-c.stopCh:
				c.logger.Info("Health checker stopped")
				return
			}
		}
	}()
}

// Stop 停止检查
func (c *Checker) Stop() {
	close(c.stopCh)
}

// check 执行一次健康检查
func (c *Checker) check() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		c.updateStatus(StatusUnhealthy, 0, fmt.Sprintf("failed to create request: %v", err))
		return
	}

	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		c.updateStatus(StatusUnhealthy, responseTime, fmt.Sprintf("request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.updateStatus(StatusHealthy, responseTime, "")
	} else {
		c.updateStatus(StatusUnhealthy, responseTime, fmt.Sprintf("unhealthy status code: %d", resp.StatusCode))
	}
}

// updateStatus 更新状态
func (c *Checker) updateStatus(status Status, responseTime time.Duration, errMsg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prevStatus := c.status
	c.status = status
	c.lastCheck = time.Now()
	c.responseTime = responseTime
	c.lastError = errMsg

	if status == StatusHealthy {
		if prevStatus != StatusHealthy {
			c.logger.Info("✓ YtDLP API is healthy",
				zap.Duration("response_time", responseTime))
		}
	} else {
		c.logger.Warn("✗ YtDLP API is unhealthy",
			zap.String("error", errMsg),
			zap.Duration("response_time", responseTime))
	}
}

// GetStatus 获取当前状态
func (c *Checker) GetStatus() (Status, time.Time, time.Duration, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status, c.lastCheck, c.responseTime, c.lastError
}

// IsHealthy 是否健康
func (c *Checker) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status == StatusHealthy
}
