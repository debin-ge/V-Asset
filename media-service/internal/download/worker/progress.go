package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"vasset/media-service/internal/download/models"
	"vasset/media-service/internal/download/ytdlp"
)

// ProgressPublisher 进度发布器
type ProgressPublisher struct {
	redis *redis.Client
}

// NewProgressPublisher 创建进度发布器
func NewProgressPublisher(redisClient *redis.Client) *ProgressPublisher {
	return &ProgressPublisher{
		redis: redisClient,
	}
}

// Publish 发布进度消息
func (p *ProgressPublisher) Publish(ctx context.Context, msg *models.ProgressMessage) error {
	channel := fmt.Sprintf("progress:%s", msg.TaskID)

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal progress message: %w", err)
	}

	if err := p.redis.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish progress: %w", err)
	}

	log.Printf("[Progress] Published to %s: %.2f%% phase=%s", channel, msg.Percent, msg.Phase)
	return nil
}

// phaseLabel 阶段中文标签映射
func phaseLabel(phase ytdlp.DownloadPhase) string {
	switch phase {
	case ytdlp.PhaseDownloadingVideo:
		return "正在下载视频流"
	case ytdlp.PhaseDownloadingAudio:
		return "正在下载音频流"
	case ytdlp.PhaseDownloading:
		return "正在下载"
	case ytdlp.PhaseMerging:
		return "正在合并音视频"
	case ytdlp.PhaseProcessing:
		return "正在处理文件"
	default:
		return "下载中"
	}
}

// PublishDownloading 发布下载中状态（带阶段和加权进度）
func (p *ProgressPublisher) PublishDownloading(ctx context.Context, taskID string, progress *models.Progress, phase ytdlp.DownloadPhase, overallPercent float64) error {
	msg := &models.ProgressMessage{
		TaskID:          taskID,
		Status:          "downloading",
		Percent:         overallPercent,
		Phase:           string(phase),
		PhaseLabel:      phaseLabel(phase),
		DownloadedBytes: progress.DownloadedBytes,
		TotalBytes:      progress.TotalBytes,
		Speed:           progress.Speed,
		ETA:             progress.ETA,
	}
	return p.Publish(ctx, msg)
}

// PublishPhase 发布阶段切换消息（无具体下载进度时使用）
func (p *ProgressPublisher) PublishPhase(ctx context.Context, taskID string, phase ytdlp.DownloadPhase, percent float64) error {
	msg := &models.ProgressMessage{
		TaskID:     taskID,
		Status:     "downloading",
		Percent:    percent,
		Phase:      string(phase),
		PhaseLabel: phaseLabel(phase),
	}
	return p.Publish(ctx, msg)
}

// PublishStarted 发布下载开始状态，避免长时间静默导致前端看不到任何进度
func (p *ProgressPublisher) PublishStarted(ctx context.Context, taskID string, message string) error {
	msg := &models.ProgressMessage{
		TaskID:  taskID,
		Status:  "downloading",
		Percent: 0,
		Message: message,
	}
	return p.Publish(ctx, msg)
}

// PublishCompleted 发布完成状态
func (p *ProgressPublisher) PublishCompleted(ctx context.Context, taskID, message string) error {
	msg := &models.ProgressMessage{
		TaskID:  taskID,
		Status:  "completed",
		Percent: 100,
		Message: message,
	}
	return p.Publish(ctx, msg)
}

// PublishFailed 发布失败状态
func (p *ProgressPublisher) PublishFailed(ctx context.Context, taskID, errorMsg string) error {
	msg := &models.ProgressMessage{
		TaskID:  taskID,
		Status:  "failed",
		Message: errorMsg,
	}
	return p.Publish(ctx, msg)
}
