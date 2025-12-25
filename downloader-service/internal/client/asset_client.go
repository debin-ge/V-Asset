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

	pb "vasset/downloader-service/proto"
)

// AssetClient Asset 服务客户端
type AssetClient struct {
	conn          *grpc.ClientConn
	client        pb.AssetServiceClient
	timeout       time.Duration
	cookieTempDir string
}

// NewAssetClient 创建 Asset 客户端
func NewAssetClient(addr string, timeout time.Duration, cookieTempDir string) (*AssetClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to asset service: %w", err)
	}

	// 确保临时目录存在
	if err := os.MkdirAll(cookieTempDir, 0755); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create cookie temp dir: %w", err)
	}

	log.Printf("[AssetClient] Connected to Asset Service: %s", addr)
	return &AssetClient{
		conn:          conn,
		client:        pb.NewAssetServiceClient(conn),
		timeout:       timeout,
		cookieTempDir: cookieTempDir,
	}, nil
}

// GetCookieContent 通过 ID 获取 cookie 内容并写入临时文件
func (c *AssetClient) GetCookieContent(cookieID int64, platform, taskID string) (string, error) {
	if cookieID <= 0 {
		return "", nil
	}

	log.Printf("[AssetClient] [Task %s] Getting cookie content for ID: %d", taskID, cookieID)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.client.GetCookie(ctx, &pb.GetCookieRequest{
		Id: cookieID,
	})
	if err != nil {
		log.Printf("[AssetClient] [Task %s] ERROR: Failed to get cookie: %v", taskID, err)
		return "", fmt.Errorf("failed to get cookie: %w", err)
	}

	if resp.Cookie == nil || resp.Cookie.Content == "" {
		log.Printf("[AssetClient] [Task %s] Cookie content is empty for ID: %d", taskID, cookieID)
		return "", nil
	}

	// 创建临时 cookie 文件
	cookieFile := filepath.Join(c.cookieTempDir, fmt.Sprintf("%s_%s_%d.txt", platform, taskID, cookieID))
	if err := os.WriteFile(cookieFile, []byte(resp.Cookie.Content), 0600); err != nil {
		log.Printf("[AssetClient] [Task %s] ERROR: Failed to write cookie file: %v", taskID, err)
		return "", fmt.Errorf("failed to write cookie file: %w", err)
	}

	log.Printf("[AssetClient] [Task %s] Cookie written to: %s (length: %d bytes)", taskID, cookieFile, len(resp.Cookie.Content))
	return cookieFile, nil
}

// ReportCookieUsage 报告 Cookie 使用结果
func (c *AssetClient) ReportCookieUsage(cookieID int64, success bool, taskID string) error {
	if cookieID <= 0 {
		return nil
	}

	log.Printf("[AssetClient] [Task %s] Reporting cookie usage: ID=%d, success=%v", taskID, cookieID, success)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := c.client.ReportCookieUsage(ctx, &pb.ReportCookieUsageRequest{
		CookieId: cookieID,
		Success:  success,
	})

	if err != nil {
		log.Printf("[AssetClient] [Task %s] ERROR: Failed to report usage: %v", taskID, err)
	} else {
		log.Printf("[AssetClient] [Task %s] Successfully reported cookie usage", taskID)
	}

	return err
}

// CleanupCookieFile 清理临时 cookie 文件
func (c *AssetClient) CleanupCookieFile(cookieFile string) error {
	if cookieFile == "" {
		return nil
	}

	log.Printf("[AssetClient] Cleaning up cookie file: %s", cookieFile)
	err := os.Remove(cookieFile)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("[AssetClient] ERROR: Failed to cleanup cookie file: %v", err)
		return err
	}
	return nil
}

// GetProxyWithRetry 从 Asset Service 获取可用代理（实现 ProxyProviderInterface）
func (c *AssetClient) GetProxyWithRetry(ctx context.Context) (string, error) {
	log.Printf("[AssetClient] Requesting available proxy from Asset Service...")

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetAvailableProxy(ctx, &pb.GetAvailableProxyRequest{})
	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to get proxy: %v", err)
		return "", fmt.Errorf("failed to get proxy: %w", err)
	}

	if resp.ProxyUrl == "" {
		log.Printf("[AssetClient] No proxy available, using direct connection")
		return "", nil
	}

	log.Printf("[AssetClient] ✓ Got proxy: %s (ID: %d)", resp.ProxyUrl, resp.ProxyId)
	return resp.ProxyUrl, nil
}

// ReportProxyUsage 报告代理使用结果
func (c *AssetClient) ReportProxyUsage(proxyID int64, success bool) error {
	if proxyID <= 0 {
		return nil
	}

	log.Printf("[AssetClient] Reporting proxy usage: ID=%d, success=%v", proxyID, success)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := c.client.ReportProxyUsage(ctx, &pb.ReportProxyUsageRequest{
		ProxyId: proxyID,
		Success: success,
	})

	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to report proxy usage: %v", err)
	} else {
		log.Printf("[AssetClient] Successfully reported proxy usage")
	}

	return err
}

// Close 关闭连接
func (c *AssetClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
