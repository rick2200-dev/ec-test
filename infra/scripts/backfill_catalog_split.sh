#!/bin/bash
# One-shot cutover helper for the Phase 3 catalog DB split.
#
# Env:
#   SOURCE_DATABASE_URL   shared cluster URL (default: localhost:5432/ecmarket_dev)
#   CATALOG_DATABASE_URL  split cluster URL  (default: localhost:5442/catalog_dev)

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${CATALOG_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5442/catalog_dev?sslmode=disable}"

dump_file="$(mktemp -t catalog_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has catalog_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc';" \
  | grep -q 1 || { echo "no catalog_svc on source — nothing to backfill"; exit 1; }

echo "==> checking target cluster has catalog_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc';" \
  | grep -q 1 || { echo "no catalog_svc on target — run migrations first"; exit 1; }

echo "==> dumping catalog_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=catalog_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target catalog_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    catalog_svc.outbox_events,
    catalog_svc.user_events,
    catalog_svc.product_categories,
    catalog_svc.skus,
    catalog_svc.products,
    catalog_svc.categories
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'categories'          AS table, COUNT(*) FROM catalog_svc.categories
UNION ALL SELECT 'products',           COUNT(*) FROM catalog_svc.products
UNION ALL SELECT 'skus',               COUNT(*) FROM catalog_svc.skus
UNION ALL SELECT 'product_categories', COUNT(*) FROM catalog_svc.product_categories
UNION ALL SELECT 'outbox_events',      COUNT(*) FROM catalog_svc.outbox_events;
"

echo "==> done."
