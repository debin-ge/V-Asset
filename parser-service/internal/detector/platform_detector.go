package detector

import (
	"regexp"

	"vasset/parser-service/internal/utils"
)

// PlatformDetector 平台检测器
type PlatformDetector struct {
	patterns map[string]*regexp.Regexp
}

// NewPlatformDetector 创建平台检测器
func NewPlatformDetector() *PlatformDetector {
	return &PlatformDetector{
		patterns: map[string]*regexp.Regexp{
			"youtube":   regexp.MustCompile(`^https?://(www\.)?(youtube\.com|youtu\.be)/`),
			"tiktok":    regexp.MustCompile(`^https?://.*tiktok\.com/`),
			"bilibili":  regexp.MustCompile(`^https?://.*bilibili\.com/`),
			"twitter":   regexp.MustCompile(`^https?://(www\.)?(twitter\.com|x\.com)/`),
			"instagram": regexp.MustCompile(`^https?://(www\.)?instagram\.com/`),
		},
	}
}

// Detect 检测URL所属平台
func (d *PlatformDetector) Detect(url string) (string, error) {
	// 先验证URL格式
	if !utils.IsValidURL(url) {
		return "", utils.ErrInvalidURL
	}

	// 匹配已知平台
	for platform, pattern := range d.patterns {
		if pattern.MatchString(url) {
			return platform, nil
		}
	}

	// 未匹配到特定平台,使用通用适配器
	return "generic", nil
}
