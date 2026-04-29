package models

import (
	"database/sql"
	"time"
)

// HistoryStatus 下载状态枚举
type HistoryStatus int

const (
	StatusPending        HistoryStatus = 0 // 等待中
	StatusProcessing     HistoryStatus = 1 // 处理中
	StatusCompleted      HistoryStatus = 2 // 已完成
	StatusFailed         HistoryStatus = 3 // 失败
	StatusPendingCleanup HistoryStatus = 4 // 待清理(quick_download完成后)
)

// DownloadHistory 下载历史记录
type DownloadHistory struct {
	ID           int64          `db:"id"`
	TaskID       string         `db:"task_id"`
	UserID       string         `db:"user_id"`
	URL          string         `db:"url"`
	Platform     string         `db:"platform"`
	Title        string         `db:"title"`
	Mode         string         `db:"mode"`      // video, audio, both
	Quality      string         `db:"quality"`   // best, 1080p, 720p, etc.
	FileSize     sql.NullInt64  `db:"file_size"` // 文件大小(字节)
	FilePath     sql.NullString `db:"file_path"` // 文件存储路径
	FileName     sql.NullString `db:"file_name"` // 文件名
	FileHash     sql.NullString `db:"file_hash"` // 文件哈希
	Status       HistoryStatus  `db:"status"`
	ErrorMessage sql.NullString `db:"error_message"` // 错误信息
	CreatedAt    time.Time      `db:"created_at"`
	StartedAt    *time.Time     `db:"started_at"`
	CompletedAt  *time.Time     `db:"completed_at"`
	Thumbnail    string         `db:"thumbnail"` // 缩略图URL
	Duration     int64          `db:"duration"`  // 视频时长(秒)
	Author       string         `db:"author"`    // 作者/上传者
}

// UserQuota 用户配额
type UserQuota struct {
	ID         int64     `db:"id"`
	UserID     string    `db:"user_id"`
	DailyLimit int       `db:"daily_limit"` // 每日下载限制
	DailyUsed  int       `db:"daily_used"`  // 今日已用
	ResetAt    time.Time `db:"reset_at"`    // 配额重置时间
	UpdatedAt  time.Time `db:"updated_at"`
}

// UserStats 用户统计
type UserStats struct {
	TotalDownloads   int64           `json:"total_downloads"`
	SuccessDownloads int64           `json:"success_downloads"`
	FailedDownloads  int64           `json:"failed_downloads"`
	TotalSize        int64           `json:"total_size_bytes"`
	TopPlatforms     []PlatformStat  `json:"top_platforms"`
	RecentActivity   []DailyActivity `json:"recent_activity"`
}

// PlatformStat 平台统计
type PlatformStat struct {
	Platform string `json:"platform" db:"platform"`
	Count    int64  `json:"count" db:"count"`
}

// DailyActivity 日活动统计
type DailyActivity struct {
	Date  string `json:"date" db:"date"`
	Count int64  `json:"count" db:"count"`
}

// TrendPoint 趋势点
type TrendPoint struct {
	Label        string  `json:"label" db:"label"`
	Count        int64   `json:"count" db:"count"`
	TotalCount   int64   `json:"total_count" db:"total_count"`
	SuccessCount int64   `json:"success_count" db:"success_count"`
	FailedCount  int64   `json:"failed_count" db:"failed_count"`
	SuccessRate  float64 `json:"success_rate" db:"success_rate"`
}

// PlatformStats 平台整体统计
type PlatformStats struct {
	TotalDownloads    int64 `json:"total_downloads"`
	SuccessDownloads  int64 `json:"success_downloads"`
	FailedDownloads   int64 `json:"failed_downloads"`
	DownloadsToday    int64 `json:"downloads_today"`
	DailyActiveUsers  int64 `json:"daily_active_users"`
	WeeklyActiveUsers int64 `json:"weekly_active_users"`
}

type DashboardDownloads struct {
	Total        int64
	TodayTotal   int64
	SuccessTotal int64
	FailedTotal  int64
	SuccessRate  float64
	FailureRate  float64
}

type DashboardCount struct {
	Key   string
	Count int64
}

type DashboardProxies struct {
	Total              int64
	Active             int64
	Available          int64
	Cooling            int64
	Saturated          int64
	HighRisk           int64
	RecentSuccess      int64
	RecentFailure      int64
	RecentFailureRate  float64
	TopErrorCategories []DashboardCount
}

type DashboardCookies struct {
	Total   int64
	Active  int64
	Expired int64
	Frozen  int64
}

type DashboardBilling struct {
	ShortfallCount int64
}

type DashboardUsers struct {
	DailyActive  int64
	WeeklyActive int64
}

type DashboardHealth struct {
	GeneratedAt time.Time
	Downloads   DashboardDownloads
	Users       DashboardUsers
	Proxies     DashboardProxies
	Cookies     DashboardCookies
	Billing     DashboardBilling
}

// HistoryFilter 历史查询过滤条件
type HistoryFilter struct {
	UserID    string
	Status    *HistoryStatus // 可选:状态过滤
	Platform  *string        // 可选:平台过滤
	StartDate *time.Time     // 可选:开始日期
	EndDate   *time.Time     // 可选:结束日期
	Page      int
	PageSize  int
	SortBy    string // created_at, file_size, etc.
	SortOrder string // asc, desc
}

// HistoryResult 历史查询结果
type HistoryResult struct {
	Total    int64
	Page     int
	PageSize int
	Items    []DownloadHistory
}

// FileInfo 文件信息
type FileInfo struct {
	FilePath string
	FileName string
	FileSize int64
	FileHash string
}
