package adapter

import (
	"vasset/parser-service/internal/ytdlp"
)

// Adapter 平台适配器接口
type Adapter interface {
	// Parse 解析视频URL（使用静态 cookie）
	Parse(url string) (*ytdlp.VideoInfo, error)

	// ParseWithCookie 解析视频URL（使用动态 cookie）
	ParseWithCookie(url string, cookieFile string) (*ytdlp.VideoInfo, error)

	// ParseWithProxyAndCookie 解析视频URL（使用动态 proxy 和 cookie）
	ParseWithProxyAndCookie(url, proxyURL, cookieFile string) (*ytdlp.VideoInfo, error)
}
