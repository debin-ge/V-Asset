package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"vasset/asset-service/internal/models"
)

// ProxyRepository 代理数据访问层
type ProxyRepository struct {
	db *sql.DB
}

// NewProxyRepository 创建代理仓库
func NewProxyRepository(db *sql.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

// UpdateUsage 更新使用统计
func (r *ProxyRepository) UpdateUsage(ctx context.Context, id int64, success bool) error {
	var query string
	if success {
		query = `
			UPDATE proxies 
			SET success_count = success_count + 1, last_used_at = $2, updated_at = $3
			WHERE id = $1`
	} else {
		query = `
			UPDATE proxies 
			SET fail_count = fail_count + 1, last_used_at = $2, updated_at = $3
			WHERE id = $1`
	}

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, now, now)
	if err != nil {
		return fmt.Errorf("update usage failed: %w", err)
	}

	return nil
}

// GetAvailableProxy 获取一个可用的代理（轮询策略：最久未使用的活跃代理）
func (r *ProxyRepository) GetAvailableProxy(ctx context.Context, protocol *models.ProxyProtocol, region *string) (*models.Proxy, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
	args = append(args, models.ProxyStatusActive)
	argIdx++

	if protocol != nil {
		conditions = append(conditions, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, *protocol)
		argIdx++
	}

	if region != nil {
		conditions = append(conditions, fmt.Sprintf("region = $%d", argIdx))
		args = append(args, *region)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT id, ip, port, username, password, protocol, region, status,
		       last_check_at, last_check_result, success_count, fail_count,
		       last_used_at, created_at, updated_at
		FROM proxies
		WHERE %s
		ORDER BY last_used_at ASC NULLS FIRST, success_count DESC
		LIMIT 1`, strings.Join(conditions, " AND "))

	proxy := &models.Proxy{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&proxy.ID, &proxy.IP, &proxy.Port, &proxy.Username, &proxy.Password,
		&proxy.Protocol, &proxy.Region, &proxy.Status, &proxy.LastCheckAt,
		&proxy.LastCheckResult, &proxy.SuccessCount, &proxy.FailCount,
		&proxy.LastUsedAt, &proxy.CreatedAt, &proxy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get available proxy failed: %w", err)
	}

	return proxy, nil
}
