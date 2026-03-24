package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"youdlp/asset-service/internal/models"
)

// BillingRepository 账务仓储
type BillingRepository struct {
	db *sql.DB
}

// NewBillingRepository 创建账务仓储
func NewBillingRepository(db *sql.DB) *BillingRepository {
	return &BillingRepository{db: db}
}

// WithTx 在事务中执行
func (r *BillingRepository) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx rollback failed after %v: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}
	return nil
}

func (r *BillingRepository) GetOrCreateAccount(ctx context.Context, userID string) (*models.BillingAccount, error) {
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO billing_accounts (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return nil, fmt.Errorf("failed to ensure billing account: %w", err)
	}

	return r.GetAccountByUserID(ctx, userID)
}

func (r *BillingRepository) GetOrCreateAccountTx(ctx context.Context, tx *sql.Tx, userID string) (*models.BillingAccount, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO billing_accounts (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return nil, fmt.Errorf("failed to ensure billing account in tx: %w", err)
	}

	return r.GetAccountByUserIDForUpdate(ctx, tx, userID)
}

func (r *BillingRepository) GetAccountByUserID(ctx context.Context, userID string) (*models.BillingAccount, error) {
	return scanBillingAccount(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, currency_code, available_balance_yuan, reserved_balance_yuan,
		       total_recharged_yuan, total_spent_yuan, total_traffic_bytes, status, version,
		       created_at, updated_at
		FROM billing_accounts
		WHERE user_id = $1
	`, userID))
}

func (r *BillingRepository) GetAccountByUserIDForUpdate(ctx context.Context, tx *sql.Tx, userID string) (*models.BillingAccount, error) {
	return scanBillingAccount(tx.QueryRowContext(ctx, `
		SELECT id, user_id, currency_code, available_balance_yuan, reserved_balance_yuan,
		       total_recharged_yuan, total_spent_yuan, total_traffic_bytes, status, version,
		       created_at, updated_at
		FROM billing_accounts
		WHERE user_id = $1
		FOR UPDATE
	`, userID))
}

func (r *BillingRepository) UpdateAccountTx(ctx context.Context, tx *sql.Tx, account *models.BillingAccount) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE billing_accounts
		SET available_balance_yuan = $1,
		    reserved_balance_yuan = $2,
		    total_recharged_yuan = $3,
		    total_spent_yuan = $4,
		    total_traffic_bytes = $5,
		    status = $6,
		    version = version + 1,
		    updated_at = $7
		WHERE id = $8
	`, account.AvailableBalanceYuan, account.ReservedBalanceYuan, account.TotalRechargedYuan, account.TotalSpentYuan, account.TotalTrafficBytes, account.Status, time.Now(), account.ID)
	if err != nil {
		return fmt.Errorf("failed to update billing account: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect account update result: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	account.Version++
	account.UpdatedAt = time.Now()
	return nil
}

func (r *BillingRepository) ListAccounts(ctx context.Context, filter models.BillingAccountFilter) (*models.BillingAccountResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	conditions := []string{"1=1"}
	args := make([]interface{}, 0)
	argPos := 1

	if len(filter.UserIDs) > 0 {
		placeholders := make([]string, 0, len(filter.UserIDs))
		for _, userID := range filter.UserIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argPos))
			args = append(args, userID)
			argPos++
		}
		conditions = append(conditions, "user_id IN ("+strings.Join(placeholders, ",")+")")
	}

	if filter.Status > 0 {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, filter.Status)
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM billing_accounts WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count billing accounts: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, filter.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, currency_code, available_balance_yuan, reserved_balance_yuan,
		       total_recharged_yuan, total_spent_yuan, total_traffic_bytes, status, version,
		       created_at, updated_at
		FROM billing_accounts
		WHERE `+whereClause+`
		ORDER BY updated_at DESC, id DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)+1)+` OFFSET $`+fmt.Sprintf("%d", len(args)+2), queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query billing accounts: %w", err)
	}
	defer rows.Close()

	items := make([]models.BillingAccount, 0)
	for rows.Next() {
		var account models.BillingAccount
		if err := rows.Scan(
			&account.ID, &account.UserID, &account.CurrencyCode, &account.AvailableBalanceYuan,
			&account.ReservedBalanceYuan, &account.TotalRechargedYuan, &account.TotalSpentYuan,
			&account.TotalTrafficBytes, &account.Status, &account.Version, &account.CreatedAt, &account.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan billing account: %w", err)
		}
		items = append(items, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate billing accounts: %w", err)
	}

	return &models.BillingAccountResult{
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Items:    items,
	}, nil
}

func (r *BillingRepository) GetActivePricing(ctx context.Context) (*models.BillingPricing, error) {
	return scanBillingPricing(r.db.QueryRowContext(ctx, `
		SELECT id, version, ingress_price_yuan_per_gb, egress_price_yuan_per_gb,
		       enabled, remark, updated_by_user_id,
		       effective_at, created_at
		FROM billing_pricing
		WHERE enabled = TRUE
		ORDER BY version DESC
		LIMIT 1
	`))
}

func (r *BillingRepository) GetPricingByVersion(ctx context.Context, version int32) (*models.BillingPricing, error) {
	return scanBillingPricing(r.db.QueryRowContext(ctx, `
		SELECT id, version, ingress_price_yuan_per_gb, egress_price_yuan_per_gb,
		       enabled, remark, updated_by_user_id,
		       effective_at, created_at
		FROM billing_pricing
		WHERE version = $1
	`, version))
}

func (r *BillingRepository) GetLatestPricingVersionTx(ctx context.Context, tx *sql.Tx) (int32, error) {
	var version int32
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM billing_pricing`).Scan(&version); err != nil {
		return 0, fmt.Errorf("failed to query latest pricing version: %w", err)
	}
	return version, nil
}

func (r *BillingRepository) DisableAllPricingTx(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `UPDATE billing_pricing SET enabled = FALSE`); err != nil {
		return fmt.Errorf("failed to disable pricing: %w", err)
	}
	return nil
}

func (r *BillingRepository) CreatePricingTx(ctx context.Context, tx *sql.Tx, pricing *models.BillingPricing) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO billing_pricing (
			version, ingress_price_yuan_per_gb, egress_price_yuan_per_gb,
			enabled, remark, updated_by_user_id, effective_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`,
		pricing.Version, pricing.IngressPriceYuanPerGB, pricing.EgressPriceYuanPerGB,
		pricing.Enabled, pricing.Remark, pricing.UpdatedByUserID, pricing.EffectiveAt,
	).Scan(&pricing.ID, &pricing.CreatedAt)
}

func (r *BillingRepository) CreateOrderTx(ctx context.Context, tx *sql.Tx, order *models.BillingChargeOrder) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO billing_charge_orders (
			order_no, user_id, history_id, task_id, scene, status, pricing_version,
			estimated_ingress_bytes, estimated_egress_bytes, estimated_traffic_bytes,
			actual_ingress_bytes, actual_egress_bytes, actual_traffic_bytes,
			held_amount_yuan, captured_amount_yuan, released_amount_yuan, shortfall_yuan,
			remark, created_at, updated_at, closed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16, $17,
			$18, $19, $20, $21
		)
		RETURNING id
	`,
		order.OrderNo, order.UserID, order.HistoryID, order.TaskID, order.Scene, order.Status, order.PricingVersion,
		order.EstimatedIngressBytes, order.EstimatedEgressBytes, order.EstimatedTrafficBytes,
		order.ActualIngressBytes, order.ActualEgressBytes, order.ActualTrafficBytes,
		order.HeldAmountYuan, order.CapturedAmountYuan, order.ReleasedAmountYuan, order.ShortfallYuan,
		order.Remark, time.Now(), time.Now(), order.ClosedAt,
	).Scan(&order.ID)
}

func (r *BillingRepository) GetOrderByTaskID(ctx context.Context, taskID string) (*models.BillingChargeOrder, error) {
	return scanBillingChargeOrder(r.db.QueryRowContext(ctx, `
		SELECT id, order_no, user_id, history_id, task_id, scene, status, pricing_version,
		       estimated_ingress_bytes, estimated_egress_bytes, estimated_traffic_bytes,
		       actual_ingress_bytes, actual_egress_bytes, actual_traffic_bytes,
		       held_amount_yuan, captured_amount_yuan, released_amount_yuan, shortfall_yuan,
		       remark, created_at, updated_at, closed_at
		FROM billing_charge_orders
		WHERE task_id = $1
	`, taskID))
}

func (r *BillingRepository) GetOrderByTaskIDForUpdate(ctx context.Context, tx *sql.Tx, taskID string) (*models.BillingChargeOrder, error) {
	return scanBillingChargeOrder(tx.QueryRowContext(ctx, `
		SELECT id, order_no, user_id, history_id, task_id, scene, status, pricing_version,
		       estimated_ingress_bytes, estimated_egress_bytes, estimated_traffic_bytes,
		       actual_ingress_bytes, actual_egress_bytes, actual_traffic_bytes,
		       held_amount_yuan, captured_amount_yuan, released_amount_yuan, shortfall_yuan,
		       remark, created_at, updated_at, closed_at
		FROM billing_charge_orders
		WHERE task_id = $1
		FOR UPDATE
	`, taskID))
}

func (r *BillingRepository) GetLatestDownloadOrderByHistoryIDForUpdate(ctx context.Context, tx *sql.Tx, historyID int64) (*models.BillingChargeOrder, error) {
	return scanBillingChargeOrder(tx.QueryRowContext(ctx, `
		SELECT id, order_no, user_id, history_id, task_id, scene, status, pricing_version,
		       estimated_ingress_bytes, estimated_egress_bytes, estimated_traffic_bytes,
		       actual_ingress_bytes, actual_egress_bytes, actual_traffic_bytes,
		       held_amount_yuan, captured_amount_yuan, released_amount_yuan, shortfall_yuan,
		       remark, created_at, updated_at, closed_at
		FROM billing_charge_orders
		WHERE history_id = $1 AND scene = $2
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE
	`, historyID, models.BillingSceneDownload))
}

func (r *BillingRepository) GetOrderByOrderNoForUpdate(ctx context.Context, tx *sql.Tx, orderNo string) (*models.BillingChargeOrder, error) {
	return scanBillingChargeOrder(tx.QueryRowContext(ctx, `
		SELECT id, order_no, user_id, history_id, task_id, scene, status, pricing_version,
		       estimated_ingress_bytes, estimated_egress_bytes, estimated_traffic_bytes,
		       actual_ingress_bytes, actual_egress_bytes, actual_traffic_bytes,
		       held_amount_yuan, captured_amount_yuan, released_amount_yuan, shortfall_yuan,
		       remark, created_at, updated_at, closed_at
		FROM billing_charge_orders
		WHERE order_no = $1
		FOR UPDATE
	`, orderNo))
}

func (r *BillingRepository) GetLatestPendingTransferHoldByOrderNoForUpdate(ctx context.Context, tx *sql.Tx, orderNo string) (*models.BillingHold, error) {
	return scanBillingHold(tx.QueryRowContext(ctx, `
		SELECT id, hold_no, order_no, user_id, history_id, task_id, transfer_id,
		       hold_type, funding_source, status, amount_yuan, captured_amount_yuan,
		       released_amount_yuan, expires_at, created_at, updated_at
		FROM billing_holds
		WHERE order_no = $1
		  AND hold_type = $2
		  AND status IN ($3, $4)
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE
	`, orderNo, models.BillingHoldTypeFileTransfer, models.BillingHoldStatusHeld, models.BillingHoldStatusPartialCaptured))
}

func (r *BillingRepository) UpdateOrderTx(ctx context.Context, tx *sql.Tx, order *models.BillingChargeOrder) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE billing_charge_orders
		SET status = $1,
		    estimated_ingress_bytes = $2,
		    estimated_egress_bytes = $3,
		    estimated_traffic_bytes = $4,
		    actual_ingress_bytes = $5,
		    actual_egress_bytes = $6,
		    actual_traffic_bytes = $7,
		    held_amount_yuan = $8,
		    captured_amount_yuan = $9,
		    released_amount_yuan = $10,
		    shortfall_yuan = $11,
		    remark = $12,
		    updated_at = $13,
		    closed_at = $14
		WHERE id = $15
	`,
		order.Status, order.EstimatedIngressBytes, order.EstimatedEgressBytes, order.EstimatedTrafficBytes,
		order.ActualIngressBytes, order.ActualEgressBytes, order.ActualTrafficBytes, order.HeldAmountYuan,
		order.CapturedAmountYuan, order.ReleasedAmountYuan, order.ShortfallYuan, order.Remark, time.Now(), order.ClosedAt, order.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update billing order: %w", err)
	}
	order.UpdatedAt = time.Now()
	return nil
}

func (r *BillingRepository) CreateHoldTx(ctx context.Context, tx *sql.Tx, hold *models.BillingHold) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO billing_holds (
			hold_no, order_no, user_id, history_id, task_id, transfer_id,
			hold_type, funding_source, status, amount_yuan, captured_amount_yuan,
			released_amount_yuan, expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15
		)
		RETURNING id
	`,
		hold.HoldNo, hold.OrderNo, hold.UserID, hold.HistoryID, hold.TaskID, hold.TransferID,
		hold.HoldType, hold.FundingSource, hold.Status, hold.AmountYuan, hold.CapturedAmountYuan,
		hold.ReleasedAmountYuan, hold.ExpiresAt, time.Now(), time.Now(),
	).Scan(&hold.ID)
}

func (r *BillingRepository) GetHoldByTaskIDForUpdate(ctx context.Context, tx *sql.Tx, taskID string, holdType int32) (*models.BillingHold, error) {
	return scanBillingHold(tx.QueryRowContext(ctx, `
		SELECT id, hold_no, order_no, user_id, history_id, task_id, transfer_id,
		       hold_type, funding_source, status, amount_yuan, captured_amount_yuan,
		       released_amount_yuan, expires_at, created_at, updated_at
		FROM billing_holds
		WHERE task_id = $1 AND hold_type = $2
		FOR UPDATE
	`, taskID, holdType))
}

func (r *BillingRepository) GetHoldByTransferIDForUpdate(ctx context.Context, tx *sql.Tx, transferID string) (*models.BillingHold, error) {
	return scanBillingHold(tx.QueryRowContext(ctx, `
		SELECT id, hold_no, order_no, user_id, history_id, task_id, transfer_id,
		       hold_type, funding_source, status, amount_yuan, captured_amount_yuan,
		       released_amount_yuan, expires_at, created_at, updated_at
		FROM billing_holds
		WHERE transfer_id = $1
		FOR UPDATE
	`, transferID))
}

func (r *BillingRepository) UpdateHoldTx(ctx context.Context, tx *sql.Tx, hold *models.BillingHold) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE billing_holds
		SET funding_source = $1,
		    status = $2,
		    amount_yuan = $3,
		    captured_amount_yuan = $4,
		    released_amount_yuan = $5,
		    expires_at = $6,
		    updated_at = $7
		WHERE id = $8
	`, hold.FundingSource, hold.Status, hold.AmountYuan, hold.CapturedAmountYuan, hold.ReleasedAmountYuan, hold.ExpiresAt, time.Now(), hold.ID)
	if err != nil {
		return fmt.Errorf("failed to update hold: %w", err)
	}
	hold.UpdatedAt = time.Now()
	return nil
}

func (r *BillingRepository) CreateUsageTx(ctx context.Context, tx *sql.Tx, usage *models.TrafficUsageRecord) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO traffic_usage_records (
			usage_no, order_no, user_id, history_id, task_id, transfer_id,
			direction, traffic_bytes, unit_price_yuan_per_gb, amount_yuan,
			pricing_version, source_service, status, created_at, confirmed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15
		)
		RETURNING id
	`,
		usage.UsageNo, usage.OrderNo, usage.UserID, usage.HistoryID, usage.TaskID, usage.TransferID,
		usage.Direction, usage.TrafficBytes, usage.UnitPriceYuanPerGB, usage.AmountYuan,
		usage.PricingVersion, usage.SourceService, usage.Status, time.Now(), usage.ConfirmedAt,
	).Scan(&usage.ID)
}

func (r *BillingRepository) CreateLedgerTx(ctx context.Context, tx *sql.Tx, entry *models.BillingLedgerEntry) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO billing_ledger_entries (
			entry_no, account_id, user_id, order_no, hold_no, history_id,
			task_id, transfer_id, operation_id, entry_type, scene, action_amount_yuan,
			available_delta_yuan, reserved_delta_yuan, balance_after_available_yuan,
			balance_after_reserved_yuan, operator_user_id, remark, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19
		)
		RETURNING id
	`,
		entry.EntryNo, entry.AccountID, entry.UserID, entry.OrderNo, entry.HoldNo, entry.HistoryID,
		entry.TaskID, entry.TransferID, entry.OperationID, entry.EntryType, entry.Scene, entry.ActionAmountYuan,
		entry.AvailableDeltaYuan, entry.ReservedDeltaYuan, entry.BalanceAfterAvailableYuan,
		entry.BalanceAfterReservedYuan, entry.OperatorUserID, entry.Remark, time.Now(),
	).Scan(&entry.ID)
}

func (r *BillingRepository) GetLedgerByOperationID(ctx context.Context, operationID string) (*models.BillingLedgerEntry, error) {
	return scanBillingLedgerEntry(r.db.QueryRowContext(ctx, `
		SELECT id, entry_no, account_id, user_id, order_no, hold_no, history_id,
		       task_id, transfer_id, operation_id, entry_type, scene, action_amount_yuan,
		       available_delta_yuan, reserved_delta_yuan, balance_after_available_yuan,
		       balance_after_reserved_yuan, operator_user_id, remark, created_at
		FROM billing_ledger_entries
		WHERE operation_id = $1
	`, operationID))
}

func (r *BillingRepository) CreateWelcomeCreditGrantTx(ctx context.Context, tx *sql.Tx, grant *models.WelcomeCreditGrant) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO welcome_credit_grants (
			user_id, operation_id, ledger_entry_no, reason_code, amount_yuan, currency_code, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		RETURNING id
	`,
		grant.UserID, grant.OperationID, grant.LedgerEntryNo, grant.ReasonCode, grant.AmountYuan, grant.CurrencyCode, grant.CreatedAt,
	).Scan(&grant.ID)
}

func (r *BillingRepository) GetWelcomeCreditGrantByOperationID(ctx context.Context, operationID string) (*models.WelcomeCreditGrant, error) {
	return scanWelcomeCreditGrant(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, operation_id, ledger_entry_no, reason_code, amount_yuan, currency_code, created_at
		FROM welcome_credit_grants
		WHERE operation_id = $1
	`, operationID))
}

func (r *BillingRepository) ListLedger(ctx context.Context, filter models.BillingLedgerFilter) (*models.BillingLedgerResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	conditions := []string{"1=1"}
	args := make([]interface{}, 0)
	argPos := 1
	if filter.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argPos))
		args = append(args, filter.UserID)
		argPos++
	}
	if filter.EntryType > 0 {
		conditions = append(conditions, fmt.Sprintf("entry_type = $%d", argPos))
		args = append(args, filter.EntryType)
		argPos++
	}
	whereClause := strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM billing_ledger_entries WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count billing ledger: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, filter.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, entry_no, account_id, user_id, order_no, hold_no, history_id,
		       task_id, transfer_id, operation_id, entry_type, scene, action_amount_yuan,
		       available_delta_yuan, reserved_delta_yuan, balance_after_available_yuan,
		       balance_after_reserved_yuan, operator_user_id, remark, created_at
		FROM billing_ledger_entries
		WHERE `+whereClause+`
		ORDER BY created_at DESC, id DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)+1)+` OFFSET $`+fmt.Sprintf("%d", len(args)+2), queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query billing ledger: %w", err)
	}
	defer rows.Close()

	items := make([]models.BillingLedgerEntry, 0)
	for rows.Next() {
		item, err := scanBillingLedgerEntryRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate billing ledger: %w", err)
	}

	return &models.BillingLedgerResult{
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Items:    items,
	}, nil
}

func (r *BillingRepository) ListUsageRecords(ctx context.Context, filter models.TrafficUsageFilter) (*models.TrafficUsageResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	conditions := []string{"1=1"}
	args := make([]interface{}, 0)
	argPos := 1
	if filter.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argPos))
		args = append(args, filter.UserID)
		argPos++
	}
	if filter.Direction > 0 {
		conditions = append(conditions, fmt.Sprintf("direction = $%d", argPos))
		args = append(args, filter.Direction)
		argPos++
	}
	whereClause := strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM traffic_usage_records WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count usage records: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, filter.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, usage_no, order_no, user_id, history_id, task_id, transfer_id,
		       direction, traffic_bytes, unit_price_yuan_per_gb, amount_yuan,
		       pricing_version, source_service, status, created_at, confirmed_at
		FROM traffic_usage_records
		WHERE `+whereClause+`
		ORDER BY created_at DESC, id DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)+1)+` OFFSET $`+fmt.Sprintf("%d", len(args)+2), queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage records: %w", err)
	}
	defer rows.Close()

	items := make([]models.TrafficUsageRecord, 0)
	for rows.Next() {
		item, err := scanTrafficUsageRecordRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate usage records: %w", err)
	}

	return &models.TrafficUsageResult{
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Items:    items,
	}, nil
}

func (r *BillingRepository) ListStatements(ctx context.Context, userID string, page, pageSize int, statementType, statementStatus int32) (*models.BillingStatementResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	baseArgs := []interface{}{userID}
	filters := []string{"1=1"}
	argPos := 2
	if statementType > 0 {
		filters = append(filters, fmt.Sprintf("type = $%d", argPos))
		baseArgs = append(baseArgs, statementType)
		argPos++
	}
	if statementStatus > 0 {
		filters = append(filters, fmt.Sprintf("status = $%d", argPos))
		baseArgs = append(baseArgs, statementStatus)
		argPos++
	}

	filterClause := strings.Join(filters, " AND ")
	baseQuery := `
		SELECT *
		FROM (
			SELECT
				order_no AS statement_id,
				2 AS type,
				history_id,
				actual_traffic_bytes AS traffic_bytes,
				captured_amount_yuan AS amount_yuan,
				status,
				remark,
				created_at
			FROM billing_charge_orders
			WHERE user_id = $1 AND captured_amount_yuan > 0

			UNION ALL

			SELECT
				entry_no AS statement_id,
				CASE WHEN entry_type = 1 THEN 1 ELSE 3 END AS type,
				history_id,
				0 AS traffic_bytes,
				action_amount_yuan AS amount_yuan,
				3 AS status,
				remark,
				created_at
			FROM billing_ledger_entries
			WHERE user_id = $1 AND entry_type IN (1, 2)
		) statements
		WHERE ` + filterClause

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (`+baseQuery+`) counted`, baseArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count billing statements: %w", err)
	}

	offset := (page - 1) * pageSize
	args := append([]interface{}{}, baseArgs...)
	args = append(args, pageSize, offset)
	rows, err := r.db.QueryContext(ctx, baseQuery+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprintf("%d", len(baseArgs)+1)+` OFFSET $`+fmt.Sprintf("%d", len(baseArgs)+2), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query billing statements: %w", err)
	}
	defer rows.Close()

	items := make([]models.BillingStatementItem, 0)
	for rows.Next() {
		var item models.BillingStatementItem
		if err := rows.Scan(
			&item.StatementID,
			&item.Type,
			&item.HistoryID,
			&item.TrafficBytes,
			&item.AmountYuan,
			&item.Status,
			&item.Remark,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan billing statement: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate billing statements: %w", err)
	}

	return &models.BillingStatementResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}, nil
}

func (r *BillingRepository) ListShortfallOrders(ctx context.Context, filter models.BillingShortfallFilter) (*models.BillingShortfallResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	args := make([]interface{}, 0, 3)
	filters := []string{"status = $1", "shortfall_yuan > 0"}
	args = append(args, models.BillingOrderStatusAwaitingShortfall)
	argPos := 2
	if filter.UserID != "" {
		filters = append(filters, fmt.Sprintf("user_id = $%d", argPos))
		args = append(args, filter.UserID)
		argPos++
	}

	whereClause := strings.Join(filters, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM billing_charge_orders
		WHERE `+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count billing shortfalls: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, `
		SELECT order_no, user_id, history_id, task_id, scene, status, pricing_version,
		       actual_ingress_bytes, actual_egress_bytes, actual_traffic_bytes,
		       held_amount_yuan, captured_amount_yuan, released_amount_yuan, shortfall_yuan,
		       remark, created_at, updated_at
		FROM billing_charge_orders
		WHERE `+whereClause+`
		ORDER BY updated_at DESC, created_at DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)-1)+` OFFSET $`+fmt.Sprintf("%d", len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query billing shortfalls: %w", err)
	}
	defer rows.Close()

	items := make([]models.BillingShortfallOrder, 0)
	for rows.Next() {
		var item models.BillingShortfallOrder
		if err := rows.Scan(
			&item.OrderNo,
			&item.UserID,
			&item.HistoryID,
			&item.TaskID,
			&item.Scene,
			&item.Status,
			&item.PricingVersion,
			&item.ActualIngressBytes,
			&item.ActualEgressBytes,
			&item.ActualTrafficBytes,
			&item.HeldAmountYuan,
			&item.CapturedAmountYuan,
			&item.ReleasedAmountYuan,
			&item.ShortfallYuan,
			&item.Remark,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan billing shortfall: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate billing shortfalls: %w", err)
	}

	return &models.BillingShortfallResult{
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Items:    items,
	}, nil
}

func scanBillingAccount(row rowScanner) (*models.BillingAccount, error) {
	var account models.BillingAccount
	if err := row.Scan(
		&account.ID, &account.UserID, &account.CurrencyCode, &account.AvailableBalanceYuan,
		&account.ReservedBalanceYuan, &account.TotalRechargedYuan, &account.TotalSpentYuan,
		&account.TotalTrafficBytes, &account.Status, &account.Version, &account.CreatedAt, &account.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &account, nil
}

func scanBillingPricing(row rowScanner) (*models.BillingPricing, error) {
	var pricing models.BillingPricing
	if err := row.Scan(
		&pricing.ID, &pricing.Version, &pricing.IngressPriceYuanPerGB, &pricing.EgressPriceYuanPerGB,
		&pricing.Enabled, &pricing.Remark, &pricing.UpdatedByUserID,
		&pricing.EffectiveAt, &pricing.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &pricing, nil
}

func scanBillingChargeOrder(row rowScanner) (*models.BillingChargeOrder, error) {
	var order models.BillingChargeOrder
	var closedAt sql.NullTime
	if err := row.Scan(
		&order.ID, &order.OrderNo, &order.UserID, &order.HistoryID, &order.TaskID, &order.Scene, &order.Status, &order.PricingVersion,
		&order.EstimatedIngressBytes, &order.EstimatedEgressBytes, &order.EstimatedTrafficBytes,
		&order.ActualIngressBytes, &order.ActualEgressBytes, &order.ActualTrafficBytes,
		&order.HeldAmountYuan, &order.CapturedAmountYuan, &order.ReleasedAmountYuan, &order.ShortfallYuan,
		&order.Remark, &order.CreatedAt, &order.UpdatedAt, &closedAt,
	); err != nil {
		return nil, err
	}
	if closedAt.Valid {
		order.ClosedAt = &closedAt.Time
	}
	return &order, nil
}

func scanBillingHold(row rowScanner) (*models.BillingHold, error) {
	var hold models.BillingHold
	var expiresAt sql.NullTime
	if err := row.Scan(
		&hold.ID, &hold.HoldNo, &hold.OrderNo, &hold.UserID, &hold.HistoryID, &hold.TaskID, &hold.TransferID,
		&hold.HoldType, &hold.FundingSource, &hold.Status, &hold.AmountYuan, &hold.CapturedAmountYuan,
		&hold.ReleasedAmountYuan, &expiresAt, &hold.CreatedAt, &hold.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		hold.ExpiresAt = &expiresAt.Time
	}
	return &hold, nil
}

func scanBillingLedgerEntry(row rowScanner) (*models.BillingLedgerEntry, error) {
	var entry models.BillingLedgerEntry
	if err := row.Scan(
		&entry.ID, &entry.EntryNo, &entry.AccountID, &entry.UserID, &entry.OrderNo, &entry.HoldNo, &entry.HistoryID,
		&entry.TaskID, &entry.TransferID, &entry.OperationID, &entry.EntryType, &entry.Scene, &entry.ActionAmountYuan,
		&entry.AvailableDeltaYuan, &entry.ReservedDeltaYuan, &entry.BalanceAfterAvailableYuan,
		&entry.BalanceAfterReservedYuan, &entry.OperatorUserID, &entry.Remark, &entry.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &entry, nil
}

func scanBillingLedgerEntryRows(rows *sql.Rows) (*models.BillingLedgerEntry, error) {
	return scanBillingLedgerEntry(rows)
}

func scanWelcomeCreditGrant(row rowScanner) (*models.WelcomeCreditGrant, error) {
	var grant models.WelcomeCreditGrant
	if err := row.Scan(
		&grant.ID,
		&grant.UserID,
		&grant.OperationID,
		&grant.LedgerEntryNo,
		&grant.ReasonCode,
		&grant.AmountYuan,
		&grant.CurrencyCode,
		&grant.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &grant, nil
}

func scanTrafficUsageRecordRows(rows *sql.Rows) (*models.TrafficUsageRecord, error) {
	var item models.TrafficUsageRecord
	var confirmedAt sql.NullTime
	if err := rows.Scan(
		&item.ID, &item.UsageNo, &item.OrderNo, &item.UserID, &item.HistoryID, &item.TaskID, &item.TransferID,
		&item.Direction, &item.TrafficBytes, &item.UnitPriceYuanPerGB, &item.AmountYuan,
		&item.PricingVersion, &item.SourceService, &item.Status, &item.CreatedAt, &confirmedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan usage record: %w", err)
	}
	if confirmedAt.Valid {
		item.ConfirmedAt = &confirmedAt.Time
	}
	return &item, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}
