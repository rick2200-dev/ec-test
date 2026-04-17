#!/bin/bash
# One-shot cutover helper for the Phase 3 order DB split.
#
# IMPORTANT: stop the order service before running. The checkout flow
# writes orders + order_lines + payouts transactionally; the
# truncate+load window would race with live writes.
#
# Env:
#   SOURCE_DATABASE_URL  shared cluster URL (default: localhost:5432/ecmarket_dev)
#   ORDER_DATABASE_URL   split cluster URL  (default: localhost:5443/order_dev)

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${ORDER_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5443/order_dev?sslmode=disable}"

dump_file="$(mktemp -t order_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has order_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'order_svc';" \
  | grep -q 1 || { echo "no order_svc on source — nothing to backfill"; exit 1; }

echo "==> checking target cluster has order_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'order_svc';" \
  | grep -q 1 || { echo "no order_svc on target — run migrations first"; exit 1; }

echo "==> dumping order_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=order_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target order_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    order_svc.order_cancellation_requests,
    order_svc.payouts,
    order_svc.order_lines,
    order_svc.commission_rules,
    order_svc.orders
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'orders'                       AS table, COUNT(*) FROM order_svc.orders
UNION ALL SELECT 'order_lines',                 COUNT(*) FROM order_svc.order_lines
UNION ALL SELECT 'payouts',                     COUNT(*) FROM order_svc.payouts
UNION ALL SELECT 'commission_rules',            COUNT(*) FROM order_svc.commission_rules
UNION ALL SELECT 'cancellation_requests',       COUNT(*) FROM order_svc.order_cancellation_requests;
"

echo "==> done."
