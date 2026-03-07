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

	conditions = append(conditions, "deleted_at IS NULL")

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
		SELECT id, host, ip, port, username, password, protocol, region, priority, platform_tags, remark, status,
		       last_check_at, last_check_result, success_count, fail_count,
		       last_used_at, deleted_at, created_at, updated_at
		FROM proxies
		WHERE %s
		ORDER BY last_used_at ASC NULLS FIRST, priority DESC, success_count DESC
		LIMIT 1`, strings.Join(conditions, " AND "))

	proxy := &models.Proxy{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&proxy.ID, &proxy.Host, &proxy.IP, &proxy.Port, &proxy.Username, &proxy.Password,
		&proxy.Protocol, &proxy.Region, &proxy.Priority, &proxy.PlatformTags, &proxy.Remark, &proxy.Status, &proxy.LastCheckAt,
		&proxy.LastCheckResult, &proxy.SuccessCount, &proxy.FailCount,
		&proxy.LastUsedAt, &proxy.DeletedAt, &proxy.CreatedAt, &proxy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get available proxy failed: %w", err)
	}

	return proxy, nil
}

// ListProxies 列出手动代理池
func (r *ProxyRepository) ListProxies(
	ctx context.Context,
	search, protocol, region *string,
	status *models.ProxyStatus,
) ([]*models.Proxy, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := make([]interface{}, 0)
	argIdx := 1

	if search != nil && *search != "" {
		conditions = append(conditions, fmt.Sprintf("(COALESCE(host, ip) ILIKE $%d OR COALESCE(region, '') ILIKE $%d OR COALESCE(platform_tags, '') ILIKE $%d OR COALESCE(remark, '') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+*search+"%")
		argIdx++
	}
	if protocol != nil && *protocol != "" {
		conditions = append(conditions, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, *protocol)
		argIdx++
	}
	if region != nil && *region != "" {
		conditions = append(conditions, fmt.Sprintf("region = $%d", argIdx))
		args = append(args, *region)
		argIdx++
	}
	if status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *status)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT id, host, ip, port, username, password, protocol, region, priority, platform_tags, remark,
		       status, last_check_at, last_check_result, success_count, fail_count, last_used_at,
		       deleted_at, created_at, updated_at
		FROM proxies
		WHERE %s
		ORDER BY status ASC, priority DESC, created_at DESC`, strings.Join(conditions, " AND "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list proxies failed: %w", err)
	}
	defer rows.Close()

	items := make([]*models.Proxy, 0)
	for rows.Next() {
		item := &models.Proxy{}
		if err := rows.Scan(
			&item.ID,
			&item.Host,
			&item.IP,
			&item.Port,
			&item.Username,
			&item.Password,
			&item.Protocol,
			&item.Region,
			&item.Priority,
			&item.PlatformTags,
			&item.Remark,
			&item.Status,
			&item.LastCheckAt,
			&item.LastCheckResult,
			&item.SuccessCount,
			&item.FailCount,
			&item.LastUsedAt,
			&item.DeletedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan proxy failed: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxies failed: %w", err)
	}

	return items, nil
}

// CreateProxy 创建手动代理
func (r *ProxyRepository) CreateProxy(ctx context.Context, proxy *models.Proxy) (int64, error) {
	query := `
		INSERT INTO proxies (
			host, ip, port, username, password, protocol, region, priority, platform_tags, remark,
			status, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		proxy.Host,
		proxy.IP,
		proxy.Port,
		proxy.Username,
		proxy.Password,
		proxy.Protocol,
		proxy.Region,
		proxy.Priority,
		proxy.PlatformTags,
		proxy.Remark,
		proxy.Status,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create proxy failed: %w", err)
	}
	return id, nil
}

// UpdateProxy 更新手动代理
func (r *ProxyRepository) UpdateProxy(ctx context.Context, proxy *models.Proxy) error {
	query := `
		UPDATE proxies
		SET host = $2,
		    ip = $3,
		    port = $4,
		    username = $5,
		    -- Preserve the existing password when the edit payload leaves it empty.
		    password = COALESCE($6, password),
		    protocol = $7,
		    region = $8,
		    priority = $9,
		    platform_tags = $10,
		    remark = $11,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		  AND deleted_at IS NULL`

	result, err := r.db.ExecContext(
		ctx,
		query,
		proxy.ID,
		proxy.Host,
		proxy.IP,
		proxy.Port,
		proxy.Username,
		proxy.Password,
		proxy.Protocol,
		proxy.Region,
		proxy.Priority,
		proxy.PlatformTags,
		proxy.Remark,
	)
	if err != nil {
		return fmt.Errorf("update proxy failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get update proxy rows affected failed: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateProxyStatus 更新手动代理状态
func (r *ProxyRepository) UpdateProxyStatus(ctx context.Context, id int64, status models.ProxyStatus) error {
	query := `
		UPDATE proxies
		SET status = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		  AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("update proxy status failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get update proxy status rows affected failed: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteProxy 软删除手动代理
func (r *ProxyRepository) DeleteProxy(ctx context.Context, id int64) error {
	query := `
		UPDATE proxies
		SET deleted_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		  AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete proxy failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get delete proxy rows affected failed: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
