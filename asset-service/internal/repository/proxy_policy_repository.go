package repository

import (
	"context"
	"database/sql"
	"fmt"

	"youdlp/asset-service/internal/models"
)

// ProxyPolicyRepository 代理来源策略仓储
type ProxyPolicyRepository struct {
	db *sql.DB
}

// NewProxyPolicyRepository 创建代理来源策略仓储
func NewProxyPolicyRepository(db *sql.DB) *ProxyPolicyRepository {
	return &ProxyPolicyRepository{db: db}
}

// GetEffectivePolicy 获取生效策略，优先 platform，其次 global
func (r *ProxyPolicyRepository) GetEffectivePolicy(ctx context.Context, platform *string) (*models.ProxySourcePolicy, error) {
	if platform != nil && *platform != "" {
		policy, err := r.getByScope(ctx, "platform", *platform)
		if err != nil {
			return nil, err
		}
		if policy != nil {
			return policy, nil
		}
	}

	return r.getByScope(ctx, "global", "")
}

func (r *ProxyPolicyRepository) getByScope(ctx context.Context, scopeType, scopeValue string) (*models.ProxySourcePolicy, error) {
	query := `
		SELECT id, scope_type, scope_value, primary_source, fallback_source, fallback_enabled,
		       dynamic_timeout_ms, dynamic_retry_count, dynamic_circuit_breaker_sec,
		       min_lease_ttl_sec, manual_selection_strategy, status, created_at, updated_at
		FROM proxy_source_policies
		WHERE scope_type = $1
		  AND COALESCE(scope_value, '') = $2
		  AND status = 0
		LIMIT 1`

	policy := &models.ProxySourcePolicy{}
	err := r.db.QueryRowContext(ctx, query, scopeType, scopeValue).Scan(
		&policy.ID,
		&policy.ScopeType,
		&policy.ScopeValue,
		&policy.PrimarySource,
		&policy.FallbackSource,
		&policy.FallbackEnabled,
		&policy.DynamicTimeoutMS,
		&policy.DynamicRetryCount,
		&policy.DynamicCircuitBreakerSec,
		&policy.MinLeaseTTLSec,
		&policy.ManualSelectionStrategy,
		&policy.Status,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get effective proxy policy failed: %w", err)
	}

	return policy, nil
}

// UpdatePolicy 更新来源策略
func (r *ProxyPolicyRepository) UpdatePolicy(
	ctx context.Context,
	id int64,
	primarySource string,
	fallbackSource *string,
	fallbackEnabled bool,
	dynamicTimeoutMS int,
	dynamicRetryCount int,
	dynamicCircuitBreakerSec int,
	minLeaseTTLSec int,
	manualSelectionStrategy string,
) error {
	query := `
		UPDATE proxy_source_policies
		SET primary_source = $2,
		    fallback_source = $3,
		    fallback_enabled = $4,
		    dynamic_timeout_ms = $5,
		    dynamic_retry_count = $6,
		    dynamic_circuit_breaker_sec = $7,
		    min_lease_ttl_sec = $8,
		    manual_selection_strategy = $9,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	result, err := r.db.ExecContext(
		ctx,
		query,
		id,
		primarySource,
		fallbackSource,
		fallbackEnabled,
		dynamicTimeoutMS,
		dynamicRetryCount,
		dynamicCircuitBreakerSec,
		minLeaseTTLSec,
		manualSelectionStrategy,
	)
	if err != nil {
		return fmt.Errorf("update proxy source policy failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get update proxy source policy rows affected failed: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
