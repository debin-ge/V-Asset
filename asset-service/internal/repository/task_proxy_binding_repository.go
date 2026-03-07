package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"vasset/asset-service/internal/models"
)

// ErrTaskProxyBindingAlreadyExists 表示任务绑定已存在
var ErrTaskProxyBindingAlreadyExists = errors.New("task proxy binding already exists")

// TaskProxyBindingRepository 任务代理绑定仓储
type TaskProxyBindingRepository struct {
	db *sql.DB
}

// NewTaskProxyBindingRepository 创建任务代理绑定仓储
func NewTaskProxyBindingRepository(db *sql.DB) *TaskProxyBindingRepository {
	return &TaskProxyBindingRepository{db: db}
}

// GetByTaskID 按任务 ID 获取绑定
func (r *TaskProxyBindingRepository) GetByTaskID(ctx context.Context, taskID string) (*models.TaskProxyBinding, error) {
	query := `
		SELECT id, task_id, source_type, source_policy_id, proxy_id, proxy_lease_id,
		       proxy_url_snapshot, protocol, region, platform, expire_at, bind_status,
		       is_degraded, degrade_reason, last_report_stage, last_report_success,
		       last_report_at, created_at, updated_at
		FROM task_proxy_bindings
		WHERE task_id = $1
		LIMIT 1`

	binding := &models.TaskProxyBinding{}
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&binding.ID,
		&binding.TaskID,
		&binding.SourceType,
		&binding.SourcePolicyID,
		&binding.ProxyID,
		&binding.ProxyLeaseID,
		&binding.ProxyURLSnapshot,
		&binding.Protocol,
		&binding.Region,
		&binding.Platform,
		&binding.ExpireAt,
		&binding.BindStatus,
		&binding.IsDegraded,
		&binding.DegradeReason,
		&binding.LastReportStage,
		&binding.LastReportSuccess,
		&binding.LastReportAt,
		&binding.CreatedAt,
		&binding.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get task proxy binding failed: %w", err)
	}

	return binding, nil
}

// CreateIfAbsent 幂等创建任务绑定
func (r *TaskProxyBindingRepository) CreateIfAbsent(ctx context.Context, binding *models.TaskProxyBinding) error {
	query := `
		INSERT INTO task_proxy_bindings (
			task_id, source_type, source_policy_id, proxy_id, proxy_lease_id,
			proxy_url_snapshot, protocol, region, platform, expire_at, bind_status,
			is_degraded, degrade_reason, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15
		)
		ON CONFLICT (task_id) DO NOTHING`

	now := time.Now()
	result, err := r.db.ExecContext(
		ctx,
		query,
		binding.TaskID,
		binding.SourceType,
		binding.SourcePolicyID,
		binding.ProxyID,
		binding.ProxyLeaseID,
		binding.ProxyURLSnapshot,
		binding.Protocol,
		binding.Region,
		binding.Platform,
		binding.ExpireAt,
		binding.BindStatus,
		binding.IsDegraded,
		binding.DegradeReason,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("create task proxy binding failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get task proxy binding rows affected failed: %w", err)
	}
	if rowsAffected == 0 {
		return ErrTaskProxyBindingAlreadyExists
	}
	return nil
}

// UpdateReport 更新任务绑定的最近上报信息
func (r *TaskProxyBindingRepository) UpdateReport(ctx context.Context, taskID, stage string, success bool) error {
	query := `
		UPDATE task_proxy_bindings
		SET last_report_stage = $2,
		    last_report_success = $3,
		    last_report_at = $4,
		    updated_at = $4
		WHERE task_id = $1`

	now := time.Now()
	if _, err := r.db.ExecContext(ctx, query, taskID, stage, success, now); err != nil {
		return fmt.Errorf("update task proxy binding report failed: %w", err)
	}
	return nil
}
