package service

import (
	"context"
	"testing"
	"time"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/models"
)

type fakeQuotaRepository struct {
	getOrCreateResp   *models.UserQuota
	getOrCreateErr    error
	updateErr         error
	consumeQuotaResp  *models.UserQuota
	consumeQuotaErr   error
	refundQuotaResp   *models.UserQuota
	refundQuotaErr    error
	refundQuotaUserID string
}

func (f *fakeQuotaRepository) GetOrCreate(context.Context, string, int) (*models.UserQuota, error) {
	return f.getOrCreateResp, f.getOrCreateErr
}

func (f *fakeQuotaRepository) Update(context.Context, *models.UserQuota) error {
	return f.updateErr
}

func (f *fakeQuotaRepository) ConsumeQuotaSafe(context.Context, string, int) (*models.UserQuota, error) {
	return f.consumeQuotaResp, f.consumeQuotaErr
}

func (f *fakeQuotaRepository) RefundQuotaSafe(_ context.Context, userID string, _ int) (*models.UserQuota, error) {
	f.refundQuotaUserID = userID
	return f.refundQuotaResp, f.refundQuotaErr
}

func TestRefundQuotaDelegatesToRepository(t *testing.T) {
	t.Parallel()

	repo := &fakeQuotaRepository{
		refundQuotaResp: &models.UserQuota{
			UserID:     "user-1",
			DailyLimit: 5,
			DailyUsed:  2,
			ResetAt:    time.Now().Add(12 * time.Hour),
		},
	}
	svc := &QuotaService{
		quotaRepo: repo,
		cfg: &config.QuotaConfig{
			DefaultDailyLimit: 5,
		},
	}

	quota, err := svc.RefundQuota(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("RefundQuota returned error: %v", err)
	}

	if repo.refundQuotaUserID != "user-1" {
		t.Fatalf("expected refund for user-1, got %q", repo.refundQuotaUserID)
	}

	if quota.DailyUsed != 2 {
		t.Fatalf("expected refunded quota daily_used=2, got %d", quota.DailyUsed)
	}

	if remaining := svc.GetRemaining(quota); remaining != 3 {
		t.Fatalf("expected remaining=3, got %d", remaining)
	}
}

func TestGetRemainingNeverNegative(t *testing.T) {
	t.Parallel()

	svc := &QuotaService{}
	if remaining := svc.GetRemaining(&models.UserQuota{DailyLimit: 1, DailyUsed: 3}); remaining != 0 {
		t.Fatalf("expected remaining=0, got %d", remaining)
	}
}
