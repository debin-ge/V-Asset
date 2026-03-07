package models

import "time"

// ProxySourceType 代理来源类型
type ProxySourceType string

const (
	ProxySourceTypeDynamicAPI ProxySourceType = "dynamic_api"
	ProxySourceTypeManualPool ProxySourceType = "manual_pool"
)

// TaskProxyBindStatus 任务代理绑定状态
type TaskProxyBindStatus string

const (
	TaskProxyBindStatusBound    TaskProxyBindStatus = "bound"
	TaskProxyBindStatusReleased TaskProxyBindStatus = "released"
	TaskProxyBindStatusExpired  TaskProxyBindStatus = "expired"
	TaskProxyBindStatusFailed   TaskProxyBindStatus = "failed"
)

// ProxySourcePolicy 代理来源策略
type ProxySourcePolicy struct {
	ID                       int64           `db:"id"`
	ScopeType                string          `db:"scope_type"`
	ScopeValue               *string         `db:"scope_value"`
	PrimarySource            ProxySourceType `db:"primary_source"`
	FallbackSource           *string         `db:"fallback_source"`
	FallbackEnabled          bool            `db:"fallback_enabled"`
	DynamicTimeoutMS         int             `db:"dynamic_timeout_ms"`
	DynamicRetryCount        int             `db:"dynamic_retry_count"`
	DynamicCircuitBreakerSec int             `db:"dynamic_circuit_breaker_sec"`
	MinLeaseTTLSec           int             `db:"min_lease_ttl_sec"`
	ManualSelectionStrategy  string          `db:"manual_selection_strategy"`
	Status                   int             `db:"status"`
	CreatedAt                time.Time       `db:"created_at"`
	UpdatedAt                time.Time       `db:"updated_at"`
}

// TaskProxyBinding 任务代理绑定
type TaskProxyBinding struct {
	ID                int64               `db:"id"`
	TaskID            string              `db:"task_id"`
	SourceType        ProxySourceType     `db:"source_type"`
	SourcePolicyID    *int64              `db:"source_policy_id"`
	ProxyID           *int64              `db:"proxy_id"`
	ProxyLeaseID      *string             `db:"proxy_lease_id"`
	ProxyURLSnapshot  string              `db:"proxy_url_snapshot"`
	Protocol          string              `db:"protocol"`
	Region            *string             `db:"region"`
	Platform          *string             `db:"platform"`
	ExpireAt          *time.Time          `db:"expire_at"`
	BindStatus        TaskProxyBindStatus `db:"bind_status"`
	IsDegraded        bool                `db:"is_degraded"`
	DegradeReason     *string             `db:"degrade_reason"`
	LastReportStage   *string             `db:"last_report_stage"`
	LastReportSuccess *bool               `db:"last_report_success"`
	LastReportAt      *time.Time          `db:"last_report_at"`
	CreatedAt         time.Time           `db:"created_at"`
	UpdatedAt         time.Time           `db:"updated_at"`
}

// ProxyAcquireResult 来源选择结果
type ProxyAcquireResult struct {
	SourceType   ProxySourceType
	ProxyID      *int64
	ProxyLeaseID *string
	ProxyURL     string
	ExpireAt     *time.Time
}
