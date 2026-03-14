package service

import (
	"testing"

	"vasset/asset-service/internal/models"
)

func TestSetOrderAwaitingShortfallMarksOrder(t *testing.T) {
	t.Parallel()

	order := &models.BillingChargeOrder{
		Scene:             models.BillingSceneDownload,
		HeldAmountFen:     100,
		CapturedAmountFen: 20,
		ReleasedAmountFen: 10,
	}

	setOrderAwaitingShortfall(order, 35, "awaiting shortfall resolution: ingress capture")

	if order.ShortfallFen != 35 {
		t.Fatalf("expected shortfall 35, got %d", order.ShortfallFen)
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
		ShortfallFen: 15,
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
		CapturedAmountFen:  120,
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
		HeldAmountFen:     180,
		CapturedAmountFen: 120,
		ReleasedAmountFen: 0,
		ShortfallFen:      0,
	}

	status := deriveOrderStatus(order)
	if status != models.BillingOrderStatusPartialCaptured {
		t.Fatalf("expected partial captured status, got %d", status)
	}
}
