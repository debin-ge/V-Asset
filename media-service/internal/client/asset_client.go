package client

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"vasset/media-service/internal/redact"
	pb "vasset/media-service/proto"
)

// AssetClient Asset服务客户端
type AssetClient struct {
	conn          *grpc.ClientConn
	client        pb.AssetServiceClient
	timeout       time.Duration
	cookieTempDir string
}

// ProxyLease 表示一次动态代理租约。
type ProxyLease struct {
	URL      string
	LeaseID  string
	ExpireAt string
}

// NewAssetClient 创建Asset客户端
func NewAssetClient(addr string, timeout int, cookieTempDir string) (*AssetClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to asset service: %w", err)
	}

	// 确保临时目录存在
	if err := os.MkdirAll(cookieTempDir, 0755); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create cookie temp dir: %w", err)
	}

	return &AssetClient{
		conn:          conn,
		client:        pb.NewAssetServiceClient(conn),
		timeout:       time.Duration(timeout) * time.Second,
		cookieTempDir: cookieTempDir,
	}, nil
}

// GetAvailableCookie 获取可用Cookie并写入临时文件，返回cookie文件路径和ID
func (c *AssetClient) GetAvailableCookie(platform string) (string, int64, error) {
	log.Printf("[AssetClient] Requesting cookie for platform: %s", platform)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.client.GetAvailableCookie(ctx, &pb.GetAvailableCookieRequest{
		Platform: platform,
	})
	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to get cookie: %v", err)
		return "", 0, fmt.Errorf("failed to get cookie: %w", err)
	}

	if resp.CookieId == 0 || resp.Content == "" {
		log.Printf("[AssetClient] No cookie available for platform: %s", platform)
		return "", 0, nil // 没有可用cookie
	}

	log.Printf("[AssetClient] Got cookie ID=%d, content length=%d bytes", resp.CookieId, len(resp.Content))

	// 创建临时cookie文件
	cookieFile := filepath.Join(c.cookieTempDir, fmt.Sprintf("%s_%d.txt", platform, resp.CookieId))
	if err := os.WriteFile(cookieFile, []byte(resp.Content), 0600); err != nil {
		log.Printf("[AssetClient] ERROR: Failed to write cookie file: %v", err)
		return "", 0, fmt.Errorf("failed to write cookie file: %w", err)
	}

	log.Printf("[AssetClient] Cookie written to: %s", cookieFile)
	return cookieFile, resp.CookieId, nil
}

// ReportCookieUsage 报告Cookie使用结果
func (c *AssetClient) ReportCookieUsage(cookieID int64, success bool) error {
	log.Printf("[AssetClient] Reporting cookie usage: ID=%d, success=%v", cookieID, success)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := c.client.ReportCookieUsage(ctx, &pb.ReportCookieUsageRequest{
		CookieId: cookieID,
		Success:  success,
	})

	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to report usage: %v", err)
	} else {
		log.Printf("[AssetClient] Successfully reported cookie usage")
	}

	return err
}

// Close 关闭连接
func (c *AssetClient) Close() error {
	return c.conn.Close()
}

// CleanupCookieFile 清理cookie文件
func (c *AssetClient) CleanupCookieFile(cookieFile string) error {
	if cookieFile == "" {
		return nil
	}

	log.Printf("[AssetClient] Cleaning up cookie file: %s", cookieFile)
	err := os.Remove(cookieFile)
	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to cleanup cookie file: %v", err)
	} else {
		log.Printf("[AssetClient] Cookie file cleaned up successfully")
	}
	return err
}

// GetAvailableProxy 从 Asset Service 获取可用代理
func (c *AssetClient) GetAvailableProxy() (*ProxyLease, error) {
	log.Printf("[AssetClient] Requesting available proxy from Asset Service...")

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.client.GetAvailableProxy(ctx, &pb.GetAvailableProxyRequest{})
	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to get proxy: %v", err)
		return nil, fmt.Errorf("failed to get proxy: %w", err)
	}

	if resp.ProxyUrl == "" {
		log.Printf("[AssetClient] No proxy available, using direct connection")
		return nil, nil
	}

	log.Printf("[AssetClient] ✓ Got proxy lease: %s (lease_id=%s, expire_at=%s)", redact.ProxyURL(resp.ProxyUrl), resp.ProxyLeaseId, resp.ExpireAt)
	return &ProxyLease{
		URL:      resp.ProxyUrl,
		LeaseID:  resp.ProxyLeaseId,
		ExpireAt: resp.ExpireAt,
	}, nil
}

// AcquireProxyForTask 为指定任务获取或复用代理
func (c *AssetClient) AcquireProxyForTask(ctx context.Context, taskID, platform string) (*ProxyLease, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}

	log.Printf("[AssetClient] Requesting task-bound proxy from Asset Service: task_id=%s, platform=%s", taskID, platform)

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.AcquireProxyForTask(ctx, &pb.AcquireProxyForTaskRequest{
		TaskId:   taskID,
		Platform: platform,
	})
	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to acquire proxy for task %s: %v", taskID, err)
		return nil, fmt.Errorf("failed to acquire proxy for task: %w", err)
	}

	if resp.ProxyUrl == "" {
		log.Printf("[AssetClient] No task-bound proxy available for task %s, using direct connection", taskID)
		return nil, nil
	}

	log.Printf("[AssetClient] ✓ Got task-bound proxy: task_id=%s, proxy=%s, lease_id=%s, expire_at=%s", taskID, redact.ProxyURL(resp.ProxyUrl), resp.ProxyLeaseId, resp.ExpireAt)
	return &ProxyLease{
		URL:      resp.ProxyUrl,
		LeaseID:  resp.ProxyLeaseId,
		ExpireAt: resp.ExpireAt,
	}, nil
}

// ReportProxyUsage 报告代理使用结果
func (c *AssetClient) ReportProxyUsage(taskID, proxyLeaseID, stage string, success bool) error {
	if taskID == "" && proxyLeaseID == "" {
		return nil
	}

	log.Printf("[AssetClient] Reporting proxy usage: task_id=%s, lease_id=%s, stage=%s, success=%v", taskID, proxyLeaseID, stage, success)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := c.client.ReportProxyUsage(ctx, &pb.ReportProxyUsageRequest{
		ProxyLeaseId: proxyLeaseID,
		Success:      success,
		TaskId:       taskID,
		Stage:        stage,
	})

	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to report proxy usage: %v", err)
	} else {
		log.Printf("[AssetClient] Successfully reported proxy usage")
	}

	return err
}
