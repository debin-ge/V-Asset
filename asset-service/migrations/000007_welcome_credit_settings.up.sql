CREATE TABLE IF NOT EXISTS welcome_credit_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled BOOLEAN NOT NULL,
    amount_yuan NUMERIC(20,8) NOT NULL,
    currency_code CHAR(3) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64) NOT NULL
);

INSERT INTO welcome_credit_settings (
    id,
    enabled,
    amount_yuan,
    currency_code,
    updated_by
)
VALUES (
    1,
    TRUE,
    1.00,
    'CNY',
    'system'
)
ON CONFLICT (id) DO NOTHING;
