package repository

import (
	"context"
	"database/sql"

	"vasset/asset-service/internal/models"
)

type WelcomeCreditSettingsRepository struct {
	db *sql.DB
}

func NewWelcomeCreditSettingsRepository(db *sql.DB) *WelcomeCreditSettingsRepository {
	return &WelcomeCreditSettingsRepository{db: db}
}

func (r *WelcomeCreditSettingsRepository) GetWelcomeCreditSettings(ctx context.Context) (*models.WelcomeCreditSettings, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT enabled, amount_yuan, currency_code, updated_at, updated_by
		FROM welcome_credit_settings
		WHERE id = 1
	`)

	var settings models.WelcomeCreditSettings
	if err := row.Scan(
		&settings.Enabled,
		&settings.AmountYuan,
		&settings.CurrencyCode,
		&settings.UpdatedAt,
		&settings.UpdatedBy,
	); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (r *WelcomeCreditSettingsRepository) UpsertWelcomeCreditSettings(ctx context.Context, settings *models.WelcomeCreditSettings) (*models.WelcomeCreditSettings, error) {
	row := r.db.QueryRowContext(ctx, `
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
	`, settings.Enabled, settings.AmountYuan, settings.CurrencyCode, settings.UpdatedBy)

	var stored models.WelcomeCreditSettings
	if err := row.Scan(
		&stored.Enabled,
		&stored.AmountYuan,
		&stored.CurrencyCode,
		&stored.UpdatedAt,
		&stored.UpdatedBy,
	); err != nil {
		return nil, err
	}

	return &stored, nil
}
