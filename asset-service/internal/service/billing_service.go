package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"youdlp/asset-service/internal/models"
	"youdlp/asset-service/internal/money"
	"youdlp/asset-service/internal/repository"
)

const (
	mbBytes       = int64(1000 * 1000)
	minBillableMB = int64(100)
	gbMB          = int64(1000)
)

var defaultWelcomeCreditSettings = &models.WelcomeCreditSettings{
	Enabled:      true,
	AmountYuan:   money.MustParse("1.00"),
	CurrencyCode: "CNY",
	UpdatedBy:    "system",
}

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrDuplicateOperation  = errors.New("operation id already used for another billing event")
)

// BillingService 账务服务
type BillingService struct {
	repo              *repository.BillingRepository
	welcomeCreditRepo *repository.WelcomeCreditSettingsRepository
}

// NewBillingService 创建账务服务
func NewBillingService(repo *repository.BillingRepository, welcomeCreditRepo *repository.WelcomeCreditSettingsRepository) *BillingService {
	return &BillingService{repo: repo, welcomeCreditRepo: welcomeCreditRepo}
}

func (s *BillingService) GetBillingAccount(ctx context.Context, userID string, autoCreate bool) (*models.BillingAccount, error) {
	if autoCreate {
		return s.repo.GetOrCreateAccount(ctx, userID)
	}
	return s.repo.GetAccountByUserID(ctx, userID)
}

func (s *BillingService) ListBillingStatements(ctx context.Context, userID string, page, pageSize int, statementType, statementStatus int32) (*models.BillingStatementResult, error) {
	return s.repo.ListStatements(ctx, userID, page, pageSize, statementType, statementStatus)
}

func (s *BillingService) GrantWelcomeCredit(ctx context.Context, userID, operationID string) (*models.BillingAccount, *models.BillingLedgerEntry, *models.WelcomeCreditGrant, bool, error) {
	if userID == "" {
		return nil, nil, nil, false, fmt.Errorf("user id is required")
	}
	if operationID == "" {
		return nil, nil, nil, false, fmt.Errorf("operation id is required")
	}

	account, err := s.repo.GetOrCreateAccount(ctx, userID)
	if err != nil {
		return nil, nil, nil, false, err
	}

	existingEntry, err := s.repo.GetLedgerByOperationID(ctx, operationID)
	if err == nil {
		if existingEntry.UserID != userID || existingEntry.Remark != models.WelcomeCreditReasonCode {
			return nil, nil, nil, false, ErrDuplicateOperation
		}
		grant, grantErr := s.repo.GetWelcomeCreditGrantByOperationID(ctx, operationID)
		if grantErr != nil && !errors.Is(grantErr, sql.ErrNoRows) {
			return nil, nil, nil, false, grantErr
		}
		return account, existingEntry, grant, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil, false, err
	}

	settings, err := s.getEffectiveWelcomeCreditSettings(ctx)
	if err != nil {
		return nil, nil, nil, false, err
	}
	if !settings.Enabled || settings.AmountYuan.Cmp(money.Zero()) <= 0 {
		return account, nil, nil, false, nil
	}

	amountYuan := settings.AmountYuan
	if amountYuan.Cmp(money.Zero()) <= 0 {
		return account, nil, nil, false, nil
	}

	var (
		entry *models.BillingLedgerEntry
		grant *models.WelcomeCreditGrant
	)
	err = s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		account, err = s.repo.GetOrCreateAccountTx(ctx, tx, userID)
		if err != nil {
			return err
		}

		account.AvailableBalanceYuan = account.AvailableBalanceYuan.Add(amountYuan)
		account.TotalRechargedYuan = account.TotalRechargedYuan.Add(amountYuan)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		now := time.Now()
		entry = &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    userID,
			OperationID:               operationID,
			EntryType:                 models.LedgerEntryTypeManualTopup,
			Scene:                     models.BillingSceneOnboarding,
			ActionAmountYuan:          abs64(amountYuan),
			AvailableDeltaYuan:        amountYuan,
			ReservedDeltaYuan:         money.Zero(),
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			Remark:                    models.WelcomeCreditReasonCode,
			CreatedAt:                 now,
		}
		if err := s.repo.CreateLedgerTx(ctx, tx, entry); err != nil {
			return err
		}

		grant = &models.WelcomeCreditGrant{
			UserID:        userID,
			OperationID:   operationID,
			LedgerEntryNo: entry.EntryNo,
			ReasonCode:    models.WelcomeCreditReasonCode,
			AmountYuan:    settings.AmountYuan,
			CurrencyCode:  settings.CurrencyCode,
			CreatedAt:     now,
		}
		return s.repo.CreateWelcomeCreditGrantTx(ctx, tx, grant)
	})
	if err != nil {
		if existingEntry, existingErr := s.repo.GetLedgerByOperationID(ctx, operationID); existingErr == nil {
			if existingEntry.UserID != userID || existingEntry.Remark != models.WelcomeCreditReasonCode {
				return nil, nil, nil, false, ErrDuplicateOperation
			}
			grant, grantErr := s.repo.GetWelcomeCreditGrantByOperationID(ctx, operationID)
			if grantErr != nil && !errors.Is(grantErr, sql.ErrNoRows) {
				return nil, nil, nil, false, grantErr
			}
			account, accErr := s.repo.GetOrCreateAccount(ctx, userID)
			return account, existingEntry, grant, false, accErr
		}
		return nil, nil, nil, false, err
	}

	return account, entry, grant, true, nil
}

func (s *BillingService) getEffectiveWelcomeCreditSettings(ctx context.Context) (*models.WelcomeCreditSettings, error) {
	settings, err := s.welcomeCreditRepo.GetWelcomeCreditSettings(ctx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return s.welcomeCreditRepo.UpsertWelcomeCreditSettings(ctx, defaultWelcomeCreditSettings)
	}

	if shouldBootstrapWelcomeCreditSettings(settings) {
		return s.welcomeCreditRepo.UpsertWelcomeCreditSettings(ctx, defaultWelcomeCreditSettings)
	}

	return settings, nil
}

func shouldBootstrapWelcomeCreditSettings(settings *models.WelcomeCreditSettings) bool {
	if settings == nil {
		return true
	}

	return strings.TrimSpace(settings.CurrencyCode) == "" || strings.TrimSpace(settings.UpdatedBy) == "" || settings.UpdatedAt.IsZero()
}

func (s *BillingService) EstimateDownloadBilling(ctx context.Context, selectedFormatFilesize int64) (*models.BillingEstimate, *models.BillingPricing, error) {
	pricing, err := s.repo.GetActivePricing(ctx)
	if err != nil {
		return nil, nil, err
	}

	fileBytes := selectedFormatFilesize
	if fileBytes <= 0 {
		return &models.BillingEstimate{
			EstimatedIngressBytes: 0,
			EstimatedEgressBytes:  0,
			EstimatedTrafficBytes: 0,
			EstimatedCostYuan:     money.Zero(),
			PricingVersion:        pricing.Version,
			IsEstimated:           true,
			EstimateReason:        "unknown_filesize",
		}, pricing, nil
	}

	ingressCost, err := calculateAmountYuan(fileBytes, pricing.IngressPriceYuanPerGB)
	if err != nil {
		return nil, nil, err
	}
	egressCost, err := calculateAmountYuan(fileBytes, pricing.EgressPriceYuanPerGB)
	if err != nil {
		return nil, nil, err
	}

	return &models.BillingEstimate{
		EstimatedIngressBytes: fileBytes,
		EstimatedEgressBytes:  fileBytes,
		EstimatedTrafficBytes: fileBytes * 2,
		EstimatedCostYuan:     ingressCost.Add(egressCost),
		PricingVersion:        pricing.Version,
		IsEstimated:           false,
		EstimateReason:        "",
	}, pricing, nil
}

func (s *BillingService) HoldInitialDownload(ctx context.Context, userID string, historyID int64, taskID string, estimate *models.BillingEstimate) (*models.BillingChargeOrder, *models.BillingHold, *models.BillingAccount, error) {
	var (
		order   *models.BillingChargeOrder
		hold    *models.BillingHold
		account *models.BillingAccount
	)

	err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		existingOrder, err := s.repo.GetOrderByTaskIDForUpdate(ctx, tx, taskID)
		if err == nil {
			order = existingOrder
			hold, err = s.repo.GetHoldByTaskIDForUpdate(ctx, tx, taskID, models.BillingHoldTypeDownloadTotal)
			if err != nil {
				return err
			}
			account, err = s.repo.GetOrCreateAccountTx(ctx, tx, userID)
			return err
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		account, err = s.repo.GetOrCreateAccountTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		if account.AvailableBalanceYuan.Cmp(estimate.EstimatedCostYuan) < 0 {
			return ErrInsufficientBalance
		}

		account.AvailableBalanceYuan = account.AvailableBalanceYuan.Sub(estimate.EstimatedCostYuan)
		account.ReservedBalanceYuan = account.ReservedBalanceYuan.Add(estimate.EstimatedCostYuan)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		now := time.Now()
		order = &models.BillingChargeOrder{
			OrderNo:               newBillingID("ord"),
			UserID:                userID,
			HistoryID:             historyID,
			TaskID:                taskID,
			Scene:                 models.BillingSceneDownload,
			Status:                models.BillingOrderStatusHeld,
			PricingVersion:        estimate.PricingVersion,
			EstimatedIngressBytes: estimate.EstimatedIngressBytes,
			EstimatedEgressBytes:  estimate.EstimatedEgressBytes,
			EstimatedTrafficBytes: estimate.EstimatedTrafficBytes,
			HeldAmountYuan:        estimate.EstimatedCostYuan,
			Remark:                "initial download hold",
			CreatedAt:             now,
			UpdatedAt:             now,
		}
		if err := s.repo.CreateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		hold = &models.BillingHold{
			HoldNo:             newBillingID("hold"),
			OrderNo:            order.OrderNo,
			UserID:             userID,
			HistoryID:          historyID,
			TaskID:             taskID,
			HoldType:           models.BillingHoldTypeDownloadTotal,
			FundingSource:      models.BillingFundingSourceNewReserve,
			Status:             models.BillingHoldStatusHeld,
			AmountYuan:         estimate.EstimatedCostYuan,
			CapturedAmountYuan: money.Zero(),
			ReleasedAmountYuan: money.Zero(),
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := s.repo.CreateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    userID,
			OrderNo:                   order.OrderNo,
			HoldNo:                    hold.HoldNo,
			HistoryID:                 historyID,
			TaskID:                    taskID,
			EntryType:                 models.LedgerEntryTypeHold,
			Scene:                     models.BillingSceneDownload,
			ActionAmountYuan:          estimate.EstimatedCostYuan,
			AvailableDeltaYuan:        estimate.EstimatedCostYuan.Neg(),
			ReservedDeltaYuan:         estimate.EstimatedCostYuan,
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			Remark:                    "hold initial download",
			CreatedAt:                 now,
		}
		return s.repo.CreateLedgerTx(ctx, tx, entry)
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return order, hold, account, nil
}

func (s *BillingService) CaptureIngressUsage(ctx context.Context, taskID string, actualIngressBytes int64) (*models.BillingChargeOrder, money.Decimal, error) {
	var (
		order          *models.BillingChargeOrder
		capturedAmount money.Decimal
	)

	err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		order, err = s.repo.GetOrderByTaskIDForUpdate(ctx, tx, taskID)
		if err != nil {
			return err
		}
		hold, err := s.repo.GetHoldByTaskIDForUpdate(ctx, tx, taskID, models.BillingHoldTypeDownloadTotal)
		if err != nil {
			return err
		}
		account, err := s.repo.GetOrCreateAccountTx(ctx, tx, order.UserID)
		if err != nil {
			return err
		}

		pricing, err := s.repo.GetPricingByVersion(ctx, order.PricingVersion)
		if err != nil {
			return err
		}
		capturedAmount, err = calculateAmountYuan(actualIngressBytes, pricing.IngressPriceYuanPerGB)
		if err != nil {
			return err
		}
		if order.ActualIngressBytes > 0 {
			if order.ActualIngressBytes != actualIngressBytes {
				return fmt.Errorf("ingress usage already recorded for task %s", taskID)
			}
			if order.CapturedAmountYuan.Cmp(money.Zero()) > 0 && order.ShortfallYuan.IsZero() {
				capturedAmount = order.CapturedAmountYuan
				return nil
			}
		}

		additionalReserve := money.Zero()
		if remaining := remainingOrderReserve(order); remaining.Cmp(capturedAmount) < 0 {
			additionalReserve = capturedAmount.Sub(remaining)
			if account.AvailableBalanceYuan.Cmp(additionalReserve) < 0 {
				if order.ActualIngressBytes == 0 {
					order.ActualIngressBytes = actualIngressBytes
					order.ActualTrafficBytes += actualIngressBytes
				}
				setOrderAwaitingShortfall(order, additionalReserve, "awaiting shortfall resolution: ingress capture")
				if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
					return err
				}
				return ErrInsufficientBalance
			}
			account.AvailableBalanceYuan = account.AvailableBalanceYuan.Sub(additionalReserve)
			account.ReservedBalanceYuan = account.ReservedBalanceYuan.Add(additionalReserve)
			order.HeldAmountYuan = order.HeldAmountYuan.Add(additionalReserve)
			hold.AmountYuan = hold.AmountYuan.Add(additionalReserve)
		}

		if order.ActualIngressBytes == 0 {
			order.ActualIngressBytes = actualIngressBytes
			order.ActualTrafficBytes += actualIngressBytes
		}
		order.ShortfallYuan = money.Zero()
		order.CapturedAmountYuan = order.CapturedAmountYuan.Add(capturedAmount)
		order.Status = deriveOrderStatus(order)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		hold.CapturedAmountYuan = hold.CapturedAmountYuan.Add(capturedAmount)
		hold.Status = deriveHoldStatus(hold)
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		account.ReservedBalanceYuan = account.ReservedBalanceYuan.Sub(capturedAmount)
		account.TotalSpentYuan = account.TotalSpentYuan.Add(capturedAmount)
		account.TotalTrafficBytes += actualIngressBytes
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		now := time.Now()
		if additionalReserve.Cmp(money.Zero()) > 0 {
			holdEntry := &models.BillingLedgerEntry{
				EntryNo:                   newBillingID("led"),
				AccountID:                 account.ID,
				UserID:                    order.UserID,
				OrderNo:                   order.OrderNo,
				HoldNo:                    hold.HoldNo,
				HistoryID:                 order.HistoryID,
				TaskID:                    order.TaskID,
				EntryType:                 models.LedgerEntryTypeHold,
				Scene:                     order.Scene,
				ActionAmountYuan:          additionalReserve,
				AvailableDeltaYuan:        additionalReserve.Neg(),
				ReservedDeltaYuan:         additionalReserve,
				BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
				BalanceAfterReservedYuan:  account.ReservedBalanceYuan.Add(capturedAmount),
				Remark:                    "top up ingress reserve",
				CreatedAt:                 now,
			}
			if err := s.repo.CreateLedgerTx(ctx, tx, holdEntry); err != nil {
				return err
			}
		}

		usage := &models.TrafficUsageRecord{
			UsageNo:            newBillingID("use"),
			OrderNo:            order.OrderNo,
			UserID:             order.UserID,
			HistoryID:          order.HistoryID,
			TaskID:             order.TaskID,
			Direction:          models.TrafficDirectionIngress,
			TrafficBytes:       actualIngressBytes,
			UnitPriceYuanPerGB: pricing.IngressPriceYuanPerGB,
			AmountYuan:         capturedAmount,
			PricingVersion:     pricing.Version,
			SourceService:      "media-service",
			Status:             models.TrafficUsageStatusConfirmed,
			ConfirmedAt:        &now,
		}
		if err := s.repo.CreateUsageTx(ctx, tx, usage); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    order.UserID,
			OrderNo:                   order.OrderNo,
			HoldNo:                    hold.HoldNo,
			HistoryID:                 order.HistoryID,
			TaskID:                    order.TaskID,
			EntryType:                 models.LedgerEntryTypeCapture,
			Scene:                     order.Scene,
			ActionAmountYuan:          capturedAmount,
			AvailableDeltaYuan:        money.Zero(),
			ReservedDeltaYuan:         capturedAmount.Neg(),
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			Remark:                    "capture ingress usage",
			CreatedAt:                 now,
		}
		return s.repo.CreateLedgerTx(ctx, tx, entry)
	})
	if err != nil {
		return nil, money.Zero(), err
	}

	return order, capturedAmount, nil
}

func (s *BillingService) ReleaseInitialDownload(ctx context.Context, taskID, reason string) (*models.BillingChargeOrder, money.Decimal, error) {
	var (
		order          *models.BillingChargeOrder
		releasedAmount money.Decimal
	)

	err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		order, err = s.repo.GetOrderByTaskIDForUpdate(ctx, tx, taskID)
		if err != nil {
			return err
		}
		hold, err := s.repo.GetHoldByTaskIDForUpdate(ctx, tx, taskID, models.BillingHoldTypeDownloadTotal)
		if err != nil {
			return err
		}
		account, err := s.repo.GetOrCreateAccountTx(ctx, tx, order.UserID)
		if err != nil {
			return err
		}

		releasedAmount = remainingHoldAmount(hold)
		if releasedAmount.IsZero() {
			return nil
		}

		hold.ReleasedAmountYuan = hold.ReleasedAmountYuan.Add(releasedAmount)
		hold.Status = models.BillingHoldStatusReleased
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		order.ReleasedAmountYuan = order.ReleasedAmountYuan.Add(releasedAmount)
		order.Remark = reason
		order.Status = deriveOrderStatus(order)
		if order.Status == models.BillingOrderStatusReleased {
			now := time.Now()
			order.ClosedAt = &now
		}
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		account.AvailableBalanceYuan = account.AvailableBalanceYuan.Add(releasedAmount)
		account.ReservedBalanceYuan = account.ReservedBalanceYuan.Sub(releasedAmount)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    order.UserID,
			OrderNo:                   order.OrderNo,
			HoldNo:                    hold.HoldNo,
			HistoryID:                 order.HistoryID,
			TaskID:                    order.TaskID,
			EntryType:                 models.LedgerEntryTypeRelease,
			Scene:                     order.Scene,
			ActionAmountYuan:          releasedAmount,
			AvailableDeltaYuan:        releasedAmount,
			ReservedDeltaYuan:         releasedAmount.Neg(),
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			Remark:                    reason,
			CreatedAt:                 time.Now(),
		}
		return s.repo.CreateLedgerTx(ctx, tx, entry)
	})
	if err != nil {
		return nil, money.Zero(), err
	}

	return order, releasedAmount, nil
}

func (s *BillingService) PrepareFileTransferBilling(ctx context.Context, userID string, historyID, fileSizeBytes int64) (*models.BillingChargeOrder, *models.BillingHold, *models.BillingAccount, *models.BillingPricing, error) {
	var (
		order   *models.BillingChargeOrder
		hold    *models.BillingHold
		account *models.BillingAccount
		pricing *models.BillingPricing
	)

	err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		account, err = s.repo.GetOrCreateAccountTx(ctx, tx, userID)
		if err != nil {
			return err
		}

		order, err = s.repo.GetLatestDownloadOrderByHistoryIDForUpdate(ctx, tx, historyID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if order != nil && order.Status == models.BillingOrderStatusAwaitingShortfall {
			if _, err := s.resolveInitialDownloadShortfall(ctx, tx, order, account, "", ""); err != nil {
				return err
			}
		}

		now := time.Now()
		fundingSource := int32(models.BillingFundingSourceNewReserve)
		requiredReserve := money.Zero()

		if errors.Is(err, sql.ErrNoRows) || order == nil || !canUseInitialOrder(order) {
			pricing, err = s.repo.GetActivePricing(ctx)
			if err != nil {
				return err
			}
			requiredReserve, err = calculateAmountYuan(fileSizeBytes, pricing.EgressPriceYuanPerGB)
			if err != nil {
				return err
			}
			if account.AvailableBalanceYuan.Cmp(requiredReserve) < 0 {
				return ErrInsufficientBalance
			}

			account.AvailableBalanceYuan = account.AvailableBalanceYuan.Sub(requiredReserve)
			account.ReservedBalanceYuan = account.ReservedBalanceYuan.Add(requiredReserve)
			if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
				return err
			}

			order = &models.BillingChargeOrder{
				OrderNo:               newBillingID("ord"),
				UserID:                userID,
				HistoryID:             historyID,
				Scene:                 models.BillingSceneRedownload,
				Status:                models.BillingOrderStatusHeld,
				PricingVersion:        pricing.Version,
				EstimatedEgressBytes:  fileSizeBytes,
				EstimatedTrafficBytes: fileSizeBytes,
				HeldAmountYuan:        requiredReserve,
				Remark:                "redownload hold",
				CreatedAt:             now,
				UpdatedAt:             now,
			}
			if err := s.repo.CreateOrderTx(ctx, tx, order); err != nil {
				return err
			}

			entry := &models.BillingLedgerEntry{
				EntryNo:                   newBillingID("led"),
				AccountID:                 account.ID,
				UserID:                    userID,
				OrderNo:                   order.OrderNo,
				HistoryID:                 historyID,
				EntryType:                 models.LedgerEntryTypeHold,
				Scene:                     models.BillingSceneRedownload,
				ActionAmountYuan:          requiredReserve,
				AvailableDeltaYuan:        requiredReserve.Neg(),
				ReservedDeltaYuan:         requiredReserve,
				BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
				BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
				Remark:                    "hold redownload transfer",
				CreatedAt:                 now,
			}
			if err := s.repo.CreateLedgerTx(ctx, tx, entry); err != nil {
				return err
			}
		} else {
			pricing, err = s.repo.GetPricingByVersion(ctx, order.PricingVersion)
			if err != nil {
				return err
			}
			requiredReserve, err = calculateAmountYuan(fileSizeBytes, pricing.EgressPriceYuanPerGB)
			if err != nil {
				return err
			}

			remainingReserve := remainingOrderReserve(order)
			additionalReserve := requiredReserve.Sub(remainingReserve)
			if additionalReserve.Cmp(money.Zero()) > 0 {
				if account.AvailableBalanceYuan.Cmp(additionalReserve) < 0 {
					setOrderAwaitingShortfall(order, additionalReserve, "awaiting shortfall resolution: first transfer reserve")
					if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
						return err
					}
					return ErrInsufficientBalance
				}
				account.AvailableBalanceYuan = account.AvailableBalanceYuan.Sub(additionalReserve)
				account.ReservedBalanceYuan = account.ReservedBalanceYuan.Add(additionalReserve)
				order.HeldAmountYuan = order.HeldAmountYuan.Add(additionalReserve)
				fundingSource = models.BillingFundingSourceNewReserve

				if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
					return err
				}
				if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
					return err
				}

				entry := &models.BillingLedgerEntry{
					EntryNo:                   newBillingID("led"),
					AccountID:                 account.ID,
					UserID:                    userID,
					OrderNo:                   order.OrderNo,
					HistoryID:                 historyID,
					TaskID:                    order.TaskID,
					EntryType:                 models.LedgerEntryTypeHold,
					Scene:                     order.Scene,
					ActionAmountYuan:          additionalReserve,
					AvailableDeltaYuan:        additionalReserve.Neg(),
					ReservedDeltaYuan:         additionalReserve,
					BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
					BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
					Remark:                    "top up first transfer reserve",
					CreatedAt:                 now,
				}
				if err := s.repo.CreateLedgerTx(ctx, tx, entry); err != nil {
					return err
				}
			} else {
				fundingSource = models.BillingFundingSourceExistingReserve
			}
		}

		hold = &models.BillingHold{
			HoldNo:             newBillingID("hold"),
			OrderNo:            order.OrderNo,
			UserID:             userID,
			HistoryID:          historyID,
			TransferID:         newBillingID("trf"),
			HoldType:           models.BillingHoldTypeFileTransfer,
			FundingSource:      fundingSource,
			Status:             models.BillingHoldStatusHeld,
			AmountYuan:         requiredReserve,
			CapturedAmountYuan: money.Zero(),
			ReleasedAmountYuan: money.Zero(),
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		return s.repo.CreateHoldTx(ctx, tx, hold)
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return order, hold, account, pricing, nil
}

func (s *BillingService) CompleteFileTransferBilling(ctx context.Context, transferID string, actualEgressBytes int64) (*models.BillingChargeOrder, money.Decimal, error) {
	var (
		order          *models.BillingChargeOrder
		capturedAmount money.Decimal
	)

	err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		hold, err := s.repo.GetHoldByTransferIDForUpdate(ctx, tx, transferID)
		if err != nil {
			return err
		}
		order, err = s.repo.GetOrderByOrderNoForUpdate(ctx, tx, hold.OrderNo)
		if err != nil {
			return err
		}
		account, err := s.repo.GetOrCreateAccountTx(ctx, tx, order.UserID)
		if err != nil {
			return err
		}

		pricing, err := s.repo.GetPricingByVersion(ctx, order.PricingVersion)
		if err != nil {
			return err
		}
		capturedAmount, err = calculateAmountYuan(actualEgressBytes, pricing.EgressPriceYuanPerGB)
		if err != nil {
			return err
		}
		if hold.Status == models.BillingHoldStatusCaptured || (hold.CapturedAmountYuan.Cmp(money.Zero()) > 0 && order.ShortfallYuan.IsZero()) {
			capturedAmount = hold.CapturedAmountYuan
			return nil
		}
		if order.ActualEgressBytes > 0 {
			if order.ActualEgressBytes != actualEgressBytes {
				return fmt.Errorf("egress usage already recorded for transfer %s", transferID)
			}
			if hold.CapturedAmountYuan.Cmp(money.Zero()) > 0 && order.ShortfallYuan.IsZero() {
				capturedAmount = hold.CapturedAmountYuan
				return nil
			}
		}

		additionalReserve := money.Zero()
		if remaining := remainingOrderReserve(order); remaining.Cmp(capturedAmount) < 0 {
			additionalReserve = capturedAmount.Sub(remaining)
			if account.AvailableBalanceYuan.Cmp(additionalReserve) < 0 {
				if order.ActualEgressBytes == 0 {
					order.ActualEgressBytes += actualEgressBytes
					order.ActualTrafficBytes += actualEgressBytes
				}
				setOrderAwaitingShortfall(order, additionalReserve, "awaiting shortfall resolution: file transfer capture")
				if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
					return err
				}
				return ErrInsufficientBalance
			}
			account.AvailableBalanceYuan = account.AvailableBalanceYuan.Sub(additionalReserve)
			account.ReservedBalanceYuan = account.ReservedBalanceYuan.Add(additionalReserve)
			order.HeldAmountYuan = order.HeldAmountYuan.Add(additionalReserve)
		}

		holdDiff := capturedAmount.Sub(hold.AmountYuan)
		if holdDiff.Cmp(money.Zero()) > 0 {
			hold.AmountYuan = hold.AmountYuan.Add(holdDiff)
		}
		hold.CapturedAmountYuan = hold.CapturedAmountYuan.Add(capturedAmount)
		hold.Status = models.BillingHoldStatusCaptured
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		if order.ActualEgressBytes == 0 {
			order.ActualEgressBytes += actualEgressBytes
			order.ActualTrafficBytes += actualEgressBytes
		}
		order.ShortfallYuan = money.Zero()
		order.CapturedAmountYuan = order.CapturedAmountYuan.Add(capturedAmount)
		order.Status = deriveOrderStatus(order)
		now := time.Now()
		if order.Status == models.BillingOrderStatusCaptured {
			order.ClosedAt = &now
		}
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		account.ReservedBalanceYuan = account.ReservedBalanceYuan.Sub(capturedAmount)
		account.TotalSpentYuan = account.TotalSpentYuan.Add(capturedAmount)
		account.TotalTrafficBytes += actualEgressBytes
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		if additionalReserve.Cmp(money.Zero()) > 0 {
			holdEntry := &models.BillingLedgerEntry{
				EntryNo:                   newBillingID("led"),
				AccountID:                 account.ID,
				UserID:                    order.UserID,
				OrderNo:                   order.OrderNo,
				HoldNo:                    hold.HoldNo,
				HistoryID:                 order.HistoryID,
				TaskID:                    order.TaskID,
				TransferID:                transferID,
				EntryType:                 models.LedgerEntryTypeHold,
				Scene:                     order.Scene,
				ActionAmountYuan:          additionalReserve,
				AvailableDeltaYuan:        additionalReserve.Neg(),
				ReservedDeltaYuan:         additionalReserve,
				BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
				BalanceAfterReservedYuan:  account.ReservedBalanceYuan.Add(capturedAmount),
				Remark:                    "top up transfer reserve",
				CreatedAt:                 now,
			}
			if err := s.repo.CreateLedgerTx(ctx, tx, holdEntry); err != nil {
				return err
			}
		}

		usage := &models.TrafficUsageRecord{
			UsageNo:            newBillingID("use"),
			OrderNo:            order.OrderNo,
			UserID:             order.UserID,
			HistoryID:          order.HistoryID,
			TaskID:             order.TaskID,
			TransferID:         transferID,
			Direction:          models.TrafficDirectionEgress,
			TrafficBytes:       actualEgressBytes,
			UnitPriceYuanPerGB: pricing.EgressPriceYuanPerGB,
			AmountYuan:         capturedAmount,
			PricingVersion:     pricing.Version,
			SourceService:      "api-gateway",
			Status:             models.TrafficUsageStatusConfirmed,
			ConfirmedAt:        &now,
		}
		if err := s.repo.CreateUsageTx(ctx, tx, usage); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    order.UserID,
			OrderNo:                   order.OrderNo,
			HoldNo:                    hold.HoldNo,
			HistoryID:                 order.HistoryID,
			TaskID:                    order.TaskID,
			TransferID:                transferID,
			EntryType:                 models.LedgerEntryTypeCapture,
			Scene:                     order.Scene,
			ActionAmountYuan:          capturedAmount,
			AvailableDeltaYuan:        money.Zero(),
			ReservedDeltaYuan:         capturedAmount.Neg(),
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			Remark:                    "capture file transfer",
			CreatedAt:                 now,
		}
		if err := s.repo.CreateLedgerTx(ctx, tx, entry); err != nil {
			return err
		}

		releaseAmount := remainingHoldAmount(hold)
		if releaseAmount.Cmp(money.Zero()) > 0 {
			hold.ReleasedAmountYuan = hold.ReleasedAmountYuan.Add(releaseAmount)
			if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
				return err
			}
			order.ReleasedAmountYuan = order.ReleasedAmountYuan.Add(releaseAmount)
			order.Status = deriveOrderStatus(order)
			if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
				return err
			}
			account.AvailableBalanceYuan = account.AvailableBalanceYuan.Add(releaseAmount)
			account.ReservedBalanceYuan = account.ReservedBalanceYuan.Sub(releaseAmount)
			if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
				return err
			}
			releaseEntry := &models.BillingLedgerEntry{
				EntryNo:                   newBillingID("led"),
				AccountID:                 account.ID,
				UserID:                    order.UserID,
				OrderNo:                   order.OrderNo,
				HoldNo:                    hold.HoldNo,
				HistoryID:                 order.HistoryID,
				TaskID:                    order.TaskID,
				TransferID:                transferID,
				EntryType:                 models.LedgerEntryTypeRelease,
				Scene:                     order.Scene,
				ActionAmountYuan:          releaseAmount,
				AvailableDeltaYuan:        releaseAmount,
				ReservedDeltaYuan:         releaseAmount.Neg(),
				BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
				BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
				Remark:                    "release unused transfer reserve",
				CreatedAt:                 time.Now(),
			}
			if err := s.repo.CreateLedgerTx(ctx, tx, releaseEntry); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, money.Zero(), err
	}

	return order, capturedAmount, nil
}

func (s *BillingService) AbortFileTransferBilling(ctx context.Context, transferID, reason string) (*models.BillingChargeOrder, money.Decimal, error) {
	var (
		order          *models.BillingChargeOrder
		releasedAmount money.Decimal
	)

	err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		hold, err := s.repo.GetHoldByTransferIDForUpdate(ctx, tx, transferID)
		if err != nil {
			return err
		}
		order, err = s.repo.GetOrderByOrderNoForUpdate(ctx, tx, hold.OrderNo)
		if err != nil {
			return err
		}
		account, err := s.repo.GetOrCreateAccountTx(ctx, tx, order.UserID)
		if err != nil {
			return err
		}

		releasedAmount = remainingHoldAmount(hold)
		if releasedAmount.IsZero() {
			return nil
		}

		hold.ReleasedAmountYuan = hold.ReleasedAmountYuan.Add(releasedAmount)
		hold.Status = models.BillingHoldStatusReleased
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		order.ReleasedAmountYuan = order.ReleasedAmountYuan.Add(releasedAmount)
		order.Remark = reason
		order.Status = deriveOrderStatus(order)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		account.AvailableBalanceYuan = account.AvailableBalanceYuan.Add(releasedAmount)
		account.ReservedBalanceYuan = account.ReservedBalanceYuan.Sub(releasedAmount)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    order.UserID,
			OrderNo:                   order.OrderNo,
			HoldNo:                    hold.HoldNo,
			HistoryID:                 order.HistoryID,
			TaskID:                    order.TaskID,
			TransferID:                transferID,
			EntryType:                 models.LedgerEntryTypeRelease,
			Scene:                     order.Scene,
			ActionAmountYuan:          releasedAmount,
			AvailableDeltaYuan:        releasedAmount,
			ReservedDeltaYuan:         releasedAmount.Neg(),
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			Remark:                    reason,
			CreatedAt:                 time.Now(),
		}
		return s.repo.CreateLedgerTx(ctx, tx, entry)
	})
	if err != nil {
		return nil, money.Zero(), err
	}

	return order, releasedAmount, nil
}

func (s *BillingService) resolveInitialDownloadShortfall(ctx context.Context, tx *sql.Tx, order *models.BillingChargeOrder, account *models.BillingAccount, remark, operatorUserID string) (*models.BillingLedgerEntry, error) {
	if order == nil || order.Scene != models.BillingSceneDownload || order.ShortfallYuan.IsZero() {
		return nil, nil
	}
	if account.AvailableBalanceYuan.Cmp(order.ShortfallYuan) < 0 {
		setOrderAwaitingShortfall(order, order.ShortfallYuan, order.Remark)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return nil, err
		}
		return nil, ErrInsufficientBalance
	}

	now := time.Now()
	shortfallYuan := order.ShortfallYuan
	account.AvailableBalanceYuan = account.AvailableBalanceYuan.Sub(shortfallYuan)
	account.ReservedBalanceYuan = account.ReservedBalanceYuan.Add(shortfallYuan)
	order.HeldAmountYuan = order.HeldAmountYuan.Add(shortfallYuan)

	holdEntry := newReserveLedgerEntry(account, order, "", shortfallYuan, remarkOrDefault(remark, "resolve initial order shortfall"), operatorUserID, now)

	if order.ActualIngressBytes > 0 && order.CapturedAmountYuan.IsZero() {
		hold, err := s.repo.GetHoldByTaskIDForUpdate(ctx, tx, order.TaskID, models.BillingHoldTypeDownloadTotal)
		if err != nil {
			return nil, err
		}
		pricing, err := s.repo.GetPricingByVersion(ctx, order.PricingVersion)
		if err != nil {
			return nil, err
		}
		capturedAmount, err := calculateAmountYuan(order.ActualIngressBytes, pricing.IngressPriceYuanPerGB)
		if err != nil {
			return nil, err
		}

		hold.AmountYuan = hold.AmountYuan.Add(shortfallYuan)
		hold.CapturedAmountYuan = hold.CapturedAmountYuan.Add(capturedAmount)
		hold.Status = deriveHoldStatus(hold)
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return nil, err
		}

		order.ShortfallYuan = money.Zero()
		order.Remark = remarkOrDefault(remark, "shortfall resolved")
		order.CapturedAmountYuan = order.CapturedAmountYuan.Add(capturedAmount)
		order.Status = deriveOrderStatus(order)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return nil, err
		}

		account.ReservedBalanceYuan = account.ReservedBalanceYuan.Sub(capturedAmount)
		account.TotalSpentYuan = account.TotalSpentYuan.Add(capturedAmount)
		account.TotalTrafficBytes += order.ActualIngressBytes
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return nil, err
		}

		holdEntry.HoldNo = hold.HoldNo
		holdEntry.BalanceAfterReservedYuan = account.ReservedBalanceYuan.Add(capturedAmount)
		if err := s.repo.CreateLedgerTx(ctx, tx, holdEntry); err != nil {
			return nil, err
		}

		usage := &models.TrafficUsageRecord{
			UsageNo:            newBillingID("use"),
			OrderNo:            order.OrderNo,
			UserID:             order.UserID,
			HistoryID:          order.HistoryID,
			TaskID:             order.TaskID,
			Direction:          models.TrafficDirectionIngress,
			TrafficBytes:       order.ActualIngressBytes,
			UnitPriceYuanPerGB: pricing.IngressPriceYuanPerGB,
			AmountYuan:         capturedAmount,
			PricingVersion:     pricing.Version,
			SourceService:      "media-service",
			Status:             models.TrafficUsageStatusConfirmed,
			ConfirmedAt:        &now,
		}
		if err := s.repo.CreateUsageTx(ctx, tx, usage); err != nil {
			return nil, err
		}

		captureEntry := &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    order.UserID,
			OrderNo:                   order.OrderNo,
			HoldNo:                    hold.HoldNo,
			HistoryID:                 order.HistoryID,
			TaskID:                    order.TaskID,
			EntryType:                 models.LedgerEntryTypeCapture,
			Scene:                     order.Scene,
			ActionAmountYuan:          capturedAmount,
			AvailableDeltaYuan:        money.Zero(),
			ReservedDeltaYuan:         capturedAmount.Neg(),
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			Remark:                    "capture ingress usage after shortfall resolution",
			CreatedAt:                 now,
		}
		return holdEntry, s.repo.CreateLedgerTx(ctx, tx, captureEntry)
	}

	order.ShortfallYuan = money.Zero()
	order.Remark = remarkOrDefault(remark, "shortfall resolved")
	order.Status = deriveOrderStatus(order)
	if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
		return nil, err
	}
	return holdEntry, s.repo.CreateLedgerTx(ctx, tx, holdEntry)
}

func (s *BillingService) resolveTransferShortfall(ctx context.Context, tx *sql.Tx, order *models.BillingChargeOrder, account *models.BillingAccount, remark, operatorUserID string) (*models.BillingLedgerEntry, error) {
	if order == nil || order.ShortfallYuan.IsZero() || order.ActualEgressBytes <= 0 {
		return nil, nil
	}
	if account.AvailableBalanceYuan.Cmp(order.ShortfallYuan) < 0 {
		setOrderAwaitingShortfall(order, order.ShortfallYuan, order.Remark)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return nil, err
		}
		return nil, ErrInsufficientBalance
	}

	hold, err := s.repo.GetLatestPendingTransferHoldByOrderNoForUpdate(ctx, tx, order.OrderNo)
	if err != nil {
		return nil, err
	}
	pricing, err := s.repo.GetPricingByVersion(ctx, order.PricingVersion)
	if err != nil {
		return nil, err
	}
	capturedAmount, err := calculateAmountYuan(order.ActualEgressBytes, pricing.EgressPriceYuanPerGB)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	shortfallYuan := order.ShortfallYuan
	account.AvailableBalanceYuan = account.AvailableBalanceYuan.Sub(shortfallYuan)
	account.ReservedBalanceYuan = account.ReservedBalanceYuan.Add(shortfallYuan)
	order.HeldAmountYuan = order.HeldAmountYuan.Add(shortfallYuan)

	holdEntry := newReserveLedgerEntry(account, order, hold.HoldNo, shortfallYuan, remarkOrDefault(remark, "resolve transfer shortfall"), operatorUserID, now)

	holdDiff := capturedAmount.Sub(hold.AmountYuan)
	if holdDiff.Cmp(money.Zero()) > 0 {
		hold.AmountYuan = hold.AmountYuan.Add(holdDiff)
	}
	hold.CapturedAmountYuan = hold.CapturedAmountYuan.Add(capturedAmount)
	hold.Status = models.BillingHoldStatusCaptured
	if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
		return nil, err
	}

	order.ShortfallYuan = money.Zero()
	order.Remark = remarkOrDefault(remark, "shortfall resolved")
	order.CapturedAmountYuan = order.CapturedAmountYuan.Add(capturedAmount)
	order.Status = deriveOrderStatus(order)
	if order.Status == models.BillingOrderStatusCaptured {
		order.ClosedAt = &now
	}
	if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
		return nil, err
	}

	account.ReservedBalanceYuan = account.ReservedBalanceYuan.Sub(capturedAmount)
	account.TotalSpentYuan = account.TotalSpentYuan.Add(capturedAmount)
	account.TotalTrafficBytes += order.ActualEgressBytes
	if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
		return nil, err
	}

	holdEntry.BalanceAfterReservedYuan = account.ReservedBalanceYuan.Add(capturedAmount)
	if err := s.repo.CreateLedgerTx(ctx, tx, holdEntry); err != nil {
		return nil, err
	}

	usage := &models.TrafficUsageRecord{
		UsageNo:            newBillingID("use"),
		OrderNo:            order.OrderNo,
		UserID:             order.UserID,
		HistoryID:          order.HistoryID,
		TaskID:             order.TaskID,
		TransferID:         hold.TransferID,
		Direction:          models.TrafficDirectionEgress,
		TrafficBytes:       order.ActualEgressBytes,
		UnitPriceYuanPerGB: pricing.EgressPriceYuanPerGB,
		AmountYuan:         capturedAmount,
		PricingVersion:     pricing.Version,
		SourceService:      "api-gateway",
		Status:             models.TrafficUsageStatusConfirmed,
		ConfirmedAt:        &now,
	}
	if err := s.repo.CreateUsageTx(ctx, tx, usage); err != nil {
		return nil, err
	}

	captureEntry := &models.BillingLedgerEntry{
		EntryNo:                   newBillingID("led"),
		AccountID:                 account.ID,
		UserID:                    order.UserID,
		OrderNo:                   order.OrderNo,
		HoldNo:                    hold.HoldNo,
		HistoryID:                 order.HistoryID,
		TaskID:                    order.TaskID,
		TransferID:                hold.TransferID,
		EntryType:                 models.LedgerEntryTypeCapture,
		Scene:                     order.Scene,
		ActionAmountYuan:          capturedAmount,
		AvailableDeltaYuan:        money.Zero(),
		ReservedDeltaYuan:         capturedAmount.Neg(),
		BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
		BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
		Remark:                    "capture file transfer after shortfall resolution",
		CreatedAt:                 now,
	}
	if err := s.repo.CreateLedgerTx(ctx, tx, captureEntry); err != nil {
		return nil, err
	}

	releaseAmount := remainingHoldAmount(hold)
	if releaseAmount.Cmp(money.Zero()) > 0 {
		hold.ReleasedAmountYuan = hold.ReleasedAmountYuan.Add(releaseAmount)
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return nil, err
		}
		order.ReleasedAmountYuan = order.ReleasedAmountYuan.Add(releaseAmount)
		order.Status = deriveOrderStatus(order)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return nil, err
		}
		account.AvailableBalanceYuan = account.AvailableBalanceYuan.Add(releaseAmount)
		account.ReservedBalanceYuan = account.ReservedBalanceYuan.Sub(releaseAmount)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return nil, err
		}
		releaseEntry := &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    order.UserID,
			OrderNo:                   order.OrderNo,
			HoldNo:                    hold.HoldNo,
			HistoryID:                 order.HistoryID,
			TaskID:                    order.TaskID,
			TransferID:                hold.TransferID,
			EntryType:                 models.LedgerEntryTypeRelease,
			Scene:                     order.Scene,
			ActionAmountYuan:          releaseAmount,
			AvailableDeltaYuan:        releaseAmount,
			ReservedDeltaYuan:         releaseAmount.Neg(),
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			Remark:                    "release unused reserve after shortfall resolution",
			CreatedAt:                 now,
		}
		if err := s.repo.CreateLedgerTx(ctx, tx, releaseEntry); err != nil {
			return nil, err
		}
	}

	return holdEntry, nil
}

func (s *BillingService) ListBillingAccounts(ctx context.Context, filter models.BillingAccountFilter) (*models.BillingAccountResult, error) {
	if len(filter.UserIDs) > 0 {
		for _, userID := range filter.UserIDs {
			if _, err := s.repo.GetOrCreateAccount(ctx, userID); err != nil {
				return nil, err
			}
		}
	}
	return s.repo.ListAccounts(ctx, filter)
}

func (s *BillingService) GetBillingAccountDetail(ctx context.Context, userID string) (*models.BillingAccount, error) {
	return s.repo.GetOrCreateAccount(ctx, userID)
}

func (s *BillingService) AdjustBillingBalance(ctx context.Context, userID, operationID string, amountYuan money.Decimal, remark, operatorUserID string) (*models.BillingAccount, *models.BillingLedgerEntry, error) {
	if operationID == "" {
		return nil, nil, fmt.Errorf("operation id is required")
	}

	existing, err := s.repo.GetLedgerByOperationID(ctx, operationID)
	if err == nil {
		account, accErr := s.repo.GetOrCreateAccount(ctx, userID)
		return account, existing, accErr
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	var (
		account *models.BillingAccount
		entry   *models.BillingLedgerEntry
	)
	err = s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		account, err = s.repo.GetOrCreateAccountTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		if amountYuan.Cmp(money.Zero()) < 0 && account.AvailableBalanceYuan.Cmp(amountYuan.Neg()) < 0 {
			return ErrInsufficientBalance
		}

		account.AvailableBalanceYuan = account.AvailableBalanceYuan.Add(amountYuan)
		if amountYuan.Cmp(money.Zero()) > 0 {
			account.TotalRechargedYuan = account.TotalRechargedYuan.Add(amountYuan)
		}
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		entryType := int32(models.LedgerEntryTypeManualAdjustment)
		if amountYuan.Cmp(money.Zero()) > 0 {
			entryType = models.LedgerEntryTypeManualTopup
		}
		entry = &models.BillingLedgerEntry{
			EntryNo:                   newBillingID("led"),
			AccountID:                 account.ID,
			UserID:                    userID,
			OperationID:               operationID,
			EntryType:                 entryType,
			Scene:                     models.BillingSceneAdmin,
			ActionAmountYuan:          abs64(amountYuan),
			AvailableDeltaYuan:        amountYuan,
			ReservedDeltaYuan:         money.Zero(),
			BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
			BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
			OperatorUserID:            operatorUserID,
			Remark:                    remark,
			CreatedAt:                 time.Now(),
		}
		return s.repo.CreateLedgerTx(ctx, tx, entry)
	})
	if err != nil {
		return nil, nil, err
	}

	return account, entry, nil
}

func (s *BillingService) ListBillingShortfalls(ctx context.Context, filter models.BillingShortfallFilter) (*models.BillingShortfallResult, error) {
	return s.repo.ListShortfallOrders(ctx, filter)
}

func (s *BillingService) ReconcileBillingShortfall(ctx context.Context, orderNo, remark, operatorUserID string) (*models.BillingChargeOrder, *models.BillingAccount, *models.BillingLedgerEntry, error) {
	var (
		order   *models.BillingChargeOrder
		account *models.BillingAccount
		entry   *models.BillingLedgerEntry
		err     error
	)

	err = s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		var err error
		order, err = s.repo.GetOrderByOrderNoForUpdate(ctx, tx, orderNo)
		if err != nil {
			return err
		}
		account, err = s.repo.GetOrCreateAccountTx(ctx, tx, order.UserID)
		if err != nil {
			return err
		}

		if order.ShortfallYuan.IsZero() || order.Status != models.BillingOrderStatusAwaitingShortfall {
			return nil
		}

		switch {
		case order.Scene == models.BillingSceneDownload && order.ActualIngressBytes > 0 && order.CapturedAmountYuan.IsZero():
			entry, err = s.resolveInitialDownloadShortfall(ctx, tx, order, account, remark, operatorUserID)
			return err
		case order.ActualEgressBytes > 0:
			entry, err = s.resolveTransferShortfall(ctx, tx, order, account, remark, operatorUserID)
			return err
		default:
			entry, err = s.resolveInitialDownloadShortfall(ctx, tx, order, account, remark, operatorUserID)
			return err
		}
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return order, account, entry, nil
}

func (s *BillingService) ListBillingLedger(ctx context.Context, filter models.BillingLedgerFilter) (*models.BillingLedgerResult, error) {
	return s.repo.ListLedger(ctx, filter)
}

func (s *BillingService) ListTrafficUsageRecords(ctx context.Context, filter models.TrafficUsageFilter) (*models.TrafficUsageResult, error) {
	return s.repo.ListUsageRecords(ctx, filter)
}

func (s *BillingService) GetBillingPricing(ctx context.Context) (*models.BillingPricing, error) {
	return s.repo.GetActivePricing(ctx)
}

func (s *BillingService) UpdateBillingPricing(ctx context.Context, ingressPrice, egressPrice string, remark, operatorUserID string) (*models.BillingPricing, error) {
	var (
		pricing *models.BillingPricing
		err     error
	)
	parsedIngressPrice, err := money.Parse(ingressPrice)
	if err != nil {
		return nil, fmt.Errorf("parse ingress price: %w", err)
	}
	parsedEgressPrice, err := money.Parse(egressPrice)
	if err != nil {
		return nil, fmt.Errorf("parse egress price: %w", err)
	}
	err = s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		latestVersion, err := s.repo.GetLatestPricingVersionTx(ctx, tx)
		if err != nil {
			return err
		}
		if err := s.repo.DisableAllPricingTx(ctx, tx); err != nil {
			return err
		}

		now := time.Now()
		pricing = &models.BillingPricing{
			Version:               latestVersion + 1,
			IngressPriceYuanPerGB: parsedIngressPrice,
			EgressPriceYuanPerGB:  parsedEgressPrice,
			Enabled:               true,
			Remark:                remark,
			UpdatedByUserID:       operatorUserID,
			EffectiveAt:           now,
			CreatedAt:             now,
		}
		return s.repo.CreatePricingTx(ctx, tx, pricing)
	})
	if err != nil {
		return nil, err
	}
	return pricing, nil
}

func canUseInitialOrder(order *models.BillingChargeOrder) bool {
	if order == nil {
		return false
	}
	if order.Scene != models.BillingSceneDownload {
		return false
	}
	if order.Status == models.BillingOrderStatusReleased || order.Status == models.BillingOrderStatusAwaitingShortfall {
		return false
	}
	return order.ActualEgressBytes == 0
}

func deriveOrderStatus(order *models.BillingChargeOrder) int32 {
	remaining := remainingOrderReserve(order)
	switch {
	case order.ShortfallYuan.Cmp(money.Zero()) > 0:
		return models.BillingOrderStatusAwaitingShortfall
	case order.CapturedAmountYuan.IsZero() && order.ReleasedAmountYuan.IsZero():
		return models.BillingOrderStatusHeld
	case order.CapturedAmountYuan.IsZero() && remaining.IsZero():
		return models.BillingOrderStatusReleased
	case order.CapturedAmountYuan.Cmp(money.Zero()) > 0 && order.ReleasedAmountYuan.Cmp(money.Zero()) > 0:
		return models.BillingOrderStatusPartialCaptured
	case order.CapturedAmountYuan.Cmp(money.Zero()) > 0 && remaining.Cmp(money.Zero()) > 0:
		return models.BillingOrderStatusPartialCaptured
	case order.CapturedAmountYuan.Cmp(money.Zero()) > 0 && remaining.IsZero():
		return models.BillingOrderStatusCaptured
	default:
		return models.BillingOrderStatusPartialCaptured
	}
}

func deriveHoldStatus(hold *models.BillingHold) int32 {
	remaining := remainingHoldAmount(hold)
	switch {
	case hold.CapturedAmountYuan.IsZero() && hold.ReleasedAmountYuan.IsZero():
		return models.BillingHoldStatusHeld
	case hold.CapturedAmountYuan.Cmp(money.Zero()) > 0 && remaining.Cmp(money.Zero()) > 0:
		return models.BillingHoldStatusPartialCaptured
	case hold.CapturedAmountYuan.Cmp(money.Zero()) > 0 && remaining.IsZero():
		return models.BillingHoldStatusCaptured
	case hold.ReleasedAmountYuan.Cmp(money.Zero()) > 0 && remaining.IsZero():
		return models.BillingHoldStatusReleased
	default:
		return models.BillingHoldStatusHeld
	}
}

func setOrderAwaitingShortfall(order *models.BillingChargeOrder, shortfallYuan money.Decimal, remark string) {
	if shortfallYuan.Cmp(money.Zero()) < 0 {
		shortfallYuan = money.Zero()
	}
	order.ShortfallYuan = shortfallYuan
	if remark != "" {
		order.Remark = remark
	}
	order.Status = deriveOrderStatus(order)
}

func newReserveLedgerEntry(account *models.BillingAccount, order *models.BillingChargeOrder, holdNo string, amountYuan money.Decimal, remark, operatorUserID string, createdAt time.Time) *models.BillingLedgerEntry {
	return &models.BillingLedgerEntry{
		EntryNo:                   newBillingID("led"),
		AccountID:                 account.ID,
		UserID:                    order.UserID,
		OrderNo:                   order.OrderNo,
		HoldNo:                    holdNo,
		HistoryID:                 order.HistoryID,
		TaskID:                    order.TaskID,
		EntryType:                 models.LedgerEntryTypeHold,
		Scene:                     order.Scene,
		ActionAmountYuan:          amountYuan,
		AvailableDeltaYuan:        amountYuan.Neg(),
		ReservedDeltaYuan:         amountYuan,
		BalanceAfterAvailableYuan: account.AvailableBalanceYuan,
		BalanceAfterReservedYuan:  account.ReservedBalanceYuan,
		OperatorUserID:            operatorUserID,
		Remark:                    remark,
		CreatedAt:                 createdAt,
	}
}

func remarkOrDefault(remark, fallback string) string {
	if remark != "" {
		return remark
	}
	return fallback
}

func remainingOrderReserve(order *models.BillingChargeOrder) money.Decimal {
	remaining := order.HeldAmountYuan.Sub(order.CapturedAmountYuan).Sub(order.ReleasedAmountYuan)
	if remaining.Cmp(money.Zero()) < 0 {
		return money.Zero()
	}
	return remaining
}

func remainingHoldAmount(hold *models.BillingHold) money.Decimal {
	remaining := hold.AmountYuan.Sub(hold.CapturedAmountYuan).Sub(hold.ReleasedAmountYuan)
	if remaining.Cmp(money.Zero()) < 0 {
		return money.Zero()
	}
	return remaining
}

func calculateAmountYuan(bytes int64, pricePerGB money.Decimal) (money.Decimal, error) {
	if bytes < 0 {
		return money.Zero(), fmt.Errorf("bytes must be non-negative")
	}

	billableMB := bytes / mbBytes
	if bytes%mbBytes != 0 {
		billableMB++
	}
	if billableMB < minBillableMB {
		billableMB = minBillableMB
	}

	amount := pricePerGB.Mul(money.FromInt64(billableMB)).DivInt64(gbMB)
	return amount.Ceil(2), nil
}

func newBillingID(prefix string) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%d_%x", prefix, time.Now().UnixNano(), buf)
}

func abs64(v money.Decimal) money.Decimal {
	return v.Abs()
}
