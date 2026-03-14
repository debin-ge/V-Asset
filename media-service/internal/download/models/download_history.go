package models

import (
	"database/sql"
	"time"
)

// 状态常量
const (
	StatusPending        = 0 // 待处理
	StatusProcessing     = 1 // 下载中
	StatusCompleted      = 2 // 完成
	StatusFailed         = 3 // 失败
	StatusPendingCleanup = 4 // 待清理(quick_download完成后)
	StatusExpired        = 5 // 已过期(已清理)
)

// StatusText 状态文本映射
var StatusText = map[int]string{
	StatusPending:        "pending",
	StatusProcessing:     "processing",
	StatusCompleted:      "completed",
	StatusFailed:         "failed",
	StatusPendingCleanup: "pending_cleanup",
	StatusExpired:        "expired",
}

// DownloadHistory 下载历史记录模型
type DownloadHistory struct {
	ID           int64          `json:"id"`
	TaskID       string         `json:"task_id"`
	UserID       string         `json:"user_id"`
	URL          string         `json:"url"`
	Platform     string         `json:"platform"`
	Title        string         `json:"title"`
	Mode         string         `json:"mode"`    // quick_download, archive
	Quality      string         `json:"quality"` // 1080p, 720p, etc.
	FilePath     sql.NullString `json:"file_path"`
	FileName     sql.NullString `json:"file_name"`
	FileSize     sql.NullInt64  `json:"file_size"` // 字节
	FileHash     sql.NullString `json:"file_hash"` // MD5
	Status       int            `json:"status"`
	ErrorMessage sql.NullString `json:"error_message"`
	RetryCount   int            `json:"retry_count"`
	ExpireAt     sql.NullTime   `json:"expire_at"`
	CreatedAt    time.Time      `json:"created_at"`
	StartedAt    sql.NullTime   `json:"started_at"`
	CompletedAt  sql.NullTime   `json:"completed_at"`
}

// DownloadTask MQ 任务消息结构
type DownloadTask struct {
	TaskID         string          `json:"task_id"`
	UserID         string          `json:"user_id"`
	HistoryID      int64           `json:"history_id"`
	URL            string          `json:"url"`
	Mode           string          `json:"mode"`    // quick_download, archive
	Quality        string          `json:"quality"` // 1080p, 720p, 160kbps, etc.
	Format         string          `json:"format"`  // mp4, webm, m4a
	FormatID       string          `json:"format_id"`
	SelectedFormat *SelectedFormat `json:"selected_format,omitempty"`
	Platform       string          `json:"platform"`
	Title          string          `json:"title"`
	Metadata       Metadata        `json:"metadata"`
	CookieID       int64           `json:"cookie_id"`       // parser 使用的 cookie ID
	ProxyURL       string          `json:"proxy_url"`       // parser 使用的 proxy URL
	ProxyLeaseID   string          `json:"proxy_lease_id"`  // parser 使用的动态代理租约 ID
	ProxyExpireAt  string          `json:"proxy_expire_at"` // parser 获取到的代理过期时间
}

// Metadata 视频元数据
type Metadata struct {
	Title    string `json:"title"`
	Duration int64  `json:"duration"`
	Platform string `json:"platform"`
}

// SelectedFormat 选中的精确格式信息
type SelectedFormat struct {
	FormatID   string  `json:"format_id"`
	Quality    string  `json:"quality,omitempty"`
	Extension  string  `json:"extension,omitempty"`
	Filesize   int64   `json:"filesize,omitempty"`
	Height     int32   `json:"height,omitempty"`
	Width      int32   `json:"width,omitempty"`
	FPS        float64 `json:"fps,omitempty"`
	VideoCodec string  `json:"video_codec,omitempty"`
	AudioCodec string  `json:"audio_codec,omitempty"`
	VBR        float64 `json:"vbr,omitempty"`
	ABR        float64 `json:"abr,omitempty"`
	ASR        int32   `json:"asr,omitempty"`
}

// ProgressMessage 进度消息
type ProgressMessage struct {
	TaskID          string  `json:"task_id"`
	Status          string  `json:"status"` // downloading, completed, failed
	Percent         float64 `json:"percent"`
	Phase           string  `json:"phase,omitempty"`       // downloading_video, downloading_audio, downloading, merging, processing
	PhaseLabel      string  `json:"phase_label,omitempty"` // 中文阶段标签
	DownloadedBytes int64   `json:"downloaded_bytes"`
	TotalBytes      int64   `json:"total_bytes"`
	Speed           string  `json:"speed"`
	ETA             string  `json:"eta"`
	Message         string  `json:"message"`
}

// Progress yt-dlp 解析的进度
type Progress struct {
	Percent         float64
	DownloadedBytes int64
	TotalBytes      int64
	Speed           string
	ETA             string
}
