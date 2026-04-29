package models

type OverviewResponse struct {
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

type TrendPoint struct {
	Label        string  `json:"label"`
	Count        int64   `json:"count"`
	TotalCount   int64   `json:"total_count"`
	SuccessCount int64   `json:"success_count"`
	FailedCount  int64   `json:"failed_count"`
	SuccessRate  float64 `json:"success_rate"`
}

type RequestTrendResponse struct {
	Granularity string       `json:"granularity"`
	Points      []TrendPoint `json:"points"`
}

type UserStatsResponse struct {
	TotalUsers        int64 `json:"total_users"`
	DailyActiveUsers  int64 `json:"daily_active_users"`
	WeeklyActiveUsers int64 `json:"weekly_active_users"`
}

type DashboardDownloads struct {
	Total        int64   `json:"total"`
	TodayTotal   int64   `json:"today_total"`
	SuccessTotal int64   `json:"success_total"`
	FailedTotal  int64   `json:"failed_total"`
	SuccessRate  float64 `json:"success_rate"`
	FailureRate  float64 `json:"failure_rate"`
}

type DashboardUsers struct {
	Total        int64   `json:"total"`
	DailyActive  int64   `json:"daily_active"`
	WeeklyActive int64   `json:"weekly_active"`
	DAUWAURate   float64 `json:"dau_wau_rate"`
	WAUTotalRate float64 `json:"wau_total_rate"`
}

type DashboardCount struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type DashboardProxies struct {
	Total              int64            `json:"total"`
	Active             int64            `json:"active"`
	Available          int64            `json:"available"`
	Cooling            int64            `json:"cooling"`
	Saturated          int64            `json:"saturated"`
	HighRisk           int64            `json:"high_risk"`
	RecentSuccess      int64            `json:"recent_success"`
	RecentFailure      int64            `json:"recent_failure"`
	RecentFailureRate  float64          `json:"recent_failure_rate"`
	TopErrorCategories []DashboardCount `json:"top_error_categories"`
}

type DashboardProxySource struct {
	Healthy           bool   `json:"healthy"`
	Mode              string `json:"mode"`
	Message           string `json:"message"`
	DynamicConfigured bool   `json:"dynamic_configured"`
	ProxyLeaseID      string `json:"proxy_lease_id,omitempty"`
	ProxyExpireAt     string `json:"proxy_expire_at,omitempty"`
}

type DashboardProxyPolicy struct {
	PrimarySource   string `json:"primary_source"`
	FallbackSource  string `json:"fallback_source,omitempty"`
	FallbackEnabled bool   `json:"fallback_enabled"`
}

type DashboardCookies struct {
	Total   int64 `json:"total"`
	Active  int64 `json:"active"`
	Expired int64 `json:"expired"`
	Frozen  int64 `json:"frozen"`
}

type DashboardBilling struct {
	ShortfallCount int64 `json:"shortfall_count"`
}

type DashboardException struct {
	Area        string `json:"area"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	ActionLabel string `json:"action_label"`
	ActionHref  string `json:"action_href,omitempty"`
}

type DashboardHealthResponse struct {
	GeneratedAt string               `json:"generated_at"`
	Downloads   DashboardDownloads   `json:"downloads"`
	Users       DashboardUsers       `json:"users"`
	Proxies     DashboardProxies     `json:"proxies"`
	ProxySource DashboardProxySource `json:"proxy_source"`
	ProxyPolicy DashboardProxyPolicy `json:"proxy_policy"`
	Cookies     DashboardCookies     `json:"cookies"`
	Billing     DashboardBilling     `json:"billing"`
	Exceptions  []DashboardException `json:"exceptions"`
}
