package adapter

import (
	"vasset/parser-service/internal/ytdlp"
)

// Adapter 平台适配器接口
type Adapter interface {
	// Parse 解析视频URL
	Parse(url string) (*ytdlp.VideoInfo, error)
}
