package worker

import "errors"

// 错误定义
var (
	ErrProxyUnavailable  = errors.New("proxy unavailable")
	ErrDownloadTimeout   = errors.New("download timeout")
	ErrVideoNotFound     = errors.New("video not found")
	ErrInsufficientSpace = errors.New("insufficient disk space")
	ErrTaskCancelled     = errors.New("task cancelled")
)
