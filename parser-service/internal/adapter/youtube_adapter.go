package adapter

import (
	"vasset/parser-service/internal/ytdlp"
)

// YouTubeAdapter YouTube平台适配器
type YouTubeAdapter struct {
	ytdlp      *ytdlp.Wrapper
	cookieFile string
	args       []string
}

// NewYouTubeAdapter 创建YouTube适配器
func NewYouTubeAdapter(wrapper *ytdlp.Wrapper, cookieFile string, extraArgs []string) *YouTubeAdapter {
	return &YouTubeAdapter{
		ytdlp:      wrapper,
		cookieFile: cookieFile,
		args:       extraArgs,
	}
}

// Parse 解析YouTube视频（使用静态 cookie）
func (a *YouTubeAdapter) Parse(url string) (*ytdlp.VideoInfo, error) {
	return a.ytdlp.ExtractInfo(url, a.cookieFile, a.args...)
}

// ParseWithCookie 解析YouTube视频（使用动态 cookie）
func (a *YouTubeAdapter) ParseWithCookie(url string, cookieFile string) (*ytdlp.VideoInfo, error) {
	// 优先使用传入的 cookie，如果为空则使用静态 cookie
	cookie := cookieFile
	if cookie == "" {
		cookie = a.cookieFile
	}
	return a.ytdlp.ExtractInfo(url, cookie, a.args...)
}

// ParseWithProxyAndCookie 解析YouTube视频（使用动态 proxy 和 cookie）
func (a *YouTubeAdapter) ParseWithProxyAndCookie(url, proxyURL, cookieFile string) (*ytdlp.VideoInfo, error) {
	// 优先使用传入的 cookie，如果为空则使用静态 cookie
	cookie := cookieFile
	if cookie == "" {
		cookie = a.cookieFile
	}
	return a.ytdlp.ExtractInfoWithProxy(url, proxyURL, cookie, a.args...)
}
