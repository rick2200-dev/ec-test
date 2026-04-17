#!/bin/bash
# One-shot cutover helper for the Phase 3 search DB split.
#
# Env:
#   SOURCE_DATABASE_URL  shared cluster URL (default: localhost:5432/ecmarket_dev)
#   SEARCH_DATABASE_URL  split cluster URL  (default: localhost:5440/search_dev)

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${SEARCH_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5440/search_dev?sslmode=disable}"

dump_file="$(mktemp -t search_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has search_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'search_svc';" \
  | grep -q 1 || { echo "no search_svc on source — nothing to backfill"; exit 1; }

echo "==> checking target cluster has search_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'search_svc';" \
  | grep -q 1 || { echo "no search_svc on target — run migrations first"; exit 1; }

echo "==> dumping search_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=search_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target search_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    search_svc.products,
    search_svc.seller_plan_boost
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'seller_plan_boost' AS table, COUNT(*) FROM search_svc.seller_plan_boost
UNION ALL SELECT 'products',        COUNT(*) FROM search_svc.products;
"

echo "==> done."
