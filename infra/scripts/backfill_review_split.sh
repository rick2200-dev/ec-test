#!/bin/bash
# One-shot cutover helper for the Phase 3 review DB split.
#
# Env:
#   SOURCE_DATABASE_URL  shared cluster URL (default: localhost:5432/ecmarket_dev)
#   REVIEW_DATABASE_URL  split cluster URL  (default: localhost:5439/review_dev)

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${REVIEW_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5439/review_dev?sslmode=disable}"

dump_file="$(mktemp -t review_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has review_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'review_svc';" \
  | grep -q 1 || { echo "no review_svc on source — nothing to backfill"; exit 1; }

echo "==> checking target cluster has review_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'review_svc';" \
  | grep -q 1 || { echo "no review_svc on target — run migrations first"; exit 1; }

echo "==> dumping review_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=review_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target review_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    review_svc.review_replies,
    review_svc.product_ratings,
    review_svc.reviews
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'reviews'          AS table, COUNT(*) FROM review_svc.reviews
UNION ALL SELECT 'review_replies',   COUNT(*) FROM review_svc.review_replies
UNION ALL SELECT 'product_ratings',  COUNT(*) FROM review_svc.product_ratings;
"

echo "==> done."
