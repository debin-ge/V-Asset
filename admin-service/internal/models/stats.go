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
	Label string `json:"label"`
	Count int64  `json:"count"`
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
