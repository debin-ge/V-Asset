package utils

import (
	"errors"
	"strings"
)

var (
	// URL相关错误
	ErrInvalidURL          = errors.New("invalid URL")
	ErrUnsupportedPlatform = errors.New("unsupported platform")

	// 视频相关错误
	ErrVideoNotFound  = errors.New("video not found")
	ErrVideoPrivate   = errors.New("video is private")
	ErrVideoDeleted   = errors.New("video has been deleted")
	ErrGeoRestricted  = errors.New("video is geo-restricted")
	ErrAgeRestricted  = errors.New("video is age-restricted")
	ErrCopyrightClaim = errors.New("video removed due to copyright claim")

	// 系统相关错误
	ErrTimeout       = errors.New("parse timeout")
	ErrCacheMiss     = errors.New("cache miss")
	ErrYTDLPNotFound = errors.New("yt-dlp binary not found")
	ErrYTDLPFailed   = errors.New("yt-dlp execution failed")
)

// MapYTDLPError 将yt-dlp的错误输出映射到具体错误
func MapYTDLPError(stderr string) error {
	lowerStderr := strings.ToLower(stderr)

	switch {
	case strings.Contains(lowerStderr, "video unavailable"):
		return ErrVideoNotFound
	case strings.Contains(lowerStderr, "private video"):
		return ErrVideoPrivate
	case strings.Contains(lowerStderr, "has been deleted"):
		return ErrVideoDeleted
	case strings.Contains(lowerStderr, "not available in your country"):
		return ErrGeoRestricted
	case strings.Contains(lowerStderr, "age-restricted"):
		return ErrAgeRestricted
	case strings.Contains(lowerStderr, "copyright"):
		return ErrCopyrightClaim
	case strings.Contains(lowerStderr, "no such file"):
		return ErrYTDLPNotFound
	case strings.Contains(lowerStderr, "timed out") || strings.Contains(lowerStderr, "timeout"):
		return ErrTimeout
	default:
		return ErrYTDLPFailed
	}
}
