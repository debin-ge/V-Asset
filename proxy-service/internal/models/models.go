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

// ProgressAPIResponse 第三方 API /progress 响应
type ProgressAPIResponse struct {
	TaskID          string  `json:"task_id"`
	Status          string  `json:"status"`           // pending, downloading, merging, completed, failed
	Progress        float64 `json:"progress"`         // 0-100
	Speed           string  `json:"speed"`            // 下载速度，如 "2.5MB/s"
	ETA             int     `json:"eta"`              // 预计剩余时间(秒)
	Error           string  `json:"error,omitempty"`  // 错误信息
	Filename        string  `json:"filename"`         // 文件名
	TotalBytes      int64   `json:"total_bytes"`      // 总字节数
	DownloadedBytes int64   `json:"downloaded_bytes"` // 已下载字节数
}
