package models

type ProxySourceStatusResponse struct {
	Healthy                   bool   `json:"healthy"`
	Mode                      string `json:"mode"`
	Message                   string `json:"message"`
	ProxyURL                  string `json:"proxy_url,omitempty"`
	ProxyLeaseID              string `json:"proxy_lease_id,omitempty"`
	ProxyExpireAt             string `json:"proxy_expire_at,omitempty"`
	CheckedAt                 string `json:"checked_at"`
	AvailableManualProxyCount int64  `json:"available_manual_proxy_count"`
	DynamicConfigured         bool   `json:"dynamic_configured"`
}

type ProxySourcePolicy struct {
	ID                       int64  `json:"id"`
	ScopeType                string `json:"scope_type"`
	ScopeValue               string `json:"scope_value,omitempty"`
	PrimarySource            string `json:"primary_source"`
	FallbackSource           string `json:"fallback_source,omitempty"`
	FallbackEnabled          bool   `json:"fallback_enabled"`
	DynamicTimeoutMS         int32  `json:"dynamic_timeout_ms"`
	DynamicRetryCount        int32  `json:"dynamic_retry_count"`
	DynamicCircuitBreakerSec int32  `json:"dynamic_circuit_breaker_sec"`
	MinLeaseTTLSec           int32  `json:"min_lease_ttl_sec"`
	ManualSelectionStrategy  string `json:"manual_selection_strategy"`
}

type UpdateProxySourcePolicyRequest struct {
	PrimarySource            string `json:"primary_source"`
	FallbackSource           string `json:"fallback_source"`
	FallbackEnabled          bool   `json:"fallback_enabled"`
	DynamicTimeoutMS         int32  `json:"dynamic_timeout_ms"`
	DynamicRetryCount        int32  `json:"dynamic_retry_count"`
	DynamicCircuitBreakerSec int32  `json:"dynamic_circuit_breaker_sec"`
	MinLeaseTTLSec           int32  `json:"min_lease_ttl_sec"`
	ManualSelectionStrategy  string `json:"manual_selection_strategy"`
}

type ProxyInfo struct {
	ID                   int64  `json:"id"`
	Host                 string `json:"host"`
	Port                 int32  `json:"port"`
	Protocol             string `json:"protocol"`
	Username             string `json:"username,omitempty"`
	Region               string `json:"region,omitempty"`
	Priority             int32  `json:"priority"`
	PlatformTags         string `json:"platform_tags,omitempty"`
	Remark               string `json:"remark,omitempty"`
	Status               int32  `json:"status"`
	LastUsedAt           string `json:"last_used_at,omitempty"`
	SuccessCount         int32  `json:"success_count"`
	FailCount            int32  `json:"fail_count"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
	CooldownUntil        string `json:"cooldown_until,omitempty"`
	ConsecutiveFailCount int32  `json:"consecutive_fail_count"`
	RiskScore            int32  `json:"risk_score"`
	LastErrorCategory    string `json:"last_error_category,omitempty"`
	LastFailAt           string `json:"last_fail_at,omitempty"`
	MaxConcurrent        int32  `json:"max_concurrent"`
	ActiveTaskCount      int32  `json:"active_task_count"`
}

type ProxyListResponse struct {
	Items    []ProxyInfo `json:"items"`
	Total    int64       `json:"total"`
	Page     int32       `json:"page"`
	PageSize int32       `json:"page_size"`
}

type ProxyUsageEventFilter struct {
	TaskID        string
	ProxyID       int64
	ProxyLeaseID  string
	SourceType    string
	Stage         string
	Platform      string
	Success       string
	ErrorCategory string
	StartTimeUnix int64
	EndTimeUnix   int64
	Page          int32
	PageSize      int32
	SortOrder     string
}

type ProxyUsageEventInfo struct {
	ID                   int64  `json:"id"`
	TaskID               string `json:"task_id"`
	ProxyID              int64  `json:"proxy_id,omitempty"`
	ProxyLeaseID         string `json:"proxy_lease_id,omitempty"`
	SourceType           string `json:"source_type"`
	Stage                string `json:"stage"`
	Platform             string `json:"platform,omitempty"`
	Success              bool   `json:"success"`
	ErrorCategory        string `json:"error_category,omitempty"`
	ErrorMessage         string `json:"error_message,omitempty"`
	CreatedAt            string `json:"created_at"`
	ProxyHost            string `json:"proxy_host,omitempty"`
	ProxyPort            int32  `json:"proxy_port,omitempty"`
	ProxyProtocol        string `json:"proxy_protocol,omitempty"`
	ProxyRegion          string `json:"proxy_region,omitempty"`
	ProxyRiskScore       int32  `json:"proxy_risk_score"`
	ProxyCooldownUntil   string `json:"proxy_cooldown_until,omitempty"`
	ProxyActiveTaskCount int32  `json:"proxy_active_task_count"`
	ProxyMaxConcurrent   int32  `json:"proxy_max_concurrent"`
}

type ProxyUsageEventCount struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type ProxyUsageEventSummary struct {
	SuccessCount   int64                  `json:"success_count"`
	FailureCount   int64                  `json:"failure_count"`
	FailureRate    float64                `json:"failure_rate"`
	CategoryCounts []ProxyUsageEventCount `json:"category_counts"`
	StageCounts    []ProxyUsageEventCount `json:"stage_counts"`
	PlatformCounts []ProxyUsageEventCount `json:"platform_counts"`
}

type ProxyUsageEventListResponse struct {
	Events   []ProxyUsageEventInfo  `json:"events"`
	Total    int64                  `json:"total"`
	Page     int32                  `json:"page"`
	PageSize int32                  `json:"page_size"`
	Summary  ProxyUsageEventSummary `json:"summary"`
}

type ListProxiesRequest struct {
	Search    string `form:"search"`
	Protocol  string `form:"protocol"`
	Region    string `form:"region"`
	Status    *int32 `form:"status"`
	Page      int32  `form:"page"`
	PageSize  int32  `form:"page_size"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
}

type CreateProxyRequest struct {
	Host         string `json:"host"`
	Port         int32  `json:"port"`
	Protocol     string `json:"protocol"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Region       string `json:"region"`
	Priority     int32  `json:"priority"`
	PlatformTags string `json:"platform_tags"`
	Remark       string `json:"remark"`
	Status       int32  `json:"status"`
}

type UpdateProxyRequest struct {
	Host         string `json:"host"`
	Port         int32  `json:"port"`
	Protocol     string `json:"protocol"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Region       string `json:"region"`
	Priority     int32  `json:"priority"`
	PlatformTags string `json:"platform_tags"`
	Remark       string `json:"remark"`
}

type UpdateProxyStatusRequest struct {
	Status int32 `json:"status"`
}
