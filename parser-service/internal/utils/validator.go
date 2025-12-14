package utils

import (
	"net/url"
	"strings"
)

// IsValidURL 验证URL格式是否有效
func IsValidURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// 必须是http或https协议
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// 必须有host
	if u.Host == "" {
		return false
	}

	return true
}

// NormalizeURL 标准化URL(去除追踪参数等)
func NormalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// 移除常见的追踪参数
	q := u.Query()
	trackingParams := []string{"utm_source", "utm_medium", "utm_campaign", "fbclid", "gclid"}
	for _, param := range trackingParams {
		q.Del(param)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// SanitizeString 清理字符串中的特殊字符
func SanitizeString(s string) string {
	// 去除首尾空白
	s = strings.TrimSpace(s)

	// 替换多个空白为单个空格
	s = strings.Join(strings.Fields(s), " ")

	return s
}
