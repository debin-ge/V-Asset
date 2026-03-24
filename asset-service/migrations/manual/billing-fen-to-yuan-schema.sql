-- Non-production helper.
-- Upgrade existing billing schema from legacy *_fen / *_per_gib columns
-- to *_yuan / *_per_gb columns expected by current services.

BEGIN;

DO $$
BEGIN
  -- billing_accounts
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_accounts' AND column_name='available_balance_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_accounts RENAME COLUMN available_balance_fen TO available_balance_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_accounts' AND column_name='reserved_balance_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_accounts RENAME COLUMN reserved_balance_fen TO reserved_balance_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_accounts' AND column_name='total_recharged_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_accounts RENAME COLUMN total_recharged_fen TO total_recharged_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_accounts' AND column_name='total_spent_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_accounts RENAME COLUMN total_spent_fen TO total_spent_yuan';
  END IF;

  -- billing_pricing
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_pricing' AND column_name='ingress_price_fen_per_gib'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_pricing RENAME COLUMN ingress_price_fen_per_gib TO ingress_price_yuan_per_gb';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_pricing' AND column_name='egress_price_fen_per_gib'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_pricing RENAME COLUMN egress_price_fen_per_gib TO egress_price_yuan_per_gb';
  END IF;

  -- billing_charge_orders
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_charge_orders' AND column_name='held_amount_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_charge_orders RENAME COLUMN held_amount_fen TO held_amount_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_charge_orders' AND column_name='captured_amount_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_charge_orders RENAME COLUMN captured_amount_fen TO captured_amount_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_charge_orders' AND column_name='released_amount_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_charge_orders RENAME COLUMN released_amount_fen TO released_amount_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_charge_orders' AND column_name='shortfall_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_charge_orders RENAME COLUMN shortfall_fen TO shortfall_yuan';
  END IF;

  -- billing_holds
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_holds' AND column_name='amount_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_holds RENAME COLUMN amount_fen TO amount_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_holds' AND column_name='captured_amount_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_holds RENAME COLUMN captured_amount_fen TO captured_amount_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_holds' AND column_name='released_amount_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_holds RENAME COLUMN released_amount_fen TO released_amount_yuan';
  END IF;

  -- traffic_usage_records
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='traffic_usage_records' AND column_name='unit_price_fen_per_gib'
  ) THEN
    EXECUTE 'ALTER TABLE public.traffic_usage_records RENAME COLUMN unit_price_fen_per_gib TO unit_price_yuan_per_gb';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='traffic_usage_records' AND column_name='amount_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.traffic_usage_records RENAME COLUMN amount_fen TO amount_yuan';
  END IF;

  -- billing_ledger_entries
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_ledger_entries' AND column_name='action_amount_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_ledger_entries RENAME COLUMN action_amount_fen TO action_amount_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_ledger_entries' AND column_name='available_delta_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_ledger_entries RENAME COLUMN available_delta_fen TO available_delta_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_ledger_entries' AND column_name='reserved_delta_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_ledger_entries RENAME COLUMN reserved_delta_fen TO reserved_delta_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_ledger_entries' AND column_name='balance_after_available_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_ledger_entries RENAME COLUMN balance_after_available_fen TO balance_after_available_yuan';
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='billing_ledger_entries' AND column_name='balance_after_reserved_fen'
  ) THEN
    EXECUTE 'ALTER TABLE public.billing_ledger_entries RENAME COLUMN balance_after_reserved_fen TO balance_after_reserved_yuan';
  END IF;
END $$;

COMMIT;
