package models

// ParseAPIResponse 第三方 API /parse 响应
type ParseAPIResponse struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Duration  float64  `json:"duration"`
	Formats   []Format `json:"formats"`
	Thumbnail string   `json:"thumbnail"`
	Uploader  string   `json:"uploader"`
	ViewCount int64    `json:"view_count"`
}

// Format 视频格式
type Format struct {
	FormatID       string  `json:"format_id"`
	Ext            string  `json:"ext"`
	Width          int     `json:"width,omitempty"`
	Height         int     `json:"height,omitempty"`
	Filesize       int64   `json:"filesize,omitempty"`
	FilesizeApprox int64   `json:"filesize_approx,omitempty"`
	VCodec         string  `json:"vcodec"`
	ACodec         string  `json:"acodec"`
	FormatNote     string  `json:"format_note"`
	Resolution     string  `json:"resolution"`
	VBR            float64 `json:"vbr,omitempty"`
	ABR            float64 `json:"abr,omitempty"`
	ASR            int     `json:"asr,omitempty"`
	FPS            float64 `json:"fps,omitempty"`
}

// StreamRequest /stream 请求体
type StreamRequest struct {
	URL      string `json:"url"`
	FormatID string `json:"format_id"`
	Name     string `json:"name"`
	Ext      string `json:"ext,omitempty"`
	IsVideo  bool   `json:"is_video"`
}

// IsVideoFormat 判断是否为视频格式
func (f *Format) IsVideoFormat() bool {
	return f.VCodec != "none" && f.VCodec != ""
}

// IsAudioFormat 判断是否为音频格式
func (f *Format) IsAudioFormat() bool {
	return f.ACodec != "none" && f.ACodec != ""
}

// GetFilesize 获取文件大小
func (f *Format) GetFilesize() int64 {
	if f.Filesize > 0 {
		return f.Filesize
	}
	return f.FilesizeApprox
}

// GetQuality 获取质量描述
func (f *Format) GetQuality() string {
	if f.Height > 0 {
		return f.FormatNote
	}
	return f.FormatNote
}
