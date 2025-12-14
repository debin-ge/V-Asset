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

// Parse 解析YouTube视频
func (a *YouTubeAdapter) Parse(url string) (*ytdlp.VideoInfo, error) {
	return a.ytdlp.ExtractInfo(url, a.cookieFile, a.args...)
}
