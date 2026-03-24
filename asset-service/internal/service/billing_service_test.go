package service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"youdlp/asset-service/internal/models"
	"youdlp/asset-service/internal/money"
	"youdlp/asset-service/internal/repository"
)

func TestSetOrderAwaitingShortfallMarksOrder(t *testing.T) {
	t.Parallel()

	order := &models.BillingChargeOrder{
		Scene:              models.BillingSceneDownload,
		HeldAmountYuan:     money.FromInt64(100),
		CapturedAmountYuan: money.FromInt64(20),
		ReleasedAmountYuan: money.FromInt64(10),
	}

	setOrderAwaitingShortfall(order, money.FromInt64(35), "awaiting shortfall resolution: ingress capture")

	if order.ShortfallYuan.Cmp(money.FromInt64(35)) != 0 {
		t.Fatalf("expected shortfall 35, got %s", order.ShortfallYuan.String())
	}
	if order.Status != models.BillingOrderStatusAwaitingShortfall {
		t.Fatalf("expected awaiting shortfall status, got %d", order.Status)
	}
	if order.Remark == "" {
		t.Fatal("expected shortfall remark to be set")
	}
}

func TestCanUseInitialOrderRejectsAwaitingShortfall(t *testing.T) {
	t.Parallel()

	order := &models.BillingChargeOrder{
		Scene:         models.BillingSceneDownload,
		Status:        models.BillingOrderStatusAwaitingShortfall,
		ShortfallYuan: money.FromInt64(15),
	}

	if canUseInitialOrder(order) {
		t.Fatal("expected awaiting shortfall order to be blocked for initial transfer")
	}
}

func TestCanUseInitialOrderAllowsCapturedIngressBeforeFirstTransfer(t *testing.T) {
	t.Parallel()

	order := &models.BillingChargeOrder{
		Scene:              models.BillingSceneDownload,
		Status:             models.BillingOrderStatusCaptured,
		CapturedAmountYuan: money.FromInt64(120),
		ActualIngressBytes: 1024,
		ActualEgressBytes:  0,
	}

	if !canUseInitialOrder(order) {
		t.Fatal("expected initial order to remain reusable before first file transfer")
	}
}

func TestDeriveOrderStatusReturnsPartialCapturedAfterShortfallCleared(t *testing.T) {
	t.Parallel()

	order := &models.BillingChargeOrder{
		Scene:              models.BillingSceneDownload,
		HeldAmountYuan:     money.FromInt64(180),
		CapturedAmountYuan: money.FromInt64(120),
		ReleasedAmountYuan: money.Zero(),
		ShortfallYuan:      money.Zero(),
	}

	status := deriveOrderStatus(order)
	if status != models.BillingOrderStatusPartialCaptured {
		t.Fatalf("expected partial captured status, got %d", status)
	}
}

func TestCalculateAmountYuan_Minimum100MBAndCeilTwoDecimals(t *testing.T) {
	t.Parallel()

	price := money.MustParse("1.23")

	underMinimum, err := calculateAmountYuan(50*mbBytes, price)
	if err != nil {
		t.Fatalf("calculate under minimum failed: %v", err)
	}
	if underMinimum.Cmp(money.MustParse("0.13")) != 0 {
		t.Fatalf("expected under-minimum amount 0.13, got %s", underMinimum.String())
	}

	overMinimum, err := calculateAmountYuan(250*mbBytes, price)
	if err != nil {
		t.Fatalf("calculate over minimum failed: %v", err)
	}
	if overMinimum.Cmp(money.MustParse("0.31")) != 0 {
		t.Fatalf("expected over-minimum amount 0.31, got %s", overMinimum.String())
	}
}

func TestEstimateDownloadBilling_UnknownFilesizeReturnsZeroEstimate(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, version, ingress_price_yuan_per_gb, egress_price_yuan_per_gb`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "version", "ingress_price_yuan_per_gb", "egress_price_yuan_per_gb",
			"enabled", "remark", "updated_by_user_id", "effective_at", "created_at",
		}).AddRow(1, 7, "1.00", "1.00", true, "test-pricing", "system", now, now))

	svc := NewBillingService(
		repository.NewBillingRepository(db),
		repository.NewWelcomeCreditSettingsRepository(db),
	)

	estimate, pricing, err := svc.EstimateDownloadBilling(context.Background(), 0)
	if err != nil {
		t.Fatalf("estimate failed: %v", err)
	}
	if pricing == nil {
		t.Fatal("expected pricing to be returned")
	}
	if estimate.EstimatedCostYuan.Cmp(money.Zero()) != 0 {
		t.Fatalf("expected estimated cost 0, got %s", estimate.EstimatedCostYuan.String())
	}
	if estimate.EstimateReason != "unknown_filesize" {
		t.Fatalf("expected reason unknown_filesize, got %s", estimate.EstimateReason)
	}
	if !estimate.IsEstimated {
		t.Fatal("expected IsEstimated=true when filesize is unknown")
	}
	if estimate.EstimatedTrafficBytes != 0 {
		t.Fatalf("expected estimated traffic bytes 0, got %d", estimate.EstimatedTrafficBytes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
