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

// Create 创建代理
func (r *ProxyRepository) Create(ctx context.Context, proxy *models.Proxy) (int64, error) {
	query := `
		INSERT INTO proxies (ip, port, username, password, protocol, region, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	now := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		proxy.IP,
		proxy.Port,
		proxy.Username,
		proxy.Password,
		proxy.Protocol,
		proxy.Region,
		proxy.Status,
		now,
		now,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("create proxy failed: %w", err)
	}

	return id, nil
}

// GetByID 根据 ID 获取代理
func (r *ProxyRepository) GetByID(ctx context.Context, id int64) (*models.Proxy, error) {
	query := `
		SELECT id, ip, port, username, password, protocol, region, status,
		       last_check_at, last_check_result, success_count, fail_count,
		       last_used_at, created_at, updated_at
		FROM proxies WHERE id = $1`

	proxy := &models.Proxy{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&proxy.ID,
		&proxy.IP,
		&proxy.Port,
		&proxy.Username,
		&proxy.Password,
		&proxy.Protocol,
		&proxy.Region,
		&proxy.Status,
		&proxy.LastCheckAt,
		&proxy.LastCheckResult,
		&proxy.SuccessCount,
		&proxy.FailCount,
		&proxy.LastUsedAt,
		&proxy.CreatedAt,
		&proxy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get proxy by id failed: %w", err)
	}

	return proxy, nil
}

// Update 更新代理
func (r *ProxyRepository) Update(ctx context.Context, proxy *models.Proxy) error {
	query := `
		UPDATE proxies 
		SET username = $2, password = $3, protocol = $4, region = $5, updated_at = $6
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		proxy.ID,
		proxy.Username,
		proxy.Password,
		proxy.Protocol,
		proxy.Region,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("update proxy failed: %w", err)
	}

	return nil
}

// Delete 删除代理
func (r *ProxyRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM proxies WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete proxy failed: %w", err)
	}
	return nil
}

// List 列表查询
func (r *ProxyRepository) List(ctx context.Context, filter *models.ProxyFilter) (*models.ProxyResult, error) {
	// 构建查询条件
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Protocol != nil {
		conditions = append(conditions, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, *filter.Protocol)
		argIdx++
	}

	if filter.Region != nil {
		conditions = append(conditions, fmt.Sprintf("region = $%d", argIdx))
		args = append(args, *filter.Region)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM proxies %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count proxies failed: %w", err)
	}

	// 查询数据
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, ip, port, username, password, protocol, region, status,
		       last_check_at, last_check_result, success_count, fail_count,
		       last_used_at, created_at, updated_at
		FROM proxies %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list proxies failed: %w", err)
	}
	defer rows.Close()

	var items []models.Proxy
	for rows.Next() {
		var p models.Proxy
		err := rows.Scan(
			&p.ID, &p.IP, &p.Port, &p.Username, &p.Password,
			&p.Protocol, &p.Region, &p.Status, &p.LastCheckAt,
			&p.LastCheckResult, &p.SuccessCount, &p.FailCount,
			&p.LastUsedAt, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan proxy failed: %w", err)
		}
		items = append(items, p)
	}

	return &models.ProxyResult{
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Items:    items,
	}, nil
}

// UpdateHealthCheck 更新健康检查结果
func (r *ProxyRepository) UpdateHealthCheck(ctx context.Context, id int64, status models.ProxyStatus, result string) error {
	query := `
		UPDATE proxies 
		SET status = $2, last_check_at = $3, last_check_result = $4, updated_at = $5
		WHERE id = $1`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, status, now, result, now)
	if err != nil {
		return fmt.Errorf("update health check failed: %w", err)
	}

	return nil
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
