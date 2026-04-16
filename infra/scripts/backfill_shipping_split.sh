#!/bin/bash
# One-shot cutover helper for the Phase 3 shipping DB split.
#
# Copies every row in shipping_svc.* (shipments, shipment_events,
# cancelled_order_tombstones, outbox_events) from the shared cluster
# into the new postgres-shipping instance. Run this BEFORE flipping the
# running shipping service at the new DB so in-flight shipments and any
# un-relayed outbox rows are preserved.
#
# Stop the shipping service first — the outbox relay is a long-running
# reader that will see the truncate/restore window. Re-running while
# shipping is writing will race on outbox_events.
#
# Env:
#   SOURCE_DATABASE_URL    shared cluster URL (default: localhost:5432/ecmarket_dev, ecmarket user)
#   SHIPPING_DATABASE_URL  split cluster URL  (default: localhost:5435/shipping_dev,  ecmarket user)
#
# Example:
#   bash infra/scripts/backfill_shipping_split.sh

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${SHIPPING_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5435/shipping_dev?sslmode=disable}"

dump_file="$(mktemp -t shipping_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has shipping_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'shipping_svc';" \
  | grep -q 1 || {
    echo "source cluster at $SOURCE_URL has no shipping_svc schema — nothing to backfill"
    exit 1
  }

echo "==> checking target cluster has shipping_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'shipping_svc';" \
  | grep -q 1 || {
    echo "target cluster at $TARGET_URL has no shipping_svc schema"
    echo "run: SHIPPING_DATABASE_URL=\"$TARGET_URL\" bash infra/scripts/migrate.sh up-service shipping"
    exit 1
  }

echo "==> dumping shipping_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=shipping_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target shipping_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    shipping_svc.outbox_events,
    shipping_svc.cancelled_order_tombstones,
    shipping_svc.shipment_events,
    shipping_svc.shipments
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'shipments'                   AS table, COUNT(*) FROM shipping_svc.shipments
UNION ALL SELECT 'shipment_events',            COUNT(*) FROM shipping_svc.shipment_events
UNION ALL SELECT 'cancelled_order_tombstones', COUNT(*) FROM shipping_svc.cancelled_order_tombstones
UNION ALL SELECT 'outbox_events',              COUNT(*) FROM shipping_svc.outbox_events;
"

echo "==> done. stop the shipping service, flip DATABASE_URL to the target, restart."
