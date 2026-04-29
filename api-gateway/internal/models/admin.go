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
	Label        string  `json:"label"`
	Count        int64   `json:"count"`
	TotalCount   int64   `json:"total_count"`
	SuccessCount int64   `json:"success_count"`
	FailedCount  int64   `json:"failed_count"`
	SuccessRate  float64 `json:"success_rate"`
}

type AdminRequestTrendResponse struct {
	Granularity string            `json:"granularity"`
	Points      []AdminTrendPoint `json:"points"`
}

type AdminDashboardDownloads struct {
	Total        int64   `json:"total"`
	TodayTotal   int64   `json:"today_total"`
	SuccessTotal int64   `json:"success_total"`
	FailedTotal  int64   `json:"failed_total"`
	SuccessRate  float64 `json:"success_rate"`
	FailureRate  float64 `json:"failure_rate"`
}

type AdminDashboardUsers struct {
	Total        int64   `json:"total"`
	DailyActive  int64   `json:"daily_active"`
	WeeklyActive int64   `json:"weekly_active"`
	DAUWAURate   float64 `json:"dau_wau_rate"`
	WAUTotalRate float64 `json:"wau_total_rate"`
}

type AdminDashboardCount struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type AdminDashboardProxies struct {
	Total              int64                 `json:"total"`
	Active             int64                 `json:"active"`
	Available          int64                 `json:"available"`
	Cooling            int64                 `json:"cooling"`
	Saturated          int64                 `json:"saturated"`
	HighRisk           int64                 `json:"high_risk"`
	RecentSuccess      int64                 `json:"recent_success"`
	RecentFailure      int64                 `json:"recent_failure"`
	RecentFailureRate  float64               `json:"recent_failure_rate"`
	TopErrorCategories []AdminDashboardCount `json:"top_error_categories"`
}

type AdminDashboardProxySource struct {
	Healthy           bool   `json:"healthy"`
	Mode              string `json:"mode"`
	Message           string `json:"message"`
	DynamicConfigured bool   `json:"dynamic_configured"`
	ProxyLeaseID      string `json:"proxy_lease_id,omitempty"`
	ProxyExpireAt     string `json:"proxy_expire_at,omitempty"`
}

type AdminDashboardProxyPolicy struct {
	PrimarySource   string `json:"primary_source"`
	FallbackSource  string `json:"fallback_source,omitempty"`
	FallbackEnabled bool   `json:"fallback_enabled"`
}

type AdminDashboardCookies struct {
	Total   int64 `json:"total"`
	Active  int64 `json:"active"`
	Expired int64 `json:"expired"`
	Frozen  int64 `json:"frozen"`
}

type AdminDashboardBilling struct {
	ShortfallCount int64 `json:"shortfall_count"`
}

type AdminDashboardException struct {
	Area        string `json:"area"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	ActionLabel string `json:"action_label"`
	ActionHref  string `json:"action_href,omitempty"`
}

type AdminDashboardHealthResponse struct {
	GeneratedAt string                    `json:"generated_at"`
	Downloads   AdminDashboardDownloads   `json:"downloads"`
	Users       AdminDashboardUsers       `json:"users"`
	Proxies     AdminDashboardProxies     `json:"proxies"`
	ProxySource AdminDashboardProxySource `json:"proxy_source"`
	ProxyPolicy AdminDashboardProxyPolicy `json:"proxy_policy"`
	Cookies     AdminDashboardCookies     `json:"cookies"`
	Billing     AdminDashboardBilling     `json:"billing"`
	Exceptions  []AdminDashboardException `json:"exceptions"`
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

type AdminProxyPagination struct {
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
	Total    int64 `json:"total"`
}

type AdminProxyListResponse struct {
	Items      []AdminProxyInfo     `json:"items"`
	Pagination AdminProxyPagination `json:"pagination"`
}

type AdminProxyUsageEvent struct {
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

type AdminProxyUsageEventCount struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type AdminProxyUsageEventSummary struct {
	SuccessCount   int64                       `json:"success_count"`
	FailureCount   int64                       `json:"failure_count"`
	FailureRate    float64                     `json:"failure_rate"`
	CategoryCounts []AdminProxyUsageEventCount `json:"category_counts"`
	StageCounts    []AdminProxyUsageEventCount `json:"stage_counts"`
	PlatformCounts []AdminProxyUsageEventCount `json:"platform_counts"`
}

type AdminProxyUsageEventPagination struct {
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
	Total    int64 `json:"total"`
}

type AdminProxyUsageEventListResponse struct {
	Events     []AdminProxyUsageEvent         `json:"events"`
	Pagination AdminProxyUsageEventPagination `json:"pagination"`
	Summary    AdminProxyUsageEventSummary    `json:"summary"`
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
