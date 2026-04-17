#!/bin/bash
# One-shot cutover helper for the Phase 3 inquiry DB split.
#
# Env:
#   SOURCE_DATABASE_URL   shared cluster URL (default: localhost:5432/ecmarket_dev)
#   INQUIRY_DATABASE_URL  split cluster URL  (default: localhost:5438/inquiry_dev)

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${INQUIRY_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5438/inquiry_dev?sslmode=disable}"

dump_file="$(mktemp -t inquiry_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has inquiry_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'inquiry_svc';" \
  | grep -q 1 || { echo "no inquiry_svc on source — nothing to backfill"; exit 1; }

echo "==> checking target cluster has inquiry_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'inquiry_svc';" \
  | grep -q 1 || { echo "no inquiry_svc on target — run migrations first"; exit 1; }

echo "==> dumping inquiry_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=inquiry_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target inquiry_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    inquiry_svc.inquiry_messages,
    inquiry_svc.inquiries
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'inquiries'        AS table, COUNT(*) FROM inquiry_svc.inquiries
UNION ALL SELECT 'inquiry_messages', COUNT(*) FROM inquiry_svc.inquiry_messages;
"

echo "==> done."
