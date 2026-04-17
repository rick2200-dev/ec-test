#!/bin/bash
# One-shot cutover helper for the Phase 3 inventory DB split.
#
# Copies every row in inventory_svc.* (inventory, stock_movements) from
# the shared cluster into the new postgres-inventory instance. Run this
# BEFORE flipping the running inventory service at the new DB.
#
# Env:
#   SOURCE_DATABASE_URL     shared cluster URL (default: localhost:5432/ecmarket_dev)
#   INVENTORY_DATABASE_URL  split cluster URL  (default: localhost:5436/inventory_dev)

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${INVENTORY_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5436/inventory_dev?sslmode=disable}"

dump_file="$(mktemp -t inventory_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has inventory_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'inventory_svc';" \
  | grep -q 1 || {
    echo "source cluster at $SOURCE_URL has no inventory_svc schema — nothing to backfill"
    exit 1
  }

echo "==> checking target cluster has inventory_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'inventory_svc';" \
  | grep -q 1 || {
    echo "target cluster at $TARGET_URL has no inventory_svc schema"
    echo "run: INVENTORY_DATABASE_URL=\"$TARGET_URL\" bash infra/scripts/migrate.sh up-service inventory"
    exit 1
  }

echo "==> dumping inventory_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=inventory_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target inventory_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    inventory_svc.stock_movements,
    inventory_svc.inventory
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'inventory'        AS table, COUNT(*) FROM inventory_svc.inventory
UNION ALL SELECT 'stock_movements', COUNT(*) FROM inventory_svc.stock_movements;
"

echo "==> done. stop the inventory service, flip DATABASE_URL to the target, restart."
