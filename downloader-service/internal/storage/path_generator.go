package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"vasset/downloader-service/internal/config"
	"vasset/downloader-service/internal/models"
)

// PathGenerator 路径生成器
type PathGenerator struct {
	basePath string
}

// NewPathGenerator 创建路径生成器
func NewPathGenerator(cfg *config.StorageConfig) *PathGenerator {
	return &PathGenerator{
		basePath: cfg.BasePath,
	}
}

// GeneratePath 生成文件存储路径
func (g *PathGenerator) GeneratePath(task *models.DownloadTask) (string, error) {
	// 清理文件名
	safeTitle := sanitizeFilename(task.Metadata.Title)
	if safeTitle == "" {
		safeTitle = task.TaskID
	}

	var filePath string

	switch task.Mode {
	case "quick_download":
		// 临时文件: /data/vasset/tmp/{task_id}/video.mp4
		filePath = filepath.Join(
			g.basePath, "tmp", task.TaskID,
			fmt.Sprintf("%s.%s", safeTitle, task.Format),
		)

	case "archive":
		// 归档文件: /data/vasset/archive/{user_id}/{YYYYMMDD}/video_{timestamp}.mp4
		date := time.Now().Format("20060102")
		timestamp := time.Now().Unix()

		filePath = filepath.Join(
			g.basePath, "archive",
			fmt.Sprintf("%d", task.UserID),
			date,
			fmt.Sprintf("%s_%d.%s", safeTitle, timestamp, task.Format),
		)

	default:
		// 默认使用临时路径
		filePath = filepath.Join(
			g.basePath, "tmp", task.TaskID,
			fmt.Sprintf("%s.%s", safeTitle, task.Format),
		)
	}

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	return filePath, nil
}

// GetFileName 从路径获取文件名
func (g *PathGenerator) GetFileName(filePath string) string {
	return filepath.Base(filePath)
}

// sanitizeFilename 清理文件名,移除非法字符
func sanitizeFilename(name string) string {
	if name == "" {
		return ""
	}

	// 移除非法字符
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	clean := reg.ReplaceAllString(name, "_")

	// 移除首尾空格和点
	clean = regexp.MustCompile(`^[\s.]+|[\s.]+$`).ReplaceAllString(clean, "")

	// 限制长度
	maxLen := 200
	if len(clean) > maxLen {
		clean = clean[:maxLen]
	}

	return clean
}
