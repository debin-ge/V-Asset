package service

import (
	"testing"

	"youdlp/asset-service/internal/models"
	"youdlp/asset-service/internal/money"
)

func TestSetOrderAwaitingShortfallMarksOrder(t *testing.T) {
	t.Parallel()

	order := &models.BillingChargeOrder{
		Scene:             models.BillingSceneDownload,
		HeldAmountFen:     money.FromInt64(100),
		CapturedAmountFen: money.FromInt64(20),
		ReleasedAmountFen: money.FromInt64(10),
	}

	setOrderAwaitingShortfall(order, money.FromInt64(35), "awaiting shortfall resolution: ingress capture")

	if order.ShortfallFen.Cmp(money.FromInt64(35)) != 0 {
		t.Fatalf("expected shortfall 35, got %s", order.ShortfallFen.String())
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
		Scene:        models.BillingSceneDownload,
		Status:       models.BillingOrderStatusAwaitingShortfall,
		ShortfallFen: money.FromInt64(15),
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
		CapturedAmountFen:  money.FromInt64(120),
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
		Scene:             models.BillingSceneDownload,
		HeldAmountFen:     money.FromInt64(180),
		CapturedAmountFen: money.FromInt64(120),
		ReleasedAmountFen: money.Zero(),
		ShortfallFen:      money.Zero(),
	}

	status := deriveOrderStatus(order)
	if status != models.BillingOrderStatusPartialCaptured {
		t.Fatalf("expected partial captured status, got %d", status)
	}
}
