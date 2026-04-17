#!/bin/bash
# One-shot cutover helper for the Phase 3 recommend DB split.
#
# Env:
#   SOURCE_DATABASE_URL    shared cluster URL (default: localhost:5432/ecmarket_dev)
#   RECOMMEND_DATABASE_URL split cluster URL  (default: localhost:5441/recommend_dev)

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${RECOMMEND_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5441/recommend_dev?sslmode=disable}"

dump_file="$(mktemp -t recommend_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has recommend_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'recommend_svc';" \
  | grep -q 1 || { echo "no recommend_svc on source — nothing to backfill"; exit 1; }

echo "==> checking target cluster has recommend_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'recommend_svc';" \
  | grep -q 1 || { echo "no recommend_svc on target — run migrations first"; exit 1; }

echo "==> dumping recommend_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=recommend_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target recommend_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    recommend_svc.product_categories,
    recommend_svc.products,
    recommend_svc.user_events
RESTART IDENTITY CASCADE;
SQL

echo "==> refreshing popular_products matview"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -c "REFRESH MATERIALIZED VIEW recommend_svc.popular_products;"

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> refreshing popular_products matview after load"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -c "REFRESH MATERIALIZED VIEW recommend_svc.popular_products;"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'user_events'         AS table, COUNT(*) FROM recommend_svc.user_events
UNION ALL SELECT 'products',           COUNT(*) FROM recommend_svc.products
UNION ALL SELECT 'product_categories', COUNT(*) FROM recommend_svc.product_categories
UNION ALL SELECT 'popular_products',   COUNT(*) FROM recommend_svc.popular_products;
"

echo "==> done."
