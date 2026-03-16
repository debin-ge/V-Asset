ALTER TABLE billing_pricing
    ADD COLUMN IF NOT EXISTS default_estimate_bytes BIGINT NOT NULL DEFAULT 0;

UPDATE billing_pricing
SET default_estimate_bytes = 0
WHERE default_estimate_bytes IS NULL;

ALTER TABLE billing_ledger_entries
    ALTER COLUMN action_amount_fen TYPE BIGINT USING ROUND(action_amount_fen * 100),
    ALTER COLUMN available_delta_fen TYPE BIGINT USING ROUND(available_delta_fen * 100),
    ALTER COLUMN reserved_delta_fen TYPE BIGINT USING ROUND(reserved_delta_fen * 100),
    ALTER COLUMN balance_after_available_fen TYPE BIGINT USING ROUND(balance_after_available_fen * 100),
    ALTER COLUMN balance_after_reserved_fen TYPE BIGINT USING ROUND(balance_after_reserved_fen * 100);

ALTER TABLE traffic_usage_records
    ALTER COLUMN amount_fen TYPE BIGINT USING ROUND(amount_fen * 100);

ALTER TABLE billing_holds
    ALTER COLUMN amount_fen TYPE BIGINT USING ROUND(amount_fen * 100),
    ALTER COLUMN captured_amount_fen TYPE BIGINT USING ROUND(captured_amount_fen * 100),
    ALTER COLUMN released_amount_fen TYPE BIGINT USING ROUND(released_amount_fen * 100);

ALTER TABLE billing_charge_orders
    ALTER COLUMN held_amount_fen TYPE BIGINT USING ROUND(held_amount_fen * 100),
    ALTER COLUMN captured_amount_fen TYPE BIGINT USING ROUND(captured_amount_fen * 100),
    ALTER COLUMN released_amount_fen TYPE BIGINT USING ROUND(released_amount_fen * 100),
    ALTER COLUMN shortfall_fen TYPE BIGINT USING ROUND(shortfall_fen * 100);

ALTER TABLE billing_accounts
    ALTER COLUMN available_balance_fen TYPE BIGINT USING ROUND(available_balance_fen * 100),
    ALTER COLUMN reserved_balance_fen TYPE BIGINT USING ROUND(reserved_balance_fen * 100),
    ALTER COLUMN total_recharged_fen TYPE BIGINT USING ROUND(total_recharged_fen * 100),
    ALTER COLUMN total_spent_fen TYPE BIGINT USING ROUND(total_spent_fen * 100);
