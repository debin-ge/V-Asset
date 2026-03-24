ALTER TABLE billing_accounts
    ALTER COLUMN available_balance_yuan TYPE NUMERIC(20,8) USING available_balance_yuan::NUMERIC / 100,
    ALTER COLUMN reserved_balance_yuan TYPE NUMERIC(20,8) USING reserved_balance_yuan::NUMERIC / 100,
    ALTER COLUMN total_recharged_yuan TYPE NUMERIC(20,8) USING total_recharged_yuan::NUMERIC / 100,
    ALTER COLUMN total_spent_yuan TYPE NUMERIC(20,8) USING total_spent_yuan::NUMERIC / 100;

ALTER TABLE billing_charge_orders
    ALTER COLUMN held_amount_yuan TYPE NUMERIC(20,8) USING held_amount_yuan::NUMERIC / 100,
    ALTER COLUMN captured_amount_yuan TYPE NUMERIC(20,8) USING captured_amount_yuan::NUMERIC / 100,
    ALTER COLUMN released_amount_yuan TYPE NUMERIC(20,8) USING released_amount_yuan::NUMERIC / 100,
    ALTER COLUMN shortfall_yuan TYPE NUMERIC(20,8) USING shortfall_yuan::NUMERIC / 100;

ALTER TABLE billing_holds
    ALTER COLUMN amount_yuan TYPE NUMERIC(20,8) USING amount_yuan::NUMERIC / 100,
    ALTER COLUMN captured_amount_yuan TYPE NUMERIC(20,8) USING captured_amount_yuan::NUMERIC / 100,
    ALTER COLUMN released_amount_yuan TYPE NUMERIC(20,8) USING released_amount_yuan::NUMERIC / 100;

ALTER TABLE traffic_usage_records
    ALTER COLUMN amount_yuan TYPE NUMERIC(20,8) USING amount_yuan::NUMERIC / 100;

ALTER TABLE billing_ledger_entries
    ALTER COLUMN action_amount_yuan TYPE NUMERIC(20,8) USING action_amount_yuan::NUMERIC / 100,
    ALTER COLUMN available_delta_yuan TYPE NUMERIC(20,8) USING available_delta_yuan::NUMERIC / 100,
    ALTER COLUMN reserved_delta_yuan TYPE NUMERIC(20,8) USING reserved_delta_yuan::NUMERIC / 100,
    ALTER COLUMN balance_after_available_yuan TYPE NUMERIC(20,8) USING balance_after_available_yuan::NUMERIC / 100,
    ALTER COLUMN balance_after_reserved_yuan TYPE NUMERIC(20,8) USING balance_after_reserved_yuan::NUMERIC / 100;

ALTER TABLE billing_pricing
    DROP COLUMN IF EXISTS default_estimate_bytes;
