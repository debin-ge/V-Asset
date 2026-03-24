package handler

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"youdlp/asset-service/internal/config"
	"youdlp/asset-service/internal/models"
	"youdlp/asset-service/internal/money"
	"youdlp/asset-service/internal/service"
	pb "youdlp/asset-service/proto"
)

type inMemoryWelcomeCreditRepo struct {
	settings *models.WelcomeCreditSettings
}

func (r *inMemoryWelcomeCreditRepo) GetWelcomeCreditSettings(context.Context) (*models.WelcomeCreditSettings, error) {
	if r.settings == nil {
		r.settings = &models.WelcomeCreditSettings{
			Enabled:      true,
			AmountYuan:   money.MustParse("1.00"),
			CurrencyCode: "CNY",
			UpdatedAt:    time.Now(),
			UpdatedBy:    "system",
		}
	}
	copy := *r.settings
	return &copy, nil
}

func (r *inMemoryWelcomeCreditRepo) UpsertWelcomeCreditSettings(_ context.Context, settings *models.WelcomeCreditSettings) (*models.WelcomeCreditSettings, error) {
	r.settings = &models.WelcomeCreditSettings{
		Enabled:      settings.Enabled,
		AmountYuan:   settings.AmountYuan,
		CurrencyCode: settings.CurrencyCode,
		UpdatedAt:    time.Now(),
		UpdatedBy:    settings.UpdatedBy,
	}
	copy := *r.settings
	return &copy, nil
}

func TestWelcomeCreditSettingsRPC(t *testing.T) {
	t.Parallel()

	repo := &inMemoryWelcomeCreditRepo{}
	welcomeSvc := service.NewWelcomeCreditService(repo)
	server := NewGRPCServer(nil, nil, nil, nil, welcomeSvc, nil, nil, &config.Config{})

	updateResp, err := server.UpdateWelcomeCreditSettings(context.Background(), &pb.UpdateWelcomeCreditSettingsRequest{
		Enabled:      true,
		AmountYuan:   "2.50",
		CurrencyCode: "cny",
		UpdatedBy:    "admin-user",
	})
	if err != nil {
		t.Fatalf("update rpc failed: %v", err)
	}
	if !updateResp.Success {
		t.Fatal("expected update success=true")
	}
	if got := updateResp.GetSettings().GetAmountYuan(); got != "2.5" {
		t.Fatalf("expected amount_yuan=2.5, got %s", got)
	}
	if got := updateResp.GetSettings().GetCurrencyCode(); got != "CNY" {
		t.Fatalf("expected currency_code=CNY, got %s", got)
	}

	getResp, err := server.GetWelcomeCreditSettings(context.Background(), &pb.GetWelcomeCreditSettingsRequest{})
	if err != nil {
		t.Fatalf("get rpc failed: %v", err)
	}
	if got := getResp.GetSettings().GetUpdatedBy(); got != "admin-user" {
		t.Fatalf("expected updated_by=admin-user, got %s", got)
	}

	_, err = server.UpdateWelcomeCreditSettings(context.Background(), &pb.UpdateWelcomeCreditSettingsRequest{
		Enabled:      true,
		AmountYuan:   "-0.5",
		CurrencyCode: "CNY",
		UpdatedBy:    "admin-user",
	})
	if err == nil {
		t.Fatal("expected invalid argument error for negative amount")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", status.Code(err))
	}
}
