package adapter

import (
	"context"

	"youdlp/media-service/internal/ytdlp"
)

// BilibiliAdapter Bilibili平台适配器
type BilibiliAdapter struct {
	ytdlp      *ytdlp.Wrapper
	cookieFile string
	args       []string
}

// NewBilibiliAdapter 创建Bilibili适配器
func NewBilibiliAdapter(wrapper *ytdlp.Wrapper, cookieFile string, extraArgs []string) *BilibiliAdapter {
	return &BilibiliAdapter{
		ytdlp:      wrapper,
		cookieFile: cookieFile,
		args:       extraArgs,
	}
}

// Parse 解析Bilibili视频（使用静态 cookie）
func (a *BilibiliAdapter) Parse(ctx context.Context, url string) (*ytdlp.VideoInfo, error) {
	return a.ytdlp.ExtractInfo(ctx, url, a.cookieFile, a.args...)
}

// ParseWithCookie 解析Bilibili视频（使用动态 cookie）
func (a *BilibiliAdapter) ParseWithCookie(ctx context.Context, url string, cookieFile string) (*ytdlp.VideoInfo, error) {
	cookie := cookieFile
	if cookie == "" {
		cookie = a.cookieFile
	}
	return a.ytdlp.ExtractInfo(ctx, url, cookie, a.args...)
}

// ParseWithProxyAndCookie 解析Bilibili视频（使用动态 proxy 和 cookie）
func (a *BilibiliAdapter) ParseWithProxyAndCookie(ctx context.Context, url, proxyURL, cookieFile string) (*ytdlp.VideoInfo, error) {
	cookie := cookieFile
	if cookie == "" {
		cookie = a.cookieFile
	}
	return a.ytdlp.ExtractInfoWithProxy(ctx, url, proxyURL, cookie, a.args...)
}
