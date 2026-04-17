#!/bin/bash
# One-shot cutover helper for the Phase 3 auth DB split.
#
# Env:
#   SOURCE_DATABASE_URL  shared cluster URL (default: localhost:5432/ecmarket_dev)
#   AUTH_DATABASE_URL    split cluster URL  (default: localhost:5437/auth_dev)

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${AUTH_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5437/auth_dev?sslmode=disable}"

dump_file="$(mktemp -t auth_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has auth_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'auth_svc';" \
  | grep -q 1 || { echo "no auth_svc on source — nothing to backfill"; exit 1; }

echo "==> checking target cluster has auth_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'auth_svc';" \
  | grep -q 1 || { echo "no auth_svc on target — run migrations first"; exit 1; }

echo "==> dumping auth_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=auth_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target auth_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    auth_svc.rbac_audit_log,
    auth_svc.seller_api_tokens,
    auth_svc.seller_users,
    auth_svc.buyers,
    auth_svc.sellers
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'sellers'           AS table, COUNT(*) FROM auth_svc.sellers
UNION ALL SELECT 'seller_users',      COUNT(*) FROM auth_svc.seller_users
UNION ALL SELECT 'seller_api_tokens', COUNT(*) FROM auth_svc.seller_api_tokens
UNION ALL SELECT 'buyers',            COUNT(*) FROM auth_svc.buyers;
"

echo "==> done."
