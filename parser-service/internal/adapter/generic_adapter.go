package adapter

import (
	"vasset/parser-service/internal/ytdlp"
)

// GenericAdapter 通用平台适配器
type GenericAdapter struct {
	ytdlp *ytdlp.Wrapper
}

// NewGenericAdapter 创建通用适配器
func NewGenericAdapter(wrapper *ytdlp.Wrapper) *GenericAdapter {
	return &GenericAdapter{
		ytdlp: wrapper,
	}
}

// Parse 解析视频(使用默认参数)
func (a *GenericAdapter) Parse(url string) (*ytdlp.VideoInfo, error) {
	return a.ytdlp.ExtractInfo(url, "")
}
