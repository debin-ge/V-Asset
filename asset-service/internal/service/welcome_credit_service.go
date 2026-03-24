package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/money"
)

var ErrInvalidWelcomeCreditAmount = errors.New("welcome credit amount cannot be negative")

type welcomeCreditRepository interface {
	GetWelcomeCreditSettings(ctx context.Context) (*models.WelcomeCreditSettings, error)
	UpsertWelcomeCreditSettings(ctx context.Context, settings *models.WelcomeCreditSettings) (*models.WelcomeCreditSettings, error)
}

type WelcomeCreditService struct {
	repo welcomeCreditRepository
}

func NewWelcomeCreditService(repo welcomeCreditRepository) *WelcomeCreditService {
	return &WelcomeCreditService{repo: repo}
}

func (s *WelcomeCreditService) GetWelcomeCreditSettings(ctx context.Context) (*models.WelcomeCreditSettings, error) {
	return s.repo.GetWelcomeCreditSettings(ctx)
}

func (s *WelcomeCreditService) UpdateWelcomeCreditSettings(ctx context.Context, enabled bool, amountYuan money.Decimal, currencyCode, updatedBy string) (*models.WelcomeCreditSettings, error) {
	if amountYuan.Cmp(money.Zero()) < 0 {
		return nil, ErrInvalidWelcomeCreditAmount
	}

	normalizedCurrencyCode := strings.ToUpper(strings.TrimSpace(currencyCode))
	if normalizedCurrencyCode == "" {
		return nil, fmt.Errorf("currency code is required")
	}

	normalizedUpdatedBy := strings.TrimSpace(updatedBy)
	if normalizedUpdatedBy == "" {
		normalizedUpdatedBy = "system"
	}

	return s.repo.UpsertWelcomeCreditSettings(ctx, &models.WelcomeCreditSettings{
		Enabled:      enabled,
		AmountYuan:   amountYuan,
		CurrencyCode: normalizedCurrencyCode,
		UpdatedBy:    normalizedUpdatedBy,
	})
}
