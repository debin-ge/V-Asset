package utils

import (
	"log"
	"sort"
	"strconv"
	"strings"
)

// VideoFormat yt-dlp返回的格式信息
type VideoFormat struct {
	FormatID       string  `json:"format_id"`
	URL            string  `json:"url"`
	Ext            string  `json:"ext"`
	Resolution     string  `json:"resolution"`
	Filesize       int64   `json:"filesize"`
	FilesizeApprox int64   `json:"filesize_approx"` // 估算文件大小（当filesize不可用时）
	FPS            float64 `json:"fps"`
	VCodec         string  `json:"vcodec"`
	ACodec         string  `json:"acodec"`
	Height         int     `json:"height"`
	Width          int     `json:"width"`
	VBR            float64 `json:"vbr"` // 视频码率 kbps
	ABR            float64 `json:"abr"` // 音频码率 kbps
	ASR            int     `json:"asr"` // 音频采样率 Hz
}

// NormalizedFormat 标准化后的格式信息
type NormalizedFormat struct {
	FormatID   string
	Quality    string // 1080p, 720p, etc.
	Extension  string
	Filesize   int64
	Height     int
	Width      int
	FPS        float64
	VideoCodec string
	AudioCodec string
	VBR        float64 // 视频码率 kbps
	ABR        float64 // 音频码率 kbps
	ASR        int     // 音频采样率 Hz
	Score      int     // 优先级分数
}

// NormalizeFormats 标准化视频格式列表
// 保留所有格式（包括纯视频、纯音频、合并格式），前端根据codec字段区分
func NormalizeFormats(rawFormats []VideoFormat) []NormalizedFormat {
	log.Printf("[DEBUG-NormalizeFormats] Input: %d raw formats", len(rawFormats))

	var result []NormalizedFormat
	var videoCount, audioCount, skippedCount int
	var maxHeight int

	for i, f := range rawFormats {
		// 跳过 storyboard 等非媒体格式
		if f.VCodec == "none" && f.ACodec == "none" {
			skippedCount++
			continue
		}

		// 提取分辨率高度（音频格式height为0是正常的）
		height := f.Height
		if height == 0 && f.VCodec != "none" && f.VCodec != "" {
			height = extractHeight(f.Resolution)
		}

		if height > maxHeight {
			maxHeight = height
		}

		// 计算优先级得分
		score := calculateScore(f, height)

		// 构造质量标签
		quality := ""
		if height > 0 {
			quality = formatQuality(height)
			videoCount++
		} else if f.ABR > 0 {
			// 音频格式使用码率作为质量描述
			quality = "audio"
			audioCount++
		}

		// 获取文件大小（优先使用精确值，回退到估算值）
		filesize := f.Filesize
		if filesize == 0 {
			filesize = f.FilesizeApprox
		}

		// 调试前5个格式
		if i < 5 {
			log.Printf("[DEBUG-NormalizeFormats] Format[%d]: id=%s, height=%d, vcodec=%s, acodec=%s, filesize=%d",
				i, f.FormatID, height, f.VCodec, f.ACodec, filesize)
		}

		result = append(result, NormalizedFormat{
			FormatID:   f.FormatID,
			Quality:    quality,
			Extension:  f.Ext,
			Filesize:   filesize,
			Height:     height,
			Width:      f.Width,
			FPS:        f.FPS,
			VideoCodec: f.VCodec,
			AudioCodec: f.ACodec,
			VBR:        f.VBR,
			ABR:        f.ABR,
			ASR:        f.ASR,
			Score:      score,
		})
	}

	// 按得分排序(从高到低)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	log.Printf("[DEBUG-NormalizeFormats] Output: %d formats (video=%d, audio=%d, skipped=%d, maxHeight=%d)",
		len(result), videoCount, audioCount, skippedCount, maxHeight)

	return result
}

// extractHeight 从分辨率字符串提取高度
func extractHeight(resolution string) int {
	if resolution == "" {
		return 0
	}

	// 格式: "1920x1080" 或 "1080p"
	parts := strings.Split(resolution, "x")
	if len(parts) == 2 {
		height, _ := strconv.Atoi(parts[1])
		return height
	}

	// 尝试解析 "1080p" 格式
	resolution = strings.TrimSuffix(resolution, "p")
	height, _ := strconv.Atoi(resolution)
	return height
}

// formatQuality 将高度转换为质量标签
func formatQuality(height int) string {
	switch {
	case height >= 2160:
		return "4K"
	case height >= 1440:
		return "2K"
	case height >= 1080:
		return "1080p"
	case height >= 720:
		return "720p"
	case height >= 480:
		return "480p"
	case height >= 360:
		return "360p"
	default:
		return "240p"
	}
}

// calculateScore 计算格式优先级得分
func calculateScore(f VideoFormat, height int) int {
	score := 0

	// 分辨率得分(主要因素)
	score += height * 10

	// 编码格式得分
	switch f.VCodec {
	case "h264", "avc1":
		score += 100 // H.264最通用
	case "vp9":
		score += 80
	case "hevc", "h265":
		score += 90
	}

	// 音频编码得分
	if f.ACodec != "none" && f.ACodec != "" {
		score += 50
	}

	// 扩展名得分
	if f.Ext == "mp4" {
		score += 30 // MP4最通用
	}

	// FPS得分
	if f.FPS >= 60 {
		score += 20
	} else if f.FPS >= 30 {
		score += 10
	}

	return score
}
