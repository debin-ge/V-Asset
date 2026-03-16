ALTER TABLE billing_accounts
    ALTER COLUMN available_balance_fen TYPE NUMERIC(20,8) USING available_balance_fen::NUMERIC / 100,
    ALTER COLUMN reserved_balance_fen TYPE NUMERIC(20,8) USING reserved_balance_fen::NUMERIC / 100,
    ALTER COLUMN total_recharged_fen TYPE NUMERIC(20,8) USING total_recharged_fen::NUMERIC / 100,
    ALTER COLUMN total_spent_fen TYPE NUMERIC(20,8) USING total_spent_fen::NUMERIC / 100;

ALTER TABLE billing_charge_orders
    ALTER COLUMN held_amount_fen TYPE NUMERIC(20,8) USING held_amount_fen::NUMERIC / 100,
    ALTER COLUMN captured_amount_fen TYPE NUMERIC(20,8) USING captured_amount_fen::NUMERIC / 100,
    ALTER COLUMN released_amount_fen TYPE NUMERIC(20,8) USING released_amount_fen::NUMERIC / 100,
    ALTER COLUMN shortfall_fen TYPE NUMERIC(20,8) USING shortfall_fen::NUMERIC / 100;

ALTER TABLE billing_holds
    ALTER COLUMN amount_fen TYPE NUMERIC(20,8) USING amount_fen::NUMERIC / 100,
    ALTER COLUMN captured_amount_fen TYPE NUMERIC(20,8) USING captured_amount_fen::NUMERIC / 100,
    ALTER COLUMN released_amount_fen TYPE NUMERIC(20,8) USING released_amount_fen::NUMERIC / 100;

ALTER TABLE traffic_usage_records
    ALTER COLUMN amount_fen TYPE NUMERIC(20,8) USING amount_fen::NUMERIC / 100;

ALTER TABLE billing_ledger_entries
    ALTER COLUMN action_amount_fen TYPE NUMERIC(20,8) USING action_amount_fen::NUMERIC / 100,
    ALTER COLUMN available_delta_fen TYPE NUMERIC(20,8) USING available_delta_fen::NUMERIC / 100,
    ALTER COLUMN reserved_delta_fen TYPE NUMERIC(20,8) USING reserved_delta_fen::NUMERIC / 100,
    ALTER COLUMN balance_after_available_fen TYPE NUMERIC(20,8) USING balance_after_available_fen::NUMERIC / 100,
    ALTER COLUMN balance_after_reserved_fen TYPE NUMERIC(20,8) USING balance_after_reserved_fen::NUMERIC / 100;

ALTER TABLE billing_pricing
    DROP COLUMN IF EXISTS default_estimate_bytes;
