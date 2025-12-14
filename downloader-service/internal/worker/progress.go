package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"vasset/downloader-service/internal/models"
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

	log.Printf("[Progress] Published to %s: %.2f%%", channel, msg.Percent)
	return nil
}

// PublishDownloading 发布下载中状态
func (p *ProgressPublisher) PublishDownloading(ctx context.Context, taskID string, progress *models.Progress) error {
	msg := &models.ProgressMessage{
		TaskID:  taskID,
		Status:  "downloading",
		Percent: progress.Percent,
		Speed:   progress.Speed,
		ETA:     progress.ETA,
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
