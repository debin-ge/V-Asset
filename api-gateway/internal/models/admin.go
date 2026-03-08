package models

type AdminLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AdminUser struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      int32  `json:"role"`
	CreatedAt string `json:"created_at,omitempty"`
}

type AdminMeResponse struct {
	User AdminUser `json:"user"`
}

type AdminLoginResponse struct {
	User AdminUser `json:"user"`
}

type AdminOverviewResponse struct {
	TotalUsers          int64 `json:"total_users"`
	DailyActiveUsers    int64 `json:"daily_active_users"`
	WeeklyActiveUsers   int64 `json:"weekly_active_users"`
	TotalDownloads      int64 `json:"total_downloads"`
	DownloadsToday      int64 `json:"downloads_today"`
	SuccessDownloads    int64 `json:"success_downloads"`
	FailedDownloads     int64 `json:"failed_downloads"`
	ActiveManualProxies int64 `json:"active_manual_proxies"`
	TotalManualProxies  int64 `json:"total_manual_proxies"`
}

type AdminTrendPoint struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type AdminRequestTrendResponse struct {
	Granularity string            `json:"granularity"`
	Points      []AdminTrendPoint `json:"points"`
}

type AdminUserStatsResponse struct {
	TotalUsers        int64 `json:"total_users"`
	DailyActiveUsers  int64 `json:"daily_active_users"`
	WeeklyActiveUsers int64 `json:"weekly_active_users"`
}

type AdminProxySourcePolicy struct {
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

type AdminUpdateProxySourcePolicyRequest struct {
	PrimarySource            string `json:"primary_source"`
	FallbackSource           string `json:"fallback_source"`
	FallbackEnabled          bool   `json:"fallback_enabled"`
	DynamicTimeoutMS         int32  `json:"dynamic_timeout_ms"`
	DynamicRetryCount        int32  `json:"dynamic_retry_count"`
	DynamicCircuitBreakerSec int32  `json:"dynamic_circuit_breaker_sec"`
	MinLeaseTTLSec           int32  `json:"min_lease_ttl_sec"`
	ManualSelectionStrategy  string `json:"manual_selection_strategy"`
}

type AdminProxyInfo struct {
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

type AdminProxyListResponse struct {
	Items []AdminProxyInfo `json:"items"`
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

type CookieListResponse struct {
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Items    []CookieInfo `json:"items"`
}
