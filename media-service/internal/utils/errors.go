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

const (
	ErrorCategoryNetworkTimeout   = "network_timeout"
	ErrorCategoryProxyAuth        = "proxy_auth"
	ErrorCategoryProxyUnreachable = "proxy_unreachable"
	ErrorCategoryRateLimited      = "rate_limited"
	ErrorCategoryBotDetected      = "bot_detected"
	ErrorCategoryCookieInvalid    = "cookie_invalid"
	ErrorCategoryTerminalVideo    = "terminal_video"
	ErrorCategoryUnknown          = "unknown"
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

// IsProxyOrBotRetryableError 判断解析失败是否适合通过更换代理重试。
func IsProxyOrBotRetryableError(err error) bool {
	category := ClassifyAccessError(err)
	return category == ErrorCategoryNetworkTimeout ||
		category == ErrorCategoryProxyAuth ||
		category == ErrorCategoryProxyUnreachable ||
		category == ErrorCategoryRateLimited ||
		category == ErrorCategoryBotDetected
}

func ClassifyAccessError(err error) string {
	if err == nil {
		return ""
	}
	if isTerminalVideoError(err) {
		return ErrorCategoryTerminalVideo
	}
	if errors.Is(err, ErrTimeout) {
		return ErrorCategoryNetworkTimeout
	}

	text := normalizeRetryableErrorText(err.Error())
	switch {
	case containsAny(text, cookieInvalidKeywords):
		return ErrorCategoryCookieInvalid
	case containsAny(text, proxyAuthKeywords):
		return ErrorCategoryProxyAuth
	case containsAny(text, rateLimitKeywords):
		return ErrorCategoryRateLimited
	case containsAny(text, botDetectionKeywords):
		return ErrorCategoryBotDetected
	case containsAny(text, timeoutKeywords):
		return ErrorCategoryNetworkTimeout
	case containsAny(text, proxyRetryableKeywords):
		return ErrorCategoryProxyUnreachable
	default:
		return ErrorCategoryUnknown
	}
}

func isTerminalVideoError(err error) bool {
	return errors.Is(err, ErrVideoNotFound) ||
		errors.Is(err, ErrVideoPrivate) ||
		errors.Is(err, ErrVideoDeleted) ||
		errors.Is(err, ErrGeoRestricted) ||
		errors.Is(err, ErrAgeRestricted) ||
		errors.Is(err, ErrCopyrightClaim)
}

func normalizeRetryableErrorText(text string) string {
	replacer := strings.NewReplacer(
		"’", "'",
		"‘", "'",
		"“", "\"",
		"”", "\"",
	)
	return replacer.Replace(strings.ToLower(text))
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

var botDetectionKeywords = []string{
	"sign in to confirm you're not a bot",
	"not a bot",
	"use --cookies-from-browser or --cookies",
	"cookies for the authentication",
	"captcha",
	"verify you are human",
	"verify that you are human",
	"verify human",
	"unusual traffic",
	"automated queries",
	"challenge",
	"cloudflare",
}

var rateLimitKeywords = []string{
	"too many requests",
	"http error 429",
	"429 too many",
	"rate limit",
	"rate-limit",
}

var proxyAuthKeywords = []string{
	"proxy authentication required",
	"http error 407",
	"407 proxy",
	"proxy auth",
}

var timeoutKeywords = []string{
	"timed out",
	"timeout",
	"deadline exceeded",
	"context deadline exceeded",
	"connection timed out",
	"connect timeout",
	"read timeout",
	"timeout awaiting response headers",
	"tls handshake timeout",
}

var cookieInvalidKeywords = []string{
	"invalid cookie",
	"cookies are no longer valid",
	"cookie has expired",
	"cookies have expired",
	"login cookies",
}

var proxyRetryableKeywords = []string{
	"proxy",
	"connection refused",
	"connection reset",
	"tunnel",
	"socks",
	"eof",
	"unexpected eof",
	"no route to host",
	"network is unreachable",
	"temporary failure in name resolution",
}
