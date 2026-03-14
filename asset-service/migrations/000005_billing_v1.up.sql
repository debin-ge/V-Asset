CREATE TABLE IF NOT EXISTS billing_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL UNIQUE,
    currency_code CHAR(3) NOT NULL DEFAULT 'CNY',
    available_balance_fen BIGINT NOT NULL DEFAULT 0,
    reserved_balance_fen BIGINT NOT NULL DEFAULT 0,
    total_recharged_fen BIGINT NOT NULL DEFAULT 0,
    total_spent_fen BIGINT NOT NULL DEFAULT 0,
    total_traffic_bytes BIGINT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 1,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS billing_pricing (
    id BIGSERIAL PRIMARY KEY,
    version INT NOT NULL UNIQUE,
    ingress_price_fen_per_gib NUMERIC(20,8) NOT NULL,
    egress_price_fen_per_gib NUMERIC(20,8) NOT NULL,
    default_estimate_bytes BIGINT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    remark VARCHAR(255) NOT NULL DEFAULT '',
    updated_by_user_id VARCHAR(64) NOT NULL,
    effective_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_pricing_enabled_unique
    ON billing_pricing(enabled)
    WHERE enabled = TRUE;

CREATE TABLE IF NOT EXISTS billing_charge_orders (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(64) NOT NULL,
    history_id BIGINT NOT NULL,
    task_id VARCHAR(64) NOT NULL DEFAULT '',
    scene SMALLINT NOT NULL,
    status SMALLINT NOT NULL,
    pricing_version INT NOT NULL,
    estimated_ingress_bytes BIGINT NOT NULL DEFAULT 0,
    estimated_egress_bytes BIGINT NOT NULL DEFAULT 0,
    estimated_traffic_bytes BIGINT NOT NULL DEFAULT 0,
    actual_ingress_bytes BIGINT NOT NULL DEFAULT 0,
    actual_egress_bytes BIGINT NOT NULL DEFAULT 0,
    actual_traffic_bytes BIGINT NOT NULL DEFAULT 0,
    held_amount_fen BIGINT NOT NULL DEFAULT 0,
    captured_amount_fen BIGINT NOT NULL DEFAULT 0,
    released_amount_fen BIGINT NOT NULL DEFAULT 0,
    shortfall_fen BIGINT NOT NULL DEFAULT 0,
    remark VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_charge_orders_task_id_unique
    ON billing_charge_orders(task_id)
    WHERE task_id <> '';

CREATE INDEX IF NOT EXISTS idx_billing_charge_orders_user_created_at
    ON billing_charge_orders(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_charge_orders_history_scene
    ON billing_charge_orders(history_id, scene, created_at DESC);

CREATE TABLE IF NOT EXISTS billing_holds (
    id BIGSERIAL PRIMARY KEY,
    hold_no VARCHAR(64) NOT NULL UNIQUE,
    order_no VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    history_id BIGINT NOT NULL,
    task_id VARCHAR(64) NOT NULL DEFAULT '',
    transfer_id VARCHAR(64) NOT NULL DEFAULT '',
    hold_type SMALLINT NOT NULL,
    funding_source SMALLINT NOT NULL,
    status SMALLINT NOT NULL,
    amount_fen BIGINT NOT NULL DEFAULT 0,
    captured_amount_fen BIGINT NOT NULL DEFAULT 0,
    released_amount_fen BIGINT NOT NULL DEFAULT 0,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_holds_task_id_download_unique
    ON billing_holds(task_id)
    WHERE hold_type = 1 AND task_id <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_holds_transfer_id_unique
    ON billing_holds(transfer_id)
    WHERE transfer_id <> '';

CREATE INDEX IF NOT EXISTS idx_billing_holds_order_no
    ON billing_holds(order_no);

CREATE TABLE IF NOT EXISTS traffic_usage_records (
    id BIGSERIAL PRIMARY KEY,
    usage_no VARCHAR(64) NOT NULL UNIQUE,
    order_no VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    history_id BIGINT NOT NULL,
    task_id VARCHAR(64) NOT NULL DEFAULT '',
    transfer_id VARCHAR(64) NOT NULL DEFAULT '',
    direction SMALLINT NOT NULL,
    traffic_bytes BIGINT NOT NULL,
    unit_price_fen_per_gib NUMERIC(20,8) NOT NULL,
    amount_fen BIGINT NOT NULL,
    pricing_version INT NOT NULL,
    source_service VARCHAR(32) NOT NULL,
    status SMALLINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_traffic_usage_records_user_created_at
    ON traffic_usage_records(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS billing_ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    entry_no VARCHAR(64) NOT NULL UNIQUE,
    account_id BIGINT NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    order_no VARCHAR(64) NOT NULL DEFAULT '',
    hold_no VARCHAR(64) NOT NULL DEFAULT '',
    history_id BIGINT NOT NULL DEFAULT 0,
    task_id VARCHAR(64) NOT NULL DEFAULT '',
    transfer_id VARCHAR(64) NOT NULL DEFAULT '',
    operation_id VARCHAR(64) NOT NULL DEFAULT '',
    entry_type SMALLINT NOT NULL,
    scene SMALLINT NOT NULL,
    action_amount_fen BIGINT NOT NULL DEFAULT 0,
    available_delta_fen BIGINT NOT NULL DEFAULT 0,
    reserved_delta_fen BIGINT NOT NULL DEFAULT 0,
    balance_after_available_fen BIGINT NOT NULL,
    balance_after_reserved_fen BIGINT NOT NULL,
    operator_user_id VARCHAR(64) NOT NULL DEFAULT '',
    remark VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_ledger_entries_operation_id_unique
    ON billing_ledger_entries(operation_id)
    WHERE operation_id <> '';

CREATE INDEX IF NOT EXISTS idx_billing_ledger_entries_user_time
    ON billing_ledger_entries(user_id, created_at DESC);

INSERT INTO billing_pricing (
    version,
    ingress_price_fen_per_gib,
    egress_price_fen_per_gib,
    default_estimate_bytes,
    enabled,
    remark,
    updated_by_user_id
)
SELECT
    1,
    0.00000000,
    0.00000000,
    104857600,
    TRUE,
    'bootstrap pricing',
    'system'
WHERE NOT EXISTS (
    SELECT 1 FROM billing_pricing WHERE version = 1
);
