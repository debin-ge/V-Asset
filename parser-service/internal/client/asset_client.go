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

	pb "vasset/parser-service/proto"
)

// AssetClient Asset服务客户端
type AssetClient struct {
	conn          *grpc.ClientConn
	client        pb.AssetServiceClient
	timeout       time.Duration
	cookieTempDir string
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
func (c *AssetClient) GetAvailableProxy() (string, int64, error) {
	log.Printf("[AssetClient] Requesting available proxy from Asset Service...")

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.client.GetAvailableProxy(ctx, &pb.GetAvailableProxyRequest{})
	if err != nil {
		log.Printf("[AssetClient] ERROR: Failed to get proxy: %v", err)
		return "", 0, fmt.Errorf("failed to get proxy: %w", err)
	}

	if resp.ProxyUrl == "" {
		log.Printf("[AssetClient] No proxy available, using direct connection")
		return "", 0, nil
	}

	log.Printf("[AssetClient] ✓ Got proxy: %s (ID: %d)", resp.ProxyUrl, resp.ProxyId)
	return resp.ProxyUrl, resp.ProxyId, nil
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
