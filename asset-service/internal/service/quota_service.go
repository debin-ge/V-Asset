package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/repository"
)

// QuotaService 配额服务
type QuotaService struct {
	quotaRepo *repository.QuotaRepository
	cfg       *config.QuotaConfig
}

// NewQuotaService 创建配额服务
func NewQuotaService(quotaRepo *repository.QuotaRepository, cfg *config.QuotaConfig) *QuotaService {
	return &QuotaService{
		quotaRepo: quotaRepo,
		cfg:       cfg,
	}
}

// CheckQuota 检查配额
func (s *QuotaService) CheckQuota(ctx context.Context, userID string) (*models.UserQuota, error) {
	quota, err := s.quotaRepo.GetOrCreate(ctx, userID, s.cfg.DefaultDailyLimit)
	if err != nil {
		return nil, err
	}

	// 检查是否需要重置
	now := time.Now()
	if now.After(quota.ResetAt) {
		quota.DailyUsed = 0
		quota.ResetAt = getNextMidnight(now)
		if err := s.quotaRepo.Update(ctx, quota); err != nil {
			return nil, err
		}
	}

	return quota, nil
}

// ConsumeQuota 消费配额
func (s *QuotaService) ConsumeQuota(ctx context.Context, userID string) (*models.UserQuota, error) {
	quota, err := s.quotaRepo.ConsumeQuotaSafe(ctx, userID, s.cfg.DefaultDailyLimit)
	if err != nil {
		if err == sql.ErrNoRows {
			return quota, errors.New("daily quota exceeded")
		}
		return nil, err
	}
	return quota, nil
}

// GetRemaining 获取剩余配额
func (s *QuotaService) GetRemaining(quota *models.UserQuota) int {
	remaining := quota.DailyLimit - quota.DailyUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// getNextMidnight 获取下一个午夜时间
func getNextMidnight(t time.Time) time.Time {
	year, month, day := t.Add(24 * time.Hour).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
