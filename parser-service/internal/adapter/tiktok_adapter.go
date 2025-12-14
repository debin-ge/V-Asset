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

// Parse 解析TikTok视频
func (a *TikTokAdapter) Parse(url string) (*ytdlp.VideoInfo, error) {
	return a.ytdlp.ExtractInfo(url, a.cookieFile, a.args...)
}
