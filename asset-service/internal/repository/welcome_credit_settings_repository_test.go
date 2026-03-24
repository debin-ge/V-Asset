package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/money"
)

func TestWelcomeCreditSettingsRepositoryRoundTrip(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	repo := NewWelcomeCreditSettingsRepository(db)
	now := time.Now()

	amountYuan := money.MustParse("1.25")
	upsertQuery := regexp.QuoteMeta(`
		INSERT INTO welcome_credit_settings (
			id, enabled, amount_yuan, currency_code, updated_at, updated_by
		) VALUES (
			1, $1, $2, $3, CURRENT_TIMESTAMP, $4
		)
		ON CONFLICT (id)
		DO UPDATE SET
			enabled = EXCLUDED.enabled,
			amount_yuan = EXCLUDED.amount_yuan,
			currency_code = EXCLUDED.currency_code,
			updated_at = CURRENT_TIMESTAMP,
			updated_by = EXCLUDED.updated_by
		RETURNING enabled, amount_yuan, currency_code, updated_at, updated_by
	`)

	mock.ExpectQuery(upsertQuery).
		WithArgs(true, amountYuan, "CNY", "tester").
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "amount_yuan", "currency_code", "updated_at", "updated_by"}).
			AddRow(true, "1.25", "CNY", now, "tester"))

	stored, err := repo.UpsertWelcomeCreditSettings(context.Background(), &models.WelcomeCreditSettings{
		Enabled:      true,
		AmountYuan:   amountYuan,
		CurrencyCode: "CNY",
		UpdatedBy:    "tester",
	})
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if stored.AmountYuan.String() != "1.25" {
		t.Fatalf("expected stored amount 1.25, got %s", stored.AmountYuan.String())
	}

	getQuery := regexp.QuoteMeta(`
		SELECT enabled, amount_yuan, currency_code, updated_at, updated_by
		FROM welcome_credit_settings
		WHERE id = 1
	`)

	mock.ExpectQuery(getQuery).
		WillReturnRows(sqlmock.NewRows([]string{"enabled", "amount_yuan", "currency_code", "updated_at", "updated_by"}).
			AddRow(true, "1.25", "CNY", now, "tester"))

	fetched, err := repo.GetWelcomeCreditSettings(context.Background())
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if fetched.AmountYuan.String() != "1.25" {
		t.Fatalf("expected fetched amount 1.25, got %s", fetched.AmountYuan.String())
	}
	if fetched.UpdatedBy != "tester" {
		t.Fatalf("expected updated_by tester, got %s", fetched.UpdatedBy)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
