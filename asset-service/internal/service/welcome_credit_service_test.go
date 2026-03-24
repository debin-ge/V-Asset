package service

import (
	"context"
	"testing"

	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/money"
)

type fakeWelcomeCreditRepository struct {
	settings *models.WelcomeCreditSettings
}

func (f *fakeWelcomeCreditRepository) GetWelcomeCreditSettings(context.Context) (*models.WelcomeCreditSettings, error) {
	return f.settings, nil
}

func (f *fakeWelcomeCreditRepository) UpsertWelcomeCreditSettings(_ context.Context, settings *models.WelcomeCreditSettings) (*models.WelcomeCreditSettings, error) {
	f.settings = settings
	return settings, nil
}

func TestWelcomeCreditSettingsRejectsNegativeAmount(t *testing.T) {
	t.Parallel()

	repo := &fakeWelcomeCreditRepository{}
	svc := NewWelcomeCreditService(repo)

	negativeAmount, err := money.Parse("-0.01")
	if err != nil {
		t.Fatalf("parse negative amount failed: %v", err)
	}

	_, err = svc.UpdateWelcomeCreditSettings(context.Background(), true, negativeAmount, "CNY", "tester")
	if err == nil {
		t.Fatal("expected negative amount validation error")
	}
	if err != ErrInvalidWelcomeCreditAmount {
		t.Fatalf("expected ErrInvalidWelcomeCreditAmount, got %v", err)
	}
}
