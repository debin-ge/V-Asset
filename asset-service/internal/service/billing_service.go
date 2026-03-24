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

	amountMinor := settings.AmountYuan.Mul(money.FromInt64(100))
	if amountMinor.Cmp(money.Zero()) <= 0 {
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

		account.AvailableBalanceFen = account.AvailableBalanceFen.Add(amountMinor)
		account.TotalRechargedFen = account.TotalRechargedFen.Add(amountMinor)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		now := time.Now()
		entry = &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   userID,
			OperationID:              operationID,
			EntryType:                models.LedgerEntryTypeManualTopup,
			Scene:                    models.BillingSceneOnboarding,
			ActionAmountFen:          abs64(amountMinor),
			AvailableDeltaFen:        amountMinor,
			ReservedDeltaFen:         money.Zero(),
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			Remark:                   models.WelcomeCreditReasonCode,
			CreatedAt:                now,
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
	isEstimated := fileBytes <= 0
	reason := ""
	if fileBytes <= 0 {
		reason = "unknown_filesize"
	}

	ingressCost, err := calculateAmountFen(fileBytes, pricing.IngressPriceFenPerGiB)
	if err != nil {
		return nil, nil, err
	}
	egressCost, err := calculateAmountFen(fileBytes, pricing.EgressPriceFenPerGiB)
	if err != nil {
		return nil, nil, err
	}

	return &models.BillingEstimate{
		EstimatedIngressBytes: fileBytes,
		EstimatedEgressBytes:  fileBytes,
		EstimatedTrafficBytes: fileBytes * 2,
		EstimatedCostFen:      ingressCost.Add(egressCost),
		PricingVersion:        pricing.Version,
		IsEstimated:           isEstimated,
		EstimateReason:        reason,
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
		if account.AvailableBalanceFen.Cmp(estimate.EstimatedCostFen) < 0 {
			return ErrInsufficientBalance
		}

		account.AvailableBalanceFen = account.AvailableBalanceFen.Sub(estimate.EstimatedCostFen)
		account.ReservedBalanceFen = account.ReservedBalanceFen.Add(estimate.EstimatedCostFen)
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
			HeldAmountFen:         estimate.EstimatedCostFen,
			Remark:                "initial download hold",
			CreatedAt:             now,
			UpdatedAt:             now,
		}
		if err := s.repo.CreateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		hold = &models.BillingHold{
			HoldNo:            newBillingID("hold"),
			OrderNo:           order.OrderNo,
			UserID:            userID,
			HistoryID:         historyID,
			TaskID:            taskID,
			HoldType:          models.BillingHoldTypeDownloadTotal,
			FundingSource:     models.BillingFundingSourceNewReserve,
			Status:            models.BillingHoldStatusHeld,
			AmountFen:         estimate.EstimatedCostFen,
			CapturedAmountFen: money.Zero(),
			ReleasedAmountFen: money.Zero(),
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := s.repo.CreateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   userID,
			OrderNo:                  order.OrderNo,
			HoldNo:                   hold.HoldNo,
			HistoryID:                historyID,
			TaskID:                   taskID,
			EntryType:                models.LedgerEntryTypeHold,
			Scene:                    models.BillingSceneDownload,
			ActionAmountFen:          estimate.EstimatedCostFen,
			AvailableDeltaFen:        estimate.EstimatedCostFen.Neg(),
			ReservedDeltaFen:         estimate.EstimatedCostFen,
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			Remark:                   "hold initial download",
			CreatedAt:                now,
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
		capturedAmount, err = calculateAmountFen(actualIngressBytes, pricing.IngressPriceFenPerGiB)
		if err != nil {
			return err
		}
		if order.ActualIngressBytes > 0 {
			if order.ActualIngressBytes != actualIngressBytes {
				return fmt.Errorf("ingress usage already recorded for task %s", taskID)
			}
			if order.CapturedAmountFen.Cmp(money.Zero()) > 0 && order.ShortfallFen.IsZero() {
				capturedAmount = order.CapturedAmountFen
				return nil
			}
		}

		additionalReserve := money.Zero()
		if remaining := remainingOrderReserve(order); remaining.Cmp(capturedAmount) < 0 {
			additionalReserve = capturedAmount.Sub(remaining)
			if account.AvailableBalanceFen.Cmp(additionalReserve) < 0 {
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
			account.AvailableBalanceFen = account.AvailableBalanceFen.Sub(additionalReserve)
			account.ReservedBalanceFen = account.ReservedBalanceFen.Add(additionalReserve)
			order.HeldAmountFen = order.HeldAmountFen.Add(additionalReserve)
			hold.AmountFen = hold.AmountFen.Add(additionalReserve)
		}

		if order.ActualIngressBytes == 0 {
			order.ActualIngressBytes = actualIngressBytes
			order.ActualTrafficBytes += actualIngressBytes
		}
		order.ShortfallFen = money.Zero()
		order.CapturedAmountFen = order.CapturedAmountFen.Add(capturedAmount)
		order.Status = deriveOrderStatus(order)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		hold.CapturedAmountFen = hold.CapturedAmountFen.Add(capturedAmount)
		hold.Status = deriveHoldStatus(hold)
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		account.ReservedBalanceFen = account.ReservedBalanceFen.Sub(capturedAmount)
		account.TotalSpentFen = account.TotalSpentFen.Add(capturedAmount)
		account.TotalTrafficBytes += actualIngressBytes
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		now := time.Now()
		if additionalReserve.Cmp(money.Zero()) > 0 {
			holdEntry := &models.BillingLedgerEntry{
				EntryNo:                  newBillingID("led"),
				AccountID:                account.ID,
				UserID:                   order.UserID,
				OrderNo:                  order.OrderNo,
				HoldNo:                   hold.HoldNo,
				HistoryID:                order.HistoryID,
				TaskID:                   order.TaskID,
				EntryType:                models.LedgerEntryTypeHold,
				Scene:                    order.Scene,
				ActionAmountFen:          additionalReserve,
				AvailableDeltaFen:        additionalReserve.Neg(),
				ReservedDeltaFen:         additionalReserve,
				BalanceAfterAvailableFen: account.AvailableBalanceFen,
				BalanceAfterReservedFen:  account.ReservedBalanceFen.Add(capturedAmount),
				Remark:                   "top up ingress reserve",
				CreatedAt:                now,
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
			UnitPriceFenPerGiB: pricing.IngressPriceFenPerGiB,
			AmountFen:          capturedAmount,
			PricingVersion:     pricing.Version,
			SourceService:      "media-service",
			Status:             models.TrafficUsageStatusConfirmed,
			ConfirmedAt:        &now,
		}
		if err := s.repo.CreateUsageTx(ctx, tx, usage); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   order.UserID,
			OrderNo:                  order.OrderNo,
			HoldNo:                   hold.HoldNo,
			HistoryID:                order.HistoryID,
			TaskID:                   order.TaskID,
			EntryType:                models.LedgerEntryTypeCapture,
			Scene:                    order.Scene,
			ActionAmountFen:          capturedAmount,
			AvailableDeltaFen:        money.Zero(),
			ReservedDeltaFen:         capturedAmount.Neg(),
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			Remark:                   "capture ingress usage",
			CreatedAt:                now,
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

		hold.ReleasedAmountFen = hold.ReleasedAmountFen.Add(releasedAmount)
		hold.Status = models.BillingHoldStatusReleased
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		order.ReleasedAmountFen = order.ReleasedAmountFen.Add(releasedAmount)
		order.Remark = reason
		order.Status = deriveOrderStatus(order)
		if order.Status == models.BillingOrderStatusReleased {
			now := time.Now()
			order.ClosedAt = &now
		}
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		account.AvailableBalanceFen = account.AvailableBalanceFen.Add(releasedAmount)
		account.ReservedBalanceFen = account.ReservedBalanceFen.Sub(releasedAmount)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   order.UserID,
			OrderNo:                  order.OrderNo,
			HoldNo:                   hold.HoldNo,
			HistoryID:                order.HistoryID,
			TaskID:                   order.TaskID,
			EntryType:                models.LedgerEntryTypeRelease,
			Scene:                    order.Scene,
			ActionAmountFen:          releasedAmount,
			AvailableDeltaFen:        releasedAmount,
			ReservedDeltaFen:         releasedAmount.Neg(),
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			Remark:                   reason,
			CreatedAt:                time.Now(),
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
			requiredReserve, err = calculateAmountFen(fileSizeBytes, pricing.EgressPriceFenPerGiB)
			if err != nil {
				return err
			}
			if account.AvailableBalanceFen.Cmp(requiredReserve) < 0 {
				return ErrInsufficientBalance
			}

			account.AvailableBalanceFen = account.AvailableBalanceFen.Sub(requiredReserve)
			account.ReservedBalanceFen = account.ReservedBalanceFen.Add(requiredReserve)
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
				HeldAmountFen:         requiredReserve,
				Remark:                "redownload hold",
				CreatedAt:             now,
				UpdatedAt:             now,
			}
			if err := s.repo.CreateOrderTx(ctx, tx, order); err != nil {
				return err
			}

			entry := &models.BillingLedgerEntry{
				EntryNo:                  newBillingID("led"),
				AccountID:                account.ID,
				UserID:                   userID,
				OrderNo:                  order.OrderNo,
				HistoryID:                historyID,
				EntryType:                models.LedgerEntryTypeHold,
				Scene:                    models.BillingSceneRedownload,
				ActionAmountFen:          requiredReserve,
				AvailableDeltaFen:        requiredReserve.Neg(),
				ReservedDeltaFen:         requiredReserve,
				BalanceAfterAvailableFen: account.AvailableBalanceFen,
				BalanceAfterReservedFen:  account.ReservedBalanceFen,
				Remark:                   "hold redownload transfer",
				CreatedAt:                now,
			}
			if err := s.repo.CreateLedgerTx(ctx, tx, entry); err != nil {
				return err
			}
		} else {
			pricing, err = s.repo.GetPricingByVersion(ctx, order.PricingVersion)
			if err != nil {
				return err
			}
			requiredReserve, err = calculateAmountFen(fileSizeBytes, pricing.EgressPriceFenPerGiB)
			if err != nil {
				return err
			}

			remainingReserve := remainingOrderReserve(order)
			additionalReserve := requiredReserve.Sub(remainingReserve)
			if additionalReserve.Cmp(money.Zero()) > 0 {
				if account.AvailableBalanceFen.Cmp(additionalReserve) < 0 {
					setOrderAwaitingShortfall(order, additionalReserve, "awaiting shortfall resolution: first transfer reserve")
					if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
						return err
					}
					return ErrInsufficientBalance
				}
				account.AvailableBalanceFen = account.AvailableBalanceFen.Sub(additionalReserve)
				account.ReservedBalanceFen = account.ReservedBalanceFen.Add(additionalReserve)
				order.HeldAmountFen = order.HeldAmountFen.Add(additionalReserve)
				fundingSource = models.BillingFundingSourceNewReserve

				if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
					return err
				}
				if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
					return err
				}

				entry := &models.BillingLedgerEntry{
					EntryNo:                  newBillingID("led"),
					AccountID:                account.ID,
					UserID:                   userID,
					OrderNo:                  order.OrderNo,
					HistoryID:                historyID,
					TaskID:                   order.TaskID,
					EntryType:                models.LedgerEntryTypeHold,
					Scene:                    order.Scene,
					ActionAmountFen:          additionalReserve,
					AvailableDeltaFen:        additionalReserve.Neg(),
					ReservedDeltaFen:         additionalReserve,
					BalanceAfterAvailableFen: account.AvailableBalanceFen,
					BalanceAfterReservedFen:  account.ReservedBalanceFen,
					Remark:                   "top up first transfer reserve",
					CreatedAt:                now,
				}
				if err := s.repo.CreateLedgerTx(ctx, tx, entry); err != nil {
					return err
				}
			} else {
				fundingSource = models.BillingFundingSourceExistingReserve
			}
		}

		hold = &models.BillingHold{
			HoldNo:            newBillingID("hold"),
			OrderNo:           order.OrderNo,
			UserID:            userID,
			HistoryID:         historyID,
			TransferID:        newBillingID("trf"),
			HoldType:          models.BillingHoldTypeFileTransfer,
			FundingSource:     fundingSource,
			Status:            models.BillingHoldStatusHeld,
			AmountFen:         requiredReserve,
			CapturedAmountFen: money.Zero(),
			ReleasedAmountFen: money.Zero(),
			CreatedAt:         now,
			UpdatedAt:         now,
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
		capturedAmount, err = calculateAmountFen(actualEgressBytes, pricing.EgressPriceFenPerGiB)
		if err != nil {
			return err
		}
		if hold.Status == models.BillingHoldStatusCaptured || (hold.CapturedAmountFen.Cmp(money.Zero()) > 0 && order.ShortfallFen.IsZero()) {
			capturedAmount = hold.CapturedAmountFen
			return nil
		}
		if order.ActualEgressBytes > 0 {
			if order.ActualEgressBytes != actualEgressBytes {
				return fmt.Errorf("egress usage already recorded for transfer %s", transferID)
			}
			if hold.CapturedAmountFen.Cmp(money.Zero()) > 0 && order.ShortfallFen.IsZero() {
				capturedAmount = hold.CapturedAmountFen
				return nil
			}
		}

		additionalReserve := money.Zero()
		if remaining := remainingOrderReserve(order); remaining.Cmp(capturedAmount) < 0 {
			additionalReserve = capturedAmount.Sub(remaining)
			if account.AvailableBalanceFen.Cmp(additionalReserve) < 0 {
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
			account.AvailableBalanceFen = account.AvailableBalanceFen.Sub(additionalReserve)
			account.ReservedBalanceFen = account.ReservedBalanceFen.Add(additionalReserve)
			order.HeldAmountFen = order.HeldAmountFen.Add(additionalReserve)
		}

		holdDiff := capturedAmount.Sub(hold.AmountFen)
		if holdDiff.Cmp(money.Zero()) > 0 {
			hold.AmountFen = hold.AmountFen.Add(holdDiff)
		}
		hold.CapturedAmountFen = hold.CapturedAmountFen.Add(capturedAmount)
		hold.Status = models.BillingHoldStatusCaptured
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		if order.ActualEgressBytes == 0 {
			order.ActualEgressBytes += actualEgressBytes
			order.ActualTrafficBytes += actualEgressBytes
		}
		order.ShortfallFen = money.Zero()
		order.CapturedAmountFen = order.CapturedAmountFen.Add(capturedAmount)
		order.Status = deriveOrderStatus(order)
		now := time.Now()
		if order.Status == models.BillingOrderStatusCaptured {
			order.ClosedAt = &now
		}
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		account.ReservedBalanceFen = account.ReservedBalanceFen.Sub(capturedAmount)
		account.TotalSpentFen = account.TotalSpentFen.Add(capturedAmount)
		account.TotalTrafficBytes += actualEgressBytes
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		if additionalReserve.Cmp(money.Zero()) > 0 {
			holdEntry := &models.BillingLedgerEntry{
				EntryNo:                  newBillingID("led"),
				AccountID:                account.ID,
				UserID:                   order.UserID,
				OrderNo:                  order.OrderNo,
				HoldNo:                   hold.HoldNo,
				HistoryID:                order.HistoryID,
				TaskID:                   order.TaskID,
				TransferID:               transferID,
				EntryType:                models.LedgerEntryTypeHold,
				Scene:                    order.Scene,
				ActionAmountFen:          additionalReserve,
				AvailableDeltaFen:        additionalReserve.Neg(),
				ReservedDeltaFen:         additionalReserve,
				BalanceAfterAvailableFen: account.AvailableBalanceFen,
				BalanceAfterReservedFen:  account.ReservedBalanceFen.Add(capturedAmount),
				Remark:                   "top up transfer reserve",
				CreatedAt:                now,
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
			UnitPriceFenPerGiB: pricing.EgressPriceFenPerGiB,
			AmountFen:          capturedAmount,
			PricingVersion:     pricing.Version,
			SourceService:      "api-gateway",
			Status:             models.TrafficUsageStatusConfirmed,
			ConfirmedAt:        &now,
		}
		if err := s.repo.CreateUsageTx(ctx, tx, usage); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   order.UserID,
			OrderNo:                  order.OrderNo,
			HoldNo:                   hold.HoldNo,
			HistoryID:                order.HistoryID,
			TaskID:                   order.TaskID,
			TransferID:               transferID,
			EntryType:                models.LedgerEntryTypeCapture,
			Scene:                    order.Scene,
			ActionAmountFen:          capturedAmount,
			AvailableDeltaFen:        money.Zero(),
			ReservedDeltaFen:         capturedAmount.Neg(),
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			Remark:                   "capture file transfer",
			CreatedAt:                now,
		}
		if err := s.repo.CreateLedgerTx(ctx, tx, entry); err != nil {
			return err
		}

		releaseAmount := remainingHoldAmount(hold)
		if releaseAmount.Cmp(money.Zero()) > 0 {
			hold.ReleasedAmountFen = hold.ReleasedAmountFen.Add(releaseAmount)
			if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
				return err
			}
			order.ReleasedAmountFen = order.ReleasedAmountFen.Add(releaseAmount)
			order.Status = deriveOrderStatus(order)
			if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
				return err
			}
			account.AvailableBalanceFen = account.AvailableBalanceFen.Add(releaseAmount)
			account.ReservedBalanceFen = account.ReservedBalanceFen.Sub(releaseAmount)
			if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
				return err
			}
			releaseEntry := &models.BillingLedgerEntry{
				EntryNo:                  newBillingID("led"),
				AccountID:                account.ID,
				UserID:                   order.UserID,
				OrderNo:                  order.OrderNo,
				HoldNo:                   hold.HoldNo,
				HistoryID:                order.HistoryID,
				TaskID:                   order.TaskID,
				TransferID:               transferID,
				EntryType:                models.LedgerEntryTypeRelease,
				Scene:                    order.Scene,
				ActionAmountFen:          releaseAmount,
				AvailableDeltaFen:        releaseAmount,
				ReservedDeltaFen:         releaseAmount.Neg(),
				BalanceAfterAvailableFen: account.AvailableBalanceFen,
				BalanceAfterReservedFen:  account.ReservedBalanceFen,
				Remark:                   "release unused transfer reserve",
				CreatedAt:                time.Now(),
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

		hold.ReleasedAmountFen = hold.ReleasedAmountFen.Add(releasedAmount)
		hold.Status = models.BillingHoldStatusReleased
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return err
		}

		order.ReleasedAmountFen = order.ReleasedAmountFen.Add(releasedAmount)
		order.Remark = reason
		order.Status = deriveOrderStatus(order)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return err
		}

		account.AvailableBalanceFen = account.AvailableBalanceFen.Add(releasedAmount)
		account.ReservedBalanceFen = account.ReservedBalanceFen.Sub(releasedAmount)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		entry := &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   order.UserID,
			OrderNo:                  order.OrderNo,
			HoldNo:                   hold.HoldNo,
			HistoryID:                order.HistoryID,
			TaskID:                   order.TaskID,
			TransferID:               transferID,
			EntryType:                models.LedgerEntryTypeRelease,
			Scene:                    order.Scene,
			ActionAmountFen:          releasedAmount,
			AvailableDeltaFen:        releasedAmount,
			ReservedDeltaFen:         releasedAmount.Neg(),
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			Remark:                   reason,
			CreatedAt:                time.Now(),
		}
		return s.repo.CreateLedgerTx(ctx, tx, entry)
	})
	if err != nil {
		return nil, money.Zero(), err
	}

	return order, releasedAmount, nil
}

func (s *BillingService) resolveInitialDownloadShortfall(ctx context.Context, tx *sql.Tx, order *models.BillingChargeOrder, account *models.BillingAccount, remark, operatorUserID string) (*models.BillingLedgerEntry, error) {
	if order == nil || order.Scene != models.BillingSceneDownload || order.ShortfallFen.IsZero() {
		return nil, nil
	}
	if account.AvailableBalanceFen.Cmp(order.ShortfallFen) < 0 {
		setOrderAwaitingShortfall(order, order.ShortfallFen, order.Remark)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return nil, err
		}
		return nil, ErrInsufficientBalance
	}

	now := time.Now()
	shortfallFen := order.ShortfallFen
	account.AvailableBalanceFen = account.AvailableBalanceFen.Sub(shortfallFen)
	account.ReservedBalanceFen = account.ReservedBalanceFen.Add(shortfallFen)
	order.HeldAmountFen = order.HeldAmountFen.Add(shortfallFen)

	holdEntry := newReserveLedgerEntry(account, order, "", shortfallFen, remarkOrDefault(remark, "resolve initial order shortfall"), operatorUserID, now)

	if order.ActualIngressBytes > 0 && order.CapturedAmountFen.IsZero() {
		hold, err := s.repo.GetHoldByTaskIDForUpdate(ctx, tx, order.TaskID, models.BillingHoldTypeDownloadTotal)
		if err != nil {
			return nil, err
		}
		pricing, err := s.repo.GetPricingByVersion(ctx, order.PricingVersion)
		if err != nil {
			return nil, err
		}
		capturedAmount, err := calculateAmountFen(order.ActualIngressBytes, pricing.IngressPriceFenPerGiB)
		if err != nil {
			return nil, err
		}

		hold.AmountFen = hold.AmountFen.Add(shortfallFen)
		hold.CapturedAmountFen = hold.CapturedAmountFen.Add(capturedAmount)
		hold.Status = deriveHoldStatus(hold)
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return nil, err
		}

		order.ShortfallFen = money.Zero()
		order.Remark = remarkOrDefault(remark, "shortfall resolved")
		order.CapturedAmountFen = order.CapturedAmountFen.Add(capturedAmount)
		order.Status = deriveOrderStatus(order)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return nil, err
		}

		account.ReservedBalanceFen = account.ReservedBalanceFen.Sub(capturedAmount)
		account.TotalSpentFen = account.TotalSpentFen.Add(capturedAmount)
		account.TotalTrafficBytes += order.ActualIngressBytes
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return nil, err
		}

		holdEntry.HoldNo = hold.HoldNo
		holdEntry.BalanceAfterReservedFen = account.ReservedBalanceFen.Add(capturedAmount)
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
			UnitPriceFenPerGiB: pricing.IngressPriceFenPerGiB,
			AmountFen:          capturedAmount,
			PricingVersion:     pricing.Version,
			SourceService:      "media-service",
			Status:             models.TrafficUsageStatusConfirmed,
			ConfirmedAt:        &now,
		}
		if err := s.repo.CreateUsageTx(ctx, tx, usage); err != nil {
			return nil, err
		}

		captureEntry := &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   order.UserID,
			OrderNo:                  order.OrderNo,
			HoldNo:                   hold.HoldNo,
			HistoryID:                order.HistoryID,
			TaskID:                   order.TaskID,
			EntryType:                models.LedgerEntryTypeCapture,
			Scene:                    order.Scene,
			ActionAmountFen:          capturedAmount,
			AvailableDeltaFen:        money.Zero(),
			ReservedDeltaFen:         capturedAmount.Neg(),
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			Remark:                   "capture ingress usage after shortfall resolution",
			CreatedAt:                now,
		}
		return holdEntry, s.repo.CreateLedgerTx(ctx, tx, captureEntry)
	}

	order.ShortfallFen = money.Zero()
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
	if order == nil || order.ShortfallFen.IsZero() || order.ActualEgressBytes <= 0 {
		return nil, nil
	}
	if account.AvailableBalanceFen.Cmp(order.ShortfallFen) < 0 {
		setOrderAwaitingShortfall(order, order.ShortfallFen, order.Remark)
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
	capturedAmount, err := calculateAmountFen(order.ActualEgressBytes, pricing.EgressPriceFenPerGiB)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	shortfallFen := order.ShortfallFen
	account.AvailableBalanceFen = account.AvailableBalanceFen.Sub(shortfallFen)
	account.ReservedBalanceFen = account.ReservedBalanceFen.Add(shortfallFen)
	order.HeldAmountFen = order.HeldAmountFen.Add(shortfallFen)

	holdEntry := newReserveLedgerEntry(account, order, hold.HoldNo, shortfallFen, remarkOrDefault(remark, "resolve transfer shortfall"), operatorUserID, now)

	holdDiff := capturedAmount.Sub(hold.AmountFen)
	if holdDiff.Cmp(money.Zero()) > 0 {
		hold.AmountFen = hold.AmountFen.Add(holdDiff)
	}
	hold.CapturedAmountFen = hold.CapturedAmountFen.Add(capturedAmount)
	hold.Status = models.BillingHoldStatusCaptured
	if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
		return nil, err
	}

	order.ShortfallFen = money.Zero()
	order.Remark = remarkOrDefault(remark, "shortfall resolved")
	order.CapturedAmountFen = order.CapturedAmountFen.Add(capturedAmount)
	order.Status = deriveOrderStatus(order)
	if order.Status == models.BillingOrderStatusCaptured {
		order.ClosedAt = &now
	}
	if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
		return nil, err
	}

	account.ReservedBalanceFen = account.ReservedBalanceFen.Sub(capturedAmount)
	account.TotalSpentFen = account.TotalSpentFen.Add(capturedAmount)
	account.TotalTrafficBytes += order.ActualEgressBytes
	if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
		return nil, err
	}

	holdEntry.BalanceAfterReservedFen = account.ReservedBalanceFen.Add(capturedAmount)
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
		UnitPriceFenPerGiB: pricing.EgressPriceFenPerGiB,
		AmountFen:          capturedAmount,
		PricingVersion:     pricing.Version,
		SourceService:      "api-gateway",
		Status:             models.TrafficUsageStatusConfirmed,
		ConfirmedAt:        &now,
	}
	if err := s.repo.CreateUsageTx(ctx, tx, usage); err != nil {
		return nil, err
	}

	captureEntry := &models.BillingLedgerEntry{
		EntryNo:                  newBillingID("led"),
		AccountID:                account.ID,
		UserID:                   order.UserID,
		OrderNo:                  order.OrderNo,
		HoldNo:                   hold.HoldNo,
		HistoryID:                order.HistoryID,
		TaskID:                   order.TaskID,
		TransferID:               hold.TransferID,
		EntryType:                models.LedgerEntryTypeCapture,
		Scene:                    order.Scene,
		ActionAmountFen:          capturedAmount,
		AvailableDeltaFen:        money.Zero(),
		ReservedDeltaFen:         capturedAmount.Neg(),
		BalanceAfterAvailableFen: account.AvailableBalanceFen,
		BalanceAfterReservedFen:  account.ReservedBalanceFen,
		Remark:                   "capture file transfer after shortfall resolution",
		CreatedAt:                now,
	}
	if err := s.repo.CreateLedgerTx(ctx, tx, captureEntry); err != nil {
		return nil, err
	}

	releaseAmount := remainingHoldAmount(hold)
	if releaseAmount.Cmp(money.Zero()) > 0 {
		hold.ReleasedAmountFen = hold.ReleasedAmountFen.Add(releaseAmount)
		if err := s.repo.UpdateHoldTx(ctx, tx, hold); err != nil {
			return nil, err
		}
		order.ReleasedAmountFen = order.ReleasedAmountFen.Add(releaseAmount)
		order.Status = deriveOrderStatus(order)
		if err := s.repo.UpdateOrderTx(ctx, tx, order); err != nil {
			return nil, err
		}
		account.AvailableBalanceFen = account.AvailableBalanceFen.Add(releaseAmount)
		account.ReservedBalanceFen = account.ReservedBalanceFen.Sub(releaseAmount)
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return nil, err
		}
		releaseEntry := &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   order.UserID,
			OrderNo:                  order.OrderNo,
			HoldNo:                   hold.HoldNo,
			HistoryID:                order.HistoryID,
			TaskID:                   order.TaskID,
			TransferID:               hold.TransferID,
			EntryType:                models.LedgerEntryTypeRelease,
			Scene:                    order.Scene,
			ActionAmountFen:          releaseAmount,
			AvailableDeltaFen:        releaseAmount,
			ReservedDeltaFen:         releaseAmount.Neg(),
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			Remark:                   "release unused reserve after shortfall resolution",
			CreatedAt:                now,
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

func (s *BillingService) AdjustBillingBalance(ctx context.Context, userID, operationID string, amountFen money.Decimal, remark, operatorUserID string) (*models.BillingAccount, *models.BillingLedgerEntry, error) {
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
		if amountFen.Cmp(money.Zero()) < 0 && account.AvailableBalanceFen.Cmp(amountFen.Neg()) < 0 {
			return ErrInsufficientBalance
		}

		account.AvailableBalanceFen = account.AvailableBalanceFen.Add(amountFen)
		if amountFen.Cmp(money.Zero()) > 0 {
			account.TotalRechargedFen = account.TotalRechargedFen.Add(amountFen)
		}
		if err := s.repo.UpdateAccountTx(ctx, tx, account); err != nil {
			return err
		}

		entryType := int32(models.LedgerEntryTypeManualAdjustment)
		if amountFen.Cmp(money.Zero()) > 0 {
			entryType = models.LedgerEntryTypeManualTopup
		}
		entry = &models.BillingLedgerEntry{
			EntryNo:                  newBillingID("led"),
			AccountID:                account.ID,
			UserID:                   userID,
			OperationID:              operationID,
			EntryType:                entryType,
			Scene:                    models.BillingSceneAdmin,
			ActionAmountFen:          abs64(amountFen),
			AvailableDeltaFen:        amountFen,
			ReservedDeltaFen:         money.Zero(),
			BalanceAfterAvailableFen: account.AvailableBalanceFen,
			BalanceAfterReservedFen:  account.ReservedBalanceFen,
			OperatorUserID:           operatorUserID,
			Remark:                   remark,
			CreatedAt:                time.Now(),
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

		if order.ShortfallFen.IsZero() || order.Status != models.BillingOrderStatusAwaitingShortfall {
			return nil
		}

		switch {
		case order.Scene == models.BillingSceneDownload && order.ActualIngressBytes > 0 && order.CapturedAmountFen.IsZero():
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
			IngressPriceFenPerGiB: parsedIngressPrice,
			EgressPriceFenPerGiB:  parsedEgressPrice,
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
	case order.ShortfallFen.Cmp(money.Zero()) > 0:
		return models.BillingOrderStatusAwaitingShortfall
	case order.CapturedAmountFen.IsZero() && order.ReleasedAmountFen.IsZero():
		return models.BillingOrderStatusHeld
	case order.CapturedAmountFen.IsZero() && remaining.IsZero():
		return models.BillingOrderStatusReleased
	case order.CapturedAmountFen.Cmp(money.Zero()) > 0 && order.ReleasedAmountFen.Cmp(money.Zero()) > 0:
		return models.BillingOrderStatusPartialCaptured
	case order.CapturedAmountFen.Cmp(money.Zero()) > 0 && remaining.Cmp(money.Zero()) > 0:
		return models.BillingOrderStatusPartialCaptured
	case order.CapturedAmountFen.Cmp(money.Zero()) > 0 && remaining.IsZero():
		return models.BillingOrderStatusCaptured
	default:
		return models.BillingOrderStatusPartialCaptured
	}
}

func deriveHoldStatus(hold *models.BillingHold) int32 {
	remaining := remainingHoldAmount(hold)
	switch {
	case hold.CapturedAmountFen.IsZero() && hold.ReleasedAmountFen.IsZero():
		return models.BillingHoldStatusHeld
	case hold.CapturedAmountFen.Cmp(money.Zero()) > 0 && remaining.Cmp(money.Zero()) > 0:
		return models.BillingHoldStatusPartialCaptured
	case hold.CapturedAmountFen.Cmp(money.Zero()) > 0 && remaining.IsZero():
		return models.BillingHoldStatusCaptured
	case hold.ReleasedAmountFen.Cmp(money.Zero()) > 0 && remaining.IsZero():
		return models.BillingHoldStatusReleased
	default:
		return models.BillingHoldStatusHeld
	}
}

func setOrderAwaitingShortfall(order *models.BillingChargeOrder, shortfallFen money.Decimal, remark string) {
	if shortfallFen.Cmp(money.Zero()) < 0 {
		shortfallFen = money.Zero()
	}
	order.ShortfallFen = shortfallFen
	if remark != "" {
		order.Remark = remark
	}
	order.Status = deriveOrderStatus(order)
}

func newReserveLedgerEntry(account *models.BillingAccount, order *models.BillingChargeOrder, holdNo string, amountFen money.Decimal, remark, operatorUserID string, createdAt time.Time) *models.BillingLedgerEntry {
	return &models.BillingLedgerEntry{
		EntryNo:                  newBillingID("led"),
		AccountID:                account.ID,
		UserID:                   order.UserID,
		OrderNo:                  order.OrderNo,
		HoldNo:                   holdNo,
		HistoryID:                order.HistoryID,
		TaskID:                   order.TaskID,
		EntryType:                models.LedgerEntryTypeHold,
		Scene:                    order.Scene,
		ActionAmountFen:          amountFen,
		AvailableDeltaFen:        amountFen.Neg(),
		ReservedDeltaFen:         amountFen,
		BalanceAfterAvailableFen: account.AvailableBalanceFen,
		BalanceAfterReservedFen:  account.ReservedBalanceFen,
		OperatorUserID:           operatorUserID,
		Remark:                   remark,
		CreatedAt:                createdAt,
	}
}

func remarkOrDefault(remark, fallback string) string {
	if remark != "" {
		return remark
	}
	return fallback
}

func remainingOrderReserve(order *models.BillingChargeOrder) money.Decimal {
	remaining := order.HeldAmountFen.Sub(order.CapturedAmountFen).Sub(order.ReleasedAmountFen)
	if remaining.Cmp(money.Zero()) < 0 {
		return money.Zero()
	}
	return remaining
}

func remainingHoldAmount(hold *models.BillingHold) money.Decimal {
	remaining := hold.AmountFen.Sub(hold.CapturedAmountFen).Sub(hold.ReleasedAmountFen)
	if remaining.Cmp(money.Zero()) < 0 {
		return money.Zero()
	}
	return remaining
}

func calculateAmountFen(bytes int64, pricePerGiB money.Decimal) (money.Decimal, error) {
	if bytes <= 0 {
		return money.Zero(), nil
	}

	billableMB := bytes / mbBytes
	if bytes%mbBytes != 0 {
		billableMB++
	}
	if billableMB < minBillableMB {
		billableMB = minBillableMB
	}

	return pricePerGiB.Mul(money.FromInt64(billableMB)).DivInt64(gbMB), nil
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
