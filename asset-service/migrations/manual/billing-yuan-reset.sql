-- Non-production only.
-- Reset billing data for the yuan + yuan/GB model and seed one active pricing row.

BEGIN;

TRUNCATE TABLE
  traffic_usage_records,
  billing_ledger_entries,
  billing_holds,
  billing_charge_orders,
  welcome_credit_grants,
  billing_accounts,
  billing_pricing
RESTART IDENTITY CASCADE;

INSERT INTO billing_pricing (
  version,
  ingress_price_yuan_per_gb,
  egress_price_yuan_per_gb,
  enabled,
  remark,
  updated_by_user_id,
  effective_at
) VALUES (
  1,
  0.10,
  0.10,
  TRUE,
  'bootstrap pricing (yuan/GB)',
  'system',
  NOW()
);

COMMIT;
