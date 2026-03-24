package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/repository"
)

func TestGrantWelcomeCreditIdempotent(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	svc := NewBillingService(repository.NewBillingRepository(db), repository.NewWelcomeCreditSettingsRepository(db))
	now := time.Now()

	expectEnsureAccount(mock, "u-1", "0", "0", 1, now)
	mock.ExpectQuery(`FROM billing_ledger_entries\s+WHERE operation_id = \$1`).
		WithArgs("welcome_credit:u-1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`FROM welcome_credit_settings\s+WHERE id = 1`).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "amount_yuan", "currency_code", "updated_at", "updated_by"}).
			AddRow(true, "1.00", "CNY", now, "system"))

	mock.ExpectBegin()
	expectEnsureAccountForUpdate(mock, "u-1", "0", "0", 1, now)
	mock.ExpectExec(`UPDATE billing_accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO billing_ledger_entries`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))
	mock.ExpectQuery(`INSERT INTO welcome_credit_grants`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(21))
	mock.ExpectCommit()

	account1, entry1, grant1, granted1, err := svc.GrantWelcomeCredit(context.Background(), "u-1", "welcome_credit:u-1")
	if err != nil {
		t.Fatalf("first grant failed: %v", err)
	}
	if !granted1 {
		t.Fatal("expected first grant to be granted")
	}
	if entry1 == nil || grant1 == nil {
		t.Fatal("expected first grant ledger and snapshot")
	}
	if account1.AvailableBalanceFen.String() != "100" {
		t.Fatalf("expected available balance 100 internal minor units, got %s", account1.AvailableBalanceFen.String())
	}

	expectEnsureAccount(mock, "u-1", "100", "100", 2, now)
	mock.ExpectQuery(`FROM billing_ledger_entries\s+WHERE operation_id = \$1`).
		WithArgs("welcome_credit:u-1").
		WillReturnRows(ledgerRows().
			AddRow(11, entry1.EntryNo, int64(1), "u-1", "", "", int64(0), "", "", "welcome_credit:u-1", models.LedgerEntryTypeManualTopup, models.BillingSceneOnboarding, "100", "100", "0", "100", "0", "", models.WelcomeCreditReasonCode, now))
	mock.ExpectQuery(`FROM welcome_credit_grants\s+WHERE operation_id = \$1`).
		WithArgs("welcome_credit:u-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "operation_id", "ledger_entry_no", "reason_code", "amount_yuan", "currency_code", "created_at"}).
			AddRow(21, "u-1", "welcome_credit:u-1", entry1.EntryNo, models.WelcomeCreditReasonCode, "1.00", "CNY", now))

	account2, entry2, grant2, granted2, err := svc.GrantWelcomeCredit(context.Background(), "u-1", "welcome_credit:u-1")
	if err != nil {
		t.Fatalf("idempotent grant failed: %v", err)
	}
	if granted2 {
		t.Fatal("expected second grant to be idempotent and not granted")
	}
	if entry2 == nil || grant2 == nil {
		t.Fatal("expected second grant to return existing ledger and snapshot")
	}
	if entry1.EntryNo != entry2.EntryNo {
		t.Fatalf("expected same ledger entry, got %s and %s", entry1.EntryNo, entry2.EntryNo)
	}
	if account2.AvailableBalanceFen.String() != "100" {
		t.Fatalf("expected idempotent balance unchanged at 100, got %s", account2.AvailableBalanceFen.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestGrantWelcomeCreditSnapshotsAmount(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	svc := NewBillingService(repository.NewBillingRepository(db), repository.NewWelcomeCreditSettingsRepository(db))
	now := time.Now()

	expectEnsureAccount(mock, "u-2", "0", "0", 1, now)
	mock.ExpectQuery(`FROM billing_ledger_entries\s+WHERE operation_id = \$1`).
		WithArgs("welcome_credit:u-2").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`FROM welcome_credit_settings\s+WHERE id = 1`).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "amount_yuan", "currency_code", "updated_at", "updated_by"}).
			AddRow(true, "1.50", "CNY", now, "system"))

	mock.ExpectBegin()
	expectEnsureAccountForUpdate(mock, "u-2", "0", "0", 1, now)
	mock.ExpectExec(`UPDATE billing_accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO billing_ledger_entries`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(12))
	mock.ExpectQuery(`INSERT INTO welcome_credit_grants`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(22))
	mock.ExpectCommit()

	_, entry, grant, granted, err := svc.GrantWelcomeCredit(context.Background(), "u-2", "welcome_credit:u-2")
	if err != nil {
		t.Fatalf("grant failed: %v", err)
	}
	if !granted {
		t.Fatal("expected grant to be applied")
	}
	if entry == nil || grant == nil {
		t.Fatal("expected ledger and grant snapshot")
	}
	if entry.Remark != models.WelcomeCreditReasonCode {
		t.Fatalf("expected ledger reason code %s, got %s", models.WelcomeCreditReasonCode, entry.Remark)
	}
	if entry.ActionAmountFen.String() != "150" {
		t.Fatalf("expected internal minor amount 150, got %s", entry.ActionAmountFen.String())
	}
	if grant.AmountYuan.String() != "1.5" {
		t.Fatalf("expected snapshot amount_yuan 1.5, got %s", grant.AmountYuan.String())
	}
	if grant.CurrencyCode != "CNY" {
		t.Fatalf("expected snapshot currency CNY, got %s", grant.CurrencyCode)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestGrantWelcomeCreditDisabledSetting(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	svc := NewBillingService(repository.NewBillingRepository(db), repository.NewWelcomeCreditSettingsRepository(db))
	now := time.Now()

	expectEnsureAccount(mock, "u-3", "0", "0", 1, now)
	mock.ExpectQuery(`FROM billing_ledger_entries\s+WHERE operation_id = \$1`).
		WithArgs("welcome_credit:u-3").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`FROM welcome_credit_settings\s+WHERE id = 1`).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "amount_yuan", "currency_code", "updated_at", "updated_by"}).
			AddRow(false, "1.00", "CNY", now, "system"))

	account, entry, grant, granted, err := svc.GrantWelcomeCredit(context.Background(), "u-3", "welcome_credit:u-3")
	if err != nil {
		t.Fatalf("grant with disabled setting failed: %v", err)
	}
	if granted {
		t.Fatal("expected disabled setting to skip grant")
	}
	if entry != nil || grant != nil {
		t.Fatal("expected no ledger and no grant snapshot when disabled")
	}
	if account.AvailableBalanceFen.String() != "0" {
		t.Fatalf("expected balance unchanged, got %s", account.AvailableBalanceFen.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestGrantWelcomeCreditRejectsDuplicateOperation(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	svc := NewBillingService(repository.NewBillingRepository(db), repository.NewWelcomeCreditSettingsRepository(db))
	now := time.Now()

	expectEnsureAccount(mock, "u-4", "0", "0", 1, now)
	mock.ExpectQuery(`FROM billing_ledger_entries\s+WHERE operation_id = \$1`).
		WithArgs("welcome_credit:u-4").
		WillReturnRows(ledgerRows().
			AddRow(31, "led_x", int64(1), "other-user", "", "", int64(0), "", "", "welcome_credit:u-4", models.LedgerEntryTypeManualTopup, models.BillingSceneAdmin, "100", "100", "0", "100", "0", "", "manual_topup", now))

	_, _, _, _, err = svc.GrantWelcomeCredit(context.Background(), "u-4", "welcome_credit:u-4")
	if !errors.Is(err, ErrDuplicateOperation) {
		t.Fatalf("expected ErrDuplicateOperation, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func expectEnsureAccount(mock sqlmock.Sqlmock, userID, available, totalRecharged string, version int32, now time.Time) {
	mock.ExpectExec(`INSERT INTO billing_accounts`).WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, user_id, currency_code, available_balance_fen, reserved_balance_fen`).
		WithArgs(userID).
		WillReturnRows(accountRows().AddRow(1, userID, "CNY", available, "0", totalRecharged, "0", int64(0), models.BillingAccountStatusActive, version, now, now))
}

func expectEnsureAccountForUpdate(mock sqlmock.Sqlmock, userID, available, totalRecharged string, version int32, now time.Time) {
	mock.ExpectExec(`INSERT INTO billing_accounts`).WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, user_id, currency_code, available_balance_fen, reserved_balance_fen[\s\S]*FOR UPDATE`).
		WithArgs(userID).
		WillReturnRows(accountRows().AddRow(1, userID, "CNY", available, "0", totalRecharged, "0", int64(0), models.BillingAccountStatusActive, version, now, now))
}

func accountRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "user_id", "currency_code", "available_balance_fen", "reserved_balance_fen", "total_recharged_fen", "total_spent_fen", "total_traffic_bytes", "status", "version", "created_at", "updated_at"})
}

func ledgerRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "entry_no", "account_id", "user_id", "order_no", "hold_no", "history_id", "task_id", "transfer_id", "operation_id", "entry_type", "scene", "action_amount_fen", "available_delta_fen", "reserved_delta_fen", "balance_after_available_fen", "balance_after_reserved_fen", "operator_user_id", "remark", "created_at"})
}

func TestGrantWelcomeCreditBootstrapsDefaultSetting(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	svc := NewBillingService(repository.NewBillingRepository(db), repository.NewWelcomeCreditSettingsRepository(db))
	now := time.Now()

	expectEnsureAccount(mock, "u-4", "0", "0", 1, now)
	mock.ExpectQuery(`FROM billing_ledger_entries\s+WHERE operation_id = \$1`).
		WithArgs("welcome_credit:u-4").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`FROM welcome_credit_settings\s+WHERE id = 1`).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO welcome_credit_settings`).
		WithArgs(true, defaultWelcomeCreditSettings.AmountYuan, defaultWelcomeCreditSettings.CurrencyCode, defaultWelcomeCreditSettings.UpdatedBy).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "amount_yuan", "currency_code", "updated_at", "updated_by"}).
			AddRow(true, "1.00", "CNY", now, "system"))

	mock.ExpectBegin()
	expectEnsureAccountForUpdate(mock, "u-4", "0", "0", 1, now)
	mock.ExpectExec(`UPDATE billing_accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO billing_ledger_entries`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(13))
	mock.ExpectQuery(`INSERT INTO welcome_credit_grants`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(23))
	mock.ExpectCommit()

	account, entry, grant, granted, err := svc.GrantWelcomeCredit(context.Background(), "u-4", "welcome_credit:u-4")
	if err != nil {
		t.Fatalf("grant with missing setting failed: %v", err)
	}
	if !granted {
		t.Fatal("expected missing setting to bootstrap default grant")
	}
	if entry == nil || grant == nil {
		t.Fatal("expected ledger and grant snapshot when default setting is bootstrapped")
	}
	if account.AvailableBalanceFen.String() != "100" {
		t.Fatalf("expected bootstrapped default balance 100, got %s", account.AvailableBalanceFen.String())
	}
	if grant.AmountYuan.String() != "1" {
		t.Fatalf("expected default snapshot amount 1, got %s", grant.AmountYuan.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestGrantWelcomeCreditBootstrapsUninitializedSetting(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	svc := NewBillingService(repository.NewBillingRepository(db), repository.NewWelcomeCreditSettingsRepository(db))
	now := time.Now()

	expectEnsureAccount(mock, "u-5", "0", "0", 1, now)
	mock.ExpectQuery(`FROM billing_ledger_entries\s+WHERE operation_id = \$1`).
		WithArgs("welcome_credit:u-5").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`FROM welcome_credit_settings\s+WHERE id = 1`).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "amount_yuan", "currency_code", "updated_at", "updated_by"}).
			AddRow(true, "0", "", time.Time{}, ""))
	mock.ExpectQuery(`INSERT INTO welcome_credit_settings`).
		WithArgs(true, defaultWelcomeCreditSettings.AmountYuan, defaultWelcomeCreditSettings.CurrencyCode, defaultWelcomeCreditSettings.UpdatedBy).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "amount_yuan", "currency_code", "updated_at", "updated_by"}).
			AddRow(true, "1.00", "CNY", now, "system"))

	mock.ExpectBegin()
	expectEnsureAccountForUpdate(mock, "u-5", "0", "0", 1, now)
	mock.ExpectExec(`UPDATE billing_accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO billing_ledger_entries`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))
	mock.ExpectQuery(`INSERT INTO welcome_credit_grants`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(24))
	mock.ExpectCommit()

	account, entry, grant, granted, err := svc.GrantWelcomeCredit(context.Background(), "u-5", "welcome_credit:u-5")
	if err != nil {
		t.Fatalf("grant with uninitialized setting failed: %v", err)
	}
	if !granted {
		t.Fatal("expected uninitialized setting to bootstrap default grant")
	}
	if entry == nil || grant == nil {
		t.Fatal("expected ledger and grant snapshot when uninitialized setting is bootstrapped")
	}
	if account.AvailableBalanceFen.String() != "100" {
		t.Fatalf("expected bootstrapped default balance 100, got %s", account.AvailableBalanceFen.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
