package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"youdlp/asset-service/internal/models"
)

// ErrProxyAlreadyExists 表示同一 IP 和端口的代理已存在
var ErrProxyAlreadyExists = errors.New("proxy already exists")

// ProxyRepository 代理数据访问层
type ProxyRepository struct {
	db *sql.DB
}

// NewProxyRepository 创建代理仓库
func NewProxyRepository(db *sql.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

// UpdateUsage 更新使用统计和风控状态。
func (r *ProxyRepository) UpdateUsage(ctx context.Context, id int64, success bool, errorCategory string, riskDelta int, cooldownUntil *time.Time) error {
	var query string
	now := time.Now()
	if success {
		query = `
			UPDATE proxies 
			SET success_count = success_count + 1,
			    consecutive_fail_count = 0,
			    risk_score = GREATEST(risk_score - 5, 0),
			    updated_at = $2
			WHERE id = $1`
		_, err := r.db.ExecContext(ctx, query, id, now)
		if err != nil {
			return fmt.Errorf("update usage failed: %w", err)
		}
		return nil
	}

	if errorCategory == "" {
		errorCategory = models.ErrorCategoryUnknown
	}
	query = `
		UPDATE proxies
		SET fail_count = fail_count + 1,
		    consecutive_fail_count = consecutive_fail_count + 1,
		    risk_score = LEAST(risk_score + $2, $3),
		    cooldown_until = $4,
		    last_error_category = $5,
		    last_fail_at = $6,
		    updated_at = $6
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, riskDelta, models.ProxyRiskMaxScore, cooldownUntil, errorCategory, now)
	if err != nil {
		return fmt.Errorf("update usage failed: %w", err)
	}

	return nil
}

// ReleaseActiveTask 释放手动代理占用计数，幂等地避免降到负数。
func (r *ProxyRepository) ReleaseActiveTask(ctx context.Context, id int64) error {
	query := `
		UPDATE proxies
		SET active_task_count = GREATEST(active_task_count - 1, 0),
		    updated_at = $2
		WHERE id = $1`

	if _, err := r.db.ExecContext(ctx, query, id, time.Now()); err != nil {
		return fmt.Errorf("release proxy active task failed: %w", err)
	}
	return nil
}

// IsUsableForBoundTask 检查已有绑定的手动代理是否仍满足风控条件。
func (r *ProxyRepository) IsUsableForBoundTask(ctx context.Context, id int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM proxies
			WHERE id = $1
			  AND status = $2
			  AND deleted_at IS NULL
			  AND (cooldown_until IS NULL OR cooldown_until <= $3)
			  AND risk_score < $4
			  AND active_task_count <= max_concurrent
		)`

	var usable bool
	if err := r.db.QueryRowContext(ctx, query, id, models.ProxyStatusActive, time.Now(), models.ProxyRiskExcludeThreshold).Scan(&usable); err != nil {
		return false, fmt.Errorf("check proxy usability failed: %w", err)
	}
	return usable, nil
}

// CountSelectableProxies 只读统计当前可被选择的手动代理数量。
func (r *ProxyRepository) CountSelectableProxies(ctx context.Context, protocol *models.ProxyProtocol, region *string) (int64, error) {
	conditions, args := r.selectionConditions(protocol, region, nil)
	query := fmt.Sprintf("SELECT COUNT(*) FROM proxies WHERE %s", strings.Join(conditions, " AND "))

	var count int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count selectable proxies failed: %w", err)
	}
	return count, nil
}

// RecordUsageEvent 记录代理使用事件。
func (r *ProxyRepository) RecordUsageEvent(ctx context.Context, taskID string, proxyID *int64, leaseID, sourceType, stage, platform string, success bool, errorCategory, errorMessage string) error {
	query := `
		INSERT INTO proxy_usage_events (
			task_id, proxy_id, proxy_lease_id, source_type, stage, platform,
			success, error_category, error_message, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	if _, err := r.db.ExecContext(ctx, query, nullableString(taskID), proxyID, nullableString(leaseID), nullableString(sourceType), stage, nullableString(platform), success, nullableString(errorCategory), nullableString(errorMessage), time.Now()); err != nil {
		return fmt.Errorf("record proxy usage event failed: %w", err)
	}
	return nil
}

// UpdatePlatformRisk 更新平台级风险状态。
func (r *ProxyRepository) UpdatePlatformRisk(ctx context.Context, platform, errorCategory string, cooldownUntil *time.Time) error {
	if platform == "" || (errorCategory != models.ErrorCategoryBotDetected && errorCategory != models.ErrorCategoryRateLimited) {
		return nil
	}

	query := `
		INSERT INTO platform_risk_states (
			platform, cooldown_until, rate_limit_level,
			recent_bot_detected_count, recent_rate_limited_count, updated_at
		)
		VALUES (
			$1, NULL, 0,
			CASE WHEN $2 = $3 THEN 1 ELSE 0 END,
			CASE WHEN $2 = $4 THEN 1 ELSE 0 END,
			$5
		)
		ON CONFLICT (platform) DO UPDATE
		SET recent_bot_detected_count = CASE
		        WHEN platform_risk_states.updated_at < $5 - INTERVAL '10 minutes' THEN CASE WHEN $2 = $3 THEN 1 ELSE 0 END
		        ELSE platform_risk_states.recent_bot_detected_count + CASE WHEN $2 = $3 THEN 1 ELSE 0 END
		    END,
		    recent_rate_limited_count = CASE
		        WHEN platform_risk_states.updated_at < $5 - INTERVAL '10 minutes' THEN CASE WHEN $2 = $4 THEN 1 ELSE 0 END
		        ELSE platform_risk_states.recent_rate_limited_count + CASE WHEN $2 = $4 THEN 1 ELSE 0 END
		    END,
		    cooldown_until = CASE
		        WHEN (
		          CASE WHEN platform_risk_states.updated_at < $5 - INTERVAL '10 minutes' THEN CASE WHEN $2 = $3 THEN 1 ELSE 0 END ELSE platform_risk_states.recent_bot_detected_count + CASE WHEN $2 = $3 THEN 1 ELSE 0 END END
		          +
		          CASE WHEN platform_risk_states.updated_at < $5 - INTERVAL '10 minutes' THEN CASE WHEN $2 = $4 THEN 1 ELSE 0 END ELSE platform_risk_states.recent_rate_limited_count + CASE WHEN $2 = $4 THEN 1 ELSE 0 END END
		        ) >= 3 THEN $6
		        ELSE platform_risk_states.cooldown_until
		    END,
		    rate_limit_level = CASE
		        WHEN (
		          CASE WHEN platform_risk_states.updated_at < $5 - INTERVAL '10 minutes' THEN CASE WHEN $2 = $3 THEN 1 ELSE 0 END ELSE platform_risk_states.recent_bot_detected_count + CASE WHEN $2 = $3 THEN 1 ELSE 0 END END
		          +
		          CASE WHEN platform_risk_states.updated_at < $5 - INTERVAL '10 minutes' THEN CASE WHEN $2 = $4 THEN 1 ELSE 0 END ELSE platform_risk_states.recent_rate_limited_count + CASE WHEN $2 = $4 THEN 1 ELSE 0 END END
		        ) >= 3 THEN 1
		        ELSE platform_risk_states.rate_limit_level
		    END,
		    updated_at = $5`

	if _, err := r.db.ExecContext(ctx, query, platform, errorCategory, models.ErrorCategoryBotDetected, models.ErrorCategoryRateLimited, time.Now(), cooldownUntil); err != nil {
		return fmt.Errorf("update platform risk failed: %w", err)
	}
	return nil
}

// ListUsageEvents 查询代理使用事件和筛选条件下的汇总统计。
func (r *ProxyRepository) ListUsageEvents(ctx context.Context, filter models.ProxyUsageEventFilter) (*models.ProxyUsageEventResult, error) {
	whereClause, args := r.proxyUsageEventWhereClause(filter)

	totalQuery := fmt.Sprintf("SELECT COUNT(*) FROM proxy_usage_events e WHERE %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, totalQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count proxy usage events failed: %w", err)
	}

	summary, err := r.proxyUsageEventSummary(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}

	sortOrder := "DESC"
	if filter.SortOrder == models.ProxyUsageSortOrderAsc {
		sortOrder = "ASC"
	}
	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	queryArgs := append(append([]interface{}{}, args...), filter.PageSize, (filter.Page-1)*filter.PageSize)
	query := fmt.Sprintf(`
		SELECT
			e.id,
			COALESCE(e.task_id, ''),
			COALESCE(e.proxy_id, 0),
			COALESCE(e.proxy_lease_id, ''),
			COALESCE(e.source_type, ''),
			e.stage,
			COALESCE(e.platform, ''),
			e.success,
			COALESCE(e.error_category, ''),
			COALESCE(e.error_message, ''),
			e.created_at,
			COALESCE(p.host, p.ip, ''),
			COALESCE(p.port, 0),
			COALESCE(p.protocol, ''),
			COALESCE(p.region, ''),
			COALESCE(p.risk_score, 0),
			p.cooldown_until,
			COALESCE(p.active_task_count, 0),
			COALESCE(p.max_concurrent, 0)
		FROM proxy_usage_events e
		LEFT JOIN proxies p ON p.id = e.proxy_id
		WHERE %s
		ORDER BY e.created_at %s, e.id %s
		LIMIT $%d OFFSET $%d`,
		whereClause,
		sortOrder,
		sortOrder,
		limitArg,
		offsetArg,
	)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list proxy usage events failed: %w", err)
	}
	defer rows.Close()

	events := make([]models.ProxyUsageEvent, 0)
	for rows.Next() {
		var event models.ProxyUsageEvent
		var proxyPort int64
		var proxyRiskScore int64
		var proxyActiveTaskCount int64
		var proxyMaxConcurrent int64
		var proxyCooldownUntil sql.NullTime
		if err := rows.Scan(
			&event.ID,
			&event.TaskID,
			&event.ProxyID,
			&event.ProxyLeaseID,
			&event.SourceType,
			&event.Stage,
			&event.Platform,
			&event.Success,
			&event.ErrorCategory,
			&event.ErrorMessage,
			&event.CreatedAt,
			&event.ProxyHost,
			&proxyPort,
			&event.ProxyProtocol,
			&event.ProxyRegion,
			&proxyRiskScore,
			&proxyCooldownUntil,
			&proxyActiveTaskCount,
			&proxyMaxConcurrent,
		); err != nil {
			return nil, fmt.Errorf("scan proxy usage event failed: %w", err)
		}

		event.ProxyPort = int32(proxyPort)
		event.ProxyRiskScore = int32(proxyRiskScore)
		event.ProxyActiveTaskCount = int32(proxyActiveTaskCount)
		event.ProxyMaxConcurrent = int32(proxyMaxConcurrent)
		if proxyCooldownUntil.Valid {
			event.ProxyCooldownUntil = &proxyCooldownUntil.Time
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy usage events failed: %w", err)
	}

	return &models.ProxyUsageEventResult{
		Events:   events,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Summary:  *summary,
	}, nil
}

func (r *ProxyRepository) GetDashboardStats(ctx context.Context, recentSince time.Time) (models.DashboardProxies, error) {
	var stats models.DashboardProxies
	now := time.Now()
	query := `
		SELECT
			COUNT(*) FILTER (WHERE deleted_at IS NULL),
			COUNT(*) FILTER (WHERE deleted_at IS NULL AND status = $1),
			COUNT(*) FILTER (
				WHERE deleted_at IS NULL
				  AND status = $1
				  AND (cooldown_until IS NULL OR cooldown_until <= $2)
				  AND risk_score < $3
				  AND active_task_count < max_concurrent
			),
			COUNT(*) FILTER (WHERE deleted_at IS NULL AND cooldown_until IS NOT NULL AND cooldown_until > $2),
			COUNT(*) FILTER (WHERE deleted_at IS NULL AND max_concurrent > 0 AND active_task_count >= max_concurrent),
			COUNT(*) FILTER (WHERE deleted_at IS NULL AND risk_score >= $4)
		FROM proxies`
	if err := r.db.QueryRowContext(ctx, query, models.ProxyStatusActive, now, models.ProxyRiskExcludeThreshold, 70).Scan(
		&stats.Total,
		&stats.Active,
		&stats.Available,
		&stats.Cooling,
		&stats.Saturated,
		&stats.HighRisk,
	); err != nil {
		return models.DashboardProxies{}, fmt.Errorf("query dashboard proxy pool stats failed: %w", err)
	}

	usageQuery := `
		SELECT
			COUNT(*) FILTER (WHERE success = TRUE),
			COUNT(*) FILTER (WHERE success = FALSE)
		FROM proxy_usage_events
		WHERE created_at >= $1`
	if err := r.db.QueryRowContext(ctx, usageQuery, recentSince).Scan(&stats.RecentSuccess, &stats.RecentFailure); err != nil {
		return models.DashboardProxies{}, fmt.Errorf("query dashboard proxy usage stats failed: %w", err)
	}
	recentTotal := stats.RecentSuccess + stats.RecentFailure
	if recentTotal > 0 {
		stats.RecentFailureRate = float64(stats.RecentFailure) / float64(recentTotal)
	}

	categoriesQuery := `
		SELECT COALESCE(error_category, 'unknown') AS key, COUNT(*) AS count
		FROM proxy_usage_events
		WHERE created_at >= $1
		  AND success = FALSE
		GROUP BY key
		ORDER BY count DESC, key ASC
		LIMIT 10`
	rows, err := r.db.QueryContext(ctx, categoriesQuery, recentSince)
	if err != nil {
		return models.DashboardProxies{}, fmt.Errorf("query dashboard proxy error categories failed: %w", err)
	}
	defer rows.Close()

	stats.TopErrorCategories = make([]models.DashboardCount, 0)
	for rows.Next() {
		var item models.DashboardCount
		if err := rows.Scan(&item.Key, &item.Count); err != nil {
			return models.DashboardProxies{}, fmt.Errorf("scan dashboard proxy error category failed: %w", err)
		}
		stats.TopErrorCategories = append(stats.TopErrorCategories, item)
	}
	if err := rows.Err(); err != nil {
		return models.DashboardProxies{}, fmt.Errorf("iterate dashboard proxy error categories failed: %w", err)
	}

	return stats, nil
}

func (r *ProxyRepository) proxyUsageEventWhereClause(filter models.ProxyUsageEventFilter) (string, []interface{}) {
	conditions := make([]string, 0)
	args := make([]interface{}, 0)
	argIdx := 1

	if !filter.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("e.created_at >= $%d", argIdx))
		args = append(args, filter.StartTime)
		argIdx++
	}
	if !filter.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("e.created_at <= $%d", argIdx))
		args = append(args, filter.EndTime)
		argIdx++
	}
	if filter.TaskID != "" {
		conditions = append(conditions, fmt.Sprintf("e.task_id = $%d", argIdx))
		args = append(args, filter.TaskID)
		argIdx++
	}
	if filter.ProxyID > 0 {
		conditions = append(conditions, fmt.Sprintf("e.proxy_id = $%d", argIdx))
		args = append(args, filter.ProxyID)
		argIdx++
	}
	if filter.ProxyLeaseID != "" {
		conditions = append(conditions, fmt.Sprintf("e.proxy_lease_id = $%d", argIdx))
		args = append(args, filter.ProxyLeaseID)
		argIdx++
	}
	if filter.SourceType != "" {
		conditions = append(conditions, fmt.Sprintf("e.source_type = $%d", argIdx))
		args = append(args, filter.SourceType)
		argIdx++
	}
	if filter.Stage != "" {
		conditions = append(conditions, fmt.Sprintf("e.stage = $%d", argIdx))
		args = append(args, filter.Stage)
		argIdx++
	}
	if filter.Platform != "" {
		conditions = append(conditions, fmt.Sprintf("e.platform = $%d", argIdx))
		args = append(args, filter.Platform)
		argIdx++
	}
	switch filter.Success {
	case models.ProxyUsageSuccessOnly:
		conditions = append(conditions, "e.success = TRUE")
	case models.ProxyUsageSuccessFailed:
		conditions = append(conditions, "e.success = FALSE")
	}
	if filter.ErrorCategory != "" {
		conditions = append(conditions, fmt.Sprintf("e.error_category = $%d", argIdx))
		args = append(args, filter.ErrorCategory)
	}
	if len(conditions) == 0 {
		return "TRUE", args
	}
	return strings.Join(conditions, " AND "), args
}

func (r *ProxyRepository) proxyUsageEventSummary(ctx context.Context, whereClause string, args []interface{}) (*models.ProxyUsageEventSummary, error) {
	summary := &models.ProxyUsageEventSummary{}
	countQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) FILTER (WHERE e.success = TRUE),
			COUNT(*) FILTER (WHERE e.success = FALSE)
		FROM proxy_usage_events e
		WHERE %s`, whereClause)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&summary.SuccessCount, &summary.FailureCount); err != nil {
		return nil, fmt.Errorf("summarize proxy usage events failed: %w", err)
	}

	total := summary.SuccessCount + summary.FailureCount
	if total > 0 {
		summary.FailureRate = float64(summary.FailureCount) / float64(total)
	}

	var err error
	summary.CategoryCounts, err = r.proxyUsageEventGroupCounts(ctx, failureOnlyWhereClause(whereClause), args, "COALESCE(e.error_category, 'unknown')")
	if err != nil {
		return nil, err
	}
	summary.StageCounts, err = r.proxyUsageEventGroupCounts(ctx, whereClause, args, "e.stage")
	if err != nil {
		return nil, err
	}
	summary.PlatformCounts, err = r.proxyUsageEventGroupCounts(ctx, whereClause, args, "COALESCE(e.platform, 'unknown')")
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func failureOnlyWhereClause(whereClause string) string {
	if whereClause == "TRUE" {
		return "e.success = FALSE"
	}
	return fmt.Sprintf("(%s) AND e.success = FALSE", whereClause)
}

func (r *ProxyRepository) proxyUsageEventGroupCounts(ctx context.Context, whereClause string, args []interface{}, expr string) ([]models.ProxyUsageEventCount, error) {
	query := fmt.Sprintf(`
		SELECT %s AS key, COUNT(*) AS count
		FROM proxy_usage_events e
		WHERE %s
		GROUP BY key
		ORDER BY count DESC, key ASC
		LIMIT 10`, expr, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("group proxy usage events failed: %w", err)
	}
	defer rows.Close()

	items := make([]models.ProxyUsageEventCount, 0)
	for rows.Next() {
		var item models.ProxyUsageEventCount
		if err := rows.Scan(&item.Key, &item.Count); err != nil {
			return nil, fmt.Errorf("scan proxy usage event group failed: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy usage event groups failed: %w", err)
	}

	return items, nil
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func (r *ProxyRepository) selectionConditions(protocol *models.ProxyProtocol, region *string, excludedID *int64) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
	args = append(args, models.ProxyStatusActive)
	argIdx++

	conditions = append(conditions, "deleted_at IS NULL")

	conditions = append(conditions, fmt.Sprintf("(cooldown_until IS NULL OR cooldown_until <= $%d)", argIdx))
	args = append(args, time.Now())
	argIdx++

	conditions = append(conditions, fmt.Sprintf("risk_score < $%d", argIdx))
	args = append(args, models.ProxyRiskExcludeThreshold)
	argIdx++

	conditions = append(conditions, "active_task_count < max_concurrent")

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

	if excludedID != nil {
		conditions = append(conditions, fmt.Sprintf("id <> $%d", argIdx))
		args = append(args, *excludedID)
	}

	return conditions, args
}

func (r *ProxyRepository) scanProxy(row interface {
	Scan(dest ...interface{}) error
}) (*models.Proxy, error) {
	proxy := &models.Proxy{}
	err := row.Scan(
		&proxy.ID, &proxy.Host, &proxy.IP, &proxy.Port, &proxy.Username, &proxy.Password,
		&proxy.Protocol, &proxy.Region, &proxy.Priority, &proxy.PlatformTags, &proxy.Remark, &proxy.Status, &proxy.LastCheckAt,
		&proxy.LastCheckResult, &proxy.SuccessCount, &proxy.FailCount,
		&proxy.LastUsedAt, &proxy.CooldownUntil, &proxy.ConsecutiveFailCount, &proxy.RiskScore,
		&proxy.LastErrorCategory, &proxy.LastFailAt, &proxy.MaxConcurrent, &proxy.ActiveTaskCount,
		&proxy.DeletedAt, &proxy.CreatedAt, &proxy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return proxy, nil
}

func proxyReturningColumns() string {
	return `p.id, p.host, p.ip, p.port, p.username, p.password,
		          p.protocol, p.region, p.priority, p.platform_tags, p.remark, p.status,
		          p.last_check_at, p.last_check_result, p.success_count, p.fail_count,
		          p.last_used_at, p.cooldown_until, p.consecutive_fail_count, p.risk_score,
		          p.last_error_category, p.last_fail_at, p.max_concurrent, p.active_task_count,
		          p.deleted_at, p.created_at, p.updated_at`
}

// GetAvailableProxy 获取一个可用的代理（轮询策略：最久未使用的活跃代理）
func (r *ProxyRepository) GetAvailableProxy(ctx context.Context, protocol *models.ProxyProtocol, region *string) (*models.Proxy, error) {
	return r.AcquireAvailableProxy(ctx, protocol, region)
}

// GetAvailableProxyExcluding 获取一个可用代理，并可排除指定代理 ID。
func (r *ProxyRepository) GetAvailableProxyExcluding(ctx context.Context, protocol *models.ProxyProtocol, region *string, excludedID *int64) (*models.Proxy, error) {
	return r.AcquireAvailableProxyExcluding(ctx, protocol, region, excludedID)
}

// AcquireAvailableProxy 原子分配一个可用代理，并立即标记最近分配时间。
func (r *ProxyRepository) AcquireAvailableProxy(ctx context.Context, protocol *models.ProxyProtocol, region *string) (*models.Proxy, error) {
	return r.AcquireAvailableProxyExcluding(ctx, protocol, region, nil)
}

// AcquireAvailableProxyExcluding 原子分配一个可用代理，并可排除指定代理 ID。
func (r *ProxyRepository) AcquireAvailableProxyExcluding(ctx context.Context, protocol *models.ProxyProtocol, region *string, excludedID *int64) (*models.Proxy, error) {
	return r.acquireAvailableProxy(ctx, protocol, region, excludedID, false)
}

// AcquireTaskProxyExcluding 原子分配一个任务绑定代理，并递增占用计数。
func (r *ProxyRepository) AcquireTaskProxyExcluding(ctx context.Context, protocol *models.ProxyProtocol, region *string, excludedID *int64) (*models.Proxy, error) {
	return r.acquireAvailableProxy(ctx, protocol, region, excludedID, true)
}

func (r *ProxyRepository) acquireAvailableProxy(ctx context.Context, protocol *models.ProxyProtocol, region *string, excludedID *int64, incrementActive bool) (*models.Proxy, error) {
	conditions, args := r.selectionConditions(protocol, region, excludedID)
	allocatedAtArg := len(args) + 1
	args = append(args, time.Now())

	activeUpdate := ""
	if incrementActive {
		activeUpdate = ",\n		    active_task_count = p.active_task_count + 1"
	}

	query := fmt.Sprintf(`
		WITH candidate AS (
			SELECT id
			FROM proxies
			WHERE %s
			ORDER BY risk_score ASC,
			         last_used_at ASC NULLS FIRST,
			         priority DESC,
			         success_count DESC,
			         consecutive_fail_count ASC,
			         id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE proxies p
		SET last_used_at = $%d,
		    updated_at = $%d%s
		FROM candidate
		WHERE p.id = candidate.id
		RETURNING %s`,
		strings.Join(conditions, " AND "),
		allocatedAtArg,
		allocatedAtArg,
		activeUpdate,
		proxyReturningColumns(),
	)

	proxy, err := r.scanProxy(r.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("get available proxy failed: %w", err)
	}

	return proxy, nil
}

// ListProxies 列出手动代理池
func (r *ProxyRepository) ListProxies(ctx context.Context, filter models.ProxyListFilter) (*models.ProxyListResult, error) {
	page := filter.Page
	if page < 1 {
		page = models.ProxyListDefaultPage
	}
	if page > models.ProxyListMaxPage {
		page = models.ProxyListMaxPage
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = models.ProxyListDefaultPageSize
	}
	if pageSize > models.ProxyListMaxPageSize {
		pageSize = models.ProxyListMaxPageSize
	}

	whereClause, args := r.proxyListWhereClause(filter)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM proxies p WHERE %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count proxies failed: %w", err)
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	queryArgs := append(append([]interface{}{}, args...), pageSize, (page-1)*pageSize)
	query := fmt.Sprintf(`
		SELECT p.id, p.host, p.ip, p.port, p.username, p.password, p.protocol, p.region, p.priority,
		       p.platform_tags, p.remark, p.status, p.last_check_at, p.last_check_result,
		       p.success_count, p.fail_count, p.last_used_at, p.cooldown_until,
		       p.consecutive_fail_count, p.risk_score, p.last_error_category, p.last_fail_at,
		       p.max_concurrent, p.active_task_count, p.deleted_at, p.created_at, p.updated_at
		FROM proxies p
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		whereClause,
		proxyListOrderBy(filter.SortBy, filter.SortOrder),
		limitArg,
		offsetArg,
	)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list proxies failed: %w", err)
	}
	defer rows.Close()

	items := make([]*models.Proxy, 0)
	for rows.Next() {
		item, err := r.scanProxy(rows)
		if err != nil {
			return nil, fmt.Errorf("scan proxy failed: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxies failed: %w", err)
	}

	return &models.ProxyListResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}, nil
}

func (r *ProxyRepository) proxyListWhereClause(filter models.ProxyListFilter) (string, []interface{}) {
	conditions := []string{"deleted_at IS NULL"}
	args := make([]interface{}, 0)
	argIdx := 1

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(COALESCE(host, ip) ILIKE $%d OR COALESCE(region, '') ILIKE $%d OR COALESCE(platform_tags, '') ILIKE $%d OR COALESCE(remark, '') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}
	if filter.Protocol != nil && *filter.Protocol != "" {
		conditions = append(conditions, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, *filter.Protocol)
		argIdx++
	}
	if filter.Region != nil && *filter.Region != "" {
		conditions = append(conditions, fmt.Sprintf("region = $%d", argIdx))
		args = append(args, *filter.Region)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
	}

	return strings.Join(conditions, " AND "), args
}

func proxyListOrderBy(sortBy, sortOrder string) string {
	sortColumns := map[string]string{
		models.ProxyListSortRiskScore:   "p.risk_score",
		models.ProxyListSortPriority:    "p.priority",
		models.ProxyListSortFailCount:   "p.fail_count",
		models.ProxyListSortActiveTasks: "p.active_task_count",
		models.ProxyListSortUpdatedAt:   "p.updated_at",
		models.ProxyListSortLastUsedAt:  "p.last_used_at",
	}

	column, ok := sortColumns[sortBy]
	if !ok || column == "" {
		return "p.status ASC, p.risk_score ASC, p.priority DESC, p.created_at DESC"
	}

	direction := "DESC"
	if strings.EqualFold(sortOrder, models.ProxyListSortOrderAsc) {
		direction = "ASC"
	}

	return fmt.Sprintf("%s %s NULLS LAST, p.id ASC", column, direction)
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
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "proxies_ip_port_key" {
			return 0, ErrProxyAlreadyExists
		}
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
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "proxies_ip_port_key" {
			return ErrProxyAlreadyExists
		}
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
