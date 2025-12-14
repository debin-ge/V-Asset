package adapter

import (
	"vasset/parser-service/internal/ytdlp"
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

// Parse 解析Bilibili视频
func (a *BilibiliAdapter) Parse(url string) (*ytdlp.VideoInfo, error) {
	return a.ytdlp.ExtractInfo(url, a.cookieFile, a.args...)
}
