package models

type ProxySourceStatusResponse struct {
	Healthy       bool   `json:"healthy"`
	Mode          string `json:"mode"`
	Message       string `json:"message"`
	ProxyURL      string `json:"proxy_url,omitempty"`
	ProxyLeaseID  string `json:"proxy_lease_id,omitempty"`
	ProxyExpireAt string `json:"proxy_expire_at,omitempty"`
	CheckedAt     string `json:"checked_at"`
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
	ID           int64  `json:"id"`
	Host         string `json:"host"`
	Port         int32  `json:"port"`
	Protocol     string `json:"protocol"`
	Username     string `json:"username,omitempty"`
	Region       string `json:"region,omitempty"`
	Priority     int32  `json:"priority"`
	PlatformTags string `json:"platform_tags,omitempty"`
	Remark       string `json:"remark,omitempty"`
	Status       int32  `json:"status"`
	LastUsedAt   string `json:"last_used_at,omitempty"`
	SuccessCount int32  `json:"success_count"`
	FailCount    int32  `json:"fail_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ProxyListResponse struct {
	Items []ProxyInfo `json:"items"`
}

type ListProxiesRequest struct {
	Search   string `form:"search"`
	Protocol string `form:"protocol"`
	Region   string `form:"region"`
	Status   *int32 `form:"status"`
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
