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
	TaskID   string   `json:"task_id"`
	UserID   string   `json:"user_id"`
	URL      string   `json:"url"`
	Mode     string   `json:"mode"`    // quick_download, archive
	Quality  string   `json:"quality"` // 1080p, 720p, etc.
	Format   string   `json:"format"`  // mp4, webm
	Metadata Metadata `json:"metadata"`
}

// Metadata 视频元数据
type Metadata struct {
	Title    string `json:"title"`
	Duration int64  `json:"duration"`
	Platform string `json:"platform"`
}

// ProgressMessage 进度消息
type ProgressMessage struct {
	TaskID          string  `json:"task_id"`
	Status          string  `json:"status"` // downloading, completed, failed
	Percent         float64 `json:"percent"`
	DownloadedBytes int64   `json:"downloaded_bytes"`
	TotalBytes      int64   `json:"total_bytes"`
	Speed           string  `json:"speed"`
	ETA             string  `json:"eta"`
	Message         string  `json:"message"`
}

// Progress yt-dlp 解析的进度
type Progress struct {
	Percent float64
	Speed   string
	ETA     string
}
