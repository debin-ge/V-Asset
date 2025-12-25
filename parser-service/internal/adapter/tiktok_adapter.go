package adapter

import (
	"vasset/parser-service/internal/ytdlp"
)

// TikTokAdapter TikTok平台适配器
type TikTokAdapter struct {
	ytdlp      *ytdlp.Wrapper
	cookieFile string
	args       []string
}

// NewTikTokAdapter 创建TikTok适配器
func NewTikTokAdapter(wrapper *ytdlp.Wrapper, cookieFile string, extraArgs []string) *TikTokAdapter {
	return &TikTokAdapter{
		ytdlp:      wrapper,
		cookieFile: cookieFile,
		args:       extraArgs,
	}
}

// Parse 解析TikTok视频（使用静态 cookie）
func (a *TikTokAdapter) Parse(url string) (*ytdlp.VideoInfo, error) {
	return a.ytdlp.ExtractInfo(url, a.cookieFile, a.args...)
}

// ParseWithCookie 解析TikTok视频（使用动态 cookie）
func (a *TikTokAdapter) ParseWithCookie(url string, cookieFile string) (*ytdlp.VideoInfo, error) {
	cookie := cookieFile
	if cookie == "" {
		cookie = a.cookieFile
	}
	return a.ytdlp.ExtractInfo(url, cookie, a.args...)
}

// ParseWithProxyAndCookie 解析TikTok视频（使用动态 proxy 和 cookie）
func (a *TikTokAdapter) ParseWithProxyAndCookie(url, proxyURL, cookieFile string) (*ytdlp.VideoInfo, error) {
	cookie := cookieFile
	if cookie == "" {
		cookie = a.cookieFile
	}
	return a.ytdlp.ExtractInfoWithProxy(url, proxyURL, cookie, a.args...)
}
