# Billing Reset (Yuan / GB)

This runbook is **non-production only** and is intended for the one-time reset after the billing unit migration to `yuan` and `yuan / GB`.

## 1) Stop related services

From repo root:

```bash
docker compose stop api-gateway asset-service media-service admin-service frontend-service admin-frontend
```

## 2) (If upgrading an old DB) rename legacy columns

If your current database still has legacy billing columns such as `*_fen` / `*_per_gib`, run:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f asset-service/migrations/manual/billing-fen-to-yuan-schema.sql
```

If you previously saw `current transaction is aborted`, run `ROLLBACK;` first in that psql session (or open a new session), then execute the command above.

## 3) Execute reset SQL

Use your asset-service database connection and run:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f asset-service/migrations/manual/billing-yuan-reset.sql
```

The SQL does the following:
- Truncates billing data tables and resets sequences.
- Inserts one bootstrap pricing row using `yuan / GB`.

## 4) Start services

```bash
docker compose up -d api-gateway asset-service media-service admin-service frontend-service admin-frontend
```

## 5) Health checks

Recommended checks:
- `GET /api/v1/user/account` returns `*_yuan` fields.
- `GET /api/v1/admin/billing/pricing` returns `ingress_price_yuan_per_gb` and `egress_price_yuan_per_gb`.
- Submit one download and verify:
  - estimate uses `estimated_cost_yuan`;
  - complete flow updates `total_spent_yuan` and creates usage + ledger records.
