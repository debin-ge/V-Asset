CREATE TABLE IF NOT EXISTS welcome_credit_grants (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    operation_id VARCHAR(64) NOT NULL UNIQUE,
    ledger_entry_no VARCHAR(64) NOT NULL,
    reason_code VARCHAR(32) NOT NULL,
    amount_yuan NUMERIC(20,8) NOT NULL,
    currency_code CHAR(3) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_welcome_credit_grants_user_created_at
    ON welcome_credit_grants(user_id, created_at DESC);
