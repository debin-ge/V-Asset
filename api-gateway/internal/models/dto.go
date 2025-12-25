package models

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname" binding:"required,min=2,max=50"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// UserResponse 用户信息响应
type UserResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      int32  `json:"role"`
	CreatedAt string `json:"created_at"`
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required,min=2,max=50"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ParseRequest 解析请求
type ParseRequest struct {
	URL       string `json:"url" binding:"required"`
	SkipCache bool   `json:"skip_cache"`
}

// ParseResponse 解析响应
type ParseResponse struct {
	VideoID     string        `json:"video_id"`
	Platform    string        `json:"platform"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Duration    int64         `json:"duration"`
	Thumbnail   string        `json:"thumbnail"`
	Author      string        `json:"author"`
	UploadDate  string        `json:"upload_date,omitempty"`
	ViewCount   int64         `json:"view_count,omitempty"`
	Formats     []VideoFormat `json:"formats"`
}

// VideoFormat 视频格式
type VideoFormat struct {
	FormatID   string  `json:"format_id"`
	Quality    string  `json:"quality"`
	Extension  string  `json:"ext"`
	Filesize   int64   `json:"filesize"`
	Height     int32   `json:"height,omitempty"`
	FPS        float64 `json:"fps,omitempty"`
	VideoCodec string  `json:"video_codec,omitempty"`
	AudioCodec string  `json:"audio_codec,omitempty"`
}

// DownloadRequest 下载请求
type DownloadRequest struct {
	URL     string `json:"url" binding:"required"`
	Mode    string `json:"mode" binding:"required,oneof=quick_download archive"` // quick_download 或 archive
	Quality string `json:"quality"`                                              // 1080p, 720p, 480p, etc.
	Format  string `json:"format"`                                               // mp4, webm
}

// DownloadResponse 下载响应
type DownloadResponse struct {
	TaskID        string `json:"task_id"`
	HistoryID     int64  `json:"history_id"`
	EstimatedTime int    `json:"estimated_time"` // 预计耗时(秒)
}

// HistoryRequest 历史查询请求
type HistoryRequest struct {
	Status    int    `form:"status"`
	Platform  string `form:"platform"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
	SortBy    string `form:"sort_by,default=created_at"`
	SortOrder string `form:"sort_order,default=desc"`
}

// HistoryItem 历史记录项
type HistoryItem struct {
	HistoryID   int64  `json:"history_id"`
	TaskID      string `json:"task_id,omitempty"`
	URL         string `json:"url"`
	Platform    string `json:"platform"`
	Title       string `json:"title"`
	Mode        string `json:"mode"`
	Quality     string `json:"quality"`
	FileSize    int64  `json:"file_size"`
	Status      int32  `json:"status"`
	FileName    string `json:"file_name,omitempty"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	Duration    int64  `json:"duration,omitempty"`
	Author      string `json:"author,omitempty"`
}

// QuotaResponse 配额响应
type QuotaResponse struct {
	DailyLimit int32  `json:"daily_limit"`
	DailyUsed  int32  `json:"daily_used"`
	Remaining  int32  `json:"remaining"`
	ResetAt    string `json:"reset_at"`
}

// FileDownloadRequest 文件下载请求
type FileDownloadRequest struct {
	HistoryID int64 `form:"history_id" binding:"required"`
}

// StatsResponse 用户统计响应
type StatsResponse struct {
	TotalDownloads   int64           `json:"total_downloads"`
	SuccessDownloads int64           `json:"success_downloads"`
	FailedDownloads  int64           `json:"failed_downloads"`
	TotalSizeBytes   int64           `json:"total_size_bytes"`
	TopPlatforms     []PlatformStat  `json:"top_platforms"`
	RecentActivity   []DailyActivity `json:"recent_activity"`
}

// PlatformStat 平台统计
type PlatformStat struct {
	Platform string `json:"platform"`
	Count    int64  `json:"count"`
}

// DailyActivity 日活动
type DailyActivity struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// ==================== 代理管理 ====================

// CreateProxyRequest 创建代理请求
type CreateProxyRequest struct {
	IP          string `json:"ip" binding:"required"`
	Port        int32  `json:"port" binding:"required"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Protocol    string `json:"protocol"`
	Region      string `json:"region"`
	CheckHealth bool   `json:"check_health"`
}

// CreateProxyResponse 创建代理响应
type CreateProxyResponse struct {
	ID                int64  `json:"id"`
	HealthCheckPassed bool   `json:"health_check_passed"`
	HealthCheckError  string `json:"health_check_error,omitempty"`
}

// UpdateProxyRequest 更新代理请求
type UpdateProxyRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
	Region   string `json:"region"`
}

// ListProxiesRequest 列出代理请求
type ListProxiesRequest struct {
	Status   int32  `form:"status"`
	Protocol string `form:"protocol"`
	Region   string `form:"region"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

// ProxyInfo 代理信息
type ProxyInfo struct {
	ID              int64  `json:"id"`
	IP              string `json:"ip"`
	Port            int32  `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Protocol        string `json:"protocol"`
	Region          string `json:"region"`
	Status          int32  `json:"status"`
	LastCheckAt     string `json:"last_check_at,omitempty"`
	LastCheckResult string `json:"last_check_result,omitempty"`
	SuccessCount    int64  `json:"success_count"`
	FailCount       int64  `json:"fail_count"`
	LastUsedAt      string `json:"last_used_at,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ProxyHealthCheckResponse 代理健康检查响应
type ProxyHealthCheckResponse struct {
	Healthy   bool   `json:"healthy"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}

// ==================== Cookie 管理 ====================

// CreateCookieRequest 创建 Cookie 请求
type CreateCookieRequest struct {
	Platform      string `json:"platform" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Content       string `json:"content" binding:"required"`
	ExpireAt      string `json:"expire_at"`
	FreezeSeconds int32  `json:"freeze_seconds"`
}

// CreateCookieResponse 创建 Cookie 响应
type CreateCookieResponse struct {
	ID int64 `json:"id"`
}

// UpdateCookieRequest 更新 Cookie 请求
type UpdateCookieRequest struct {
	Name          string `json:"name"`
	Content       string `json:"content"`
	ExpireAt      string `json:"expire_at"`
	FreezeSeconds int32  `json:"freeze_seconds"`
}

// ListCookiesRequest 列出 Cookie 请求
type ListCookiesRequest struct {
	Platform string `form:"platform"`
	Status   int32  `form:"status"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

// CookieInfo Cookie 信息
type CookieInfo struct {
	ID            int64  `json:"id"`
	Platform      string `json:"platform"`
	Name          string `json:"name"`
	Content       string `json:"content"`
	Status        int32  `json:"status"`
	ExpireAt      string `json:"expire_at,omitempty"`
	FrozenUntil   string `json:"frozen_until,omitempty"`
	FreezeSeconds int32  `json:"freeze_seconds"`
	LastUsedAt    string `json:"last_used_at,omitempty"`
	UseCount      int64  `json:"use_count"`
	SuccessCount  int64  `json:"success_count"`
	FailCount     int64  `json:"fail_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// FreezeCookieRequest 冷冻 Cookie 请求
type FreezeCookieRequest struct {
	FreezeSeconds int32 `json:"freeze_seconds"`
}

// FreezeCookieResponse 冷冻 Cookie 响应
type FreezeCookieResponse struct {
	Success     bool   `json:"success"`
	FrozenUntil string `json:"frozen_until"`
}
