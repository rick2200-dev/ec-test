#!/bin/bash
# One-shot cutover helper for the Phase 3 subscription DB split.
#
# Copies every row in subscription_svc.* (plans + subscriptions, both
# seller- and buyer-side) from the shared cluster into the new
# postgres-subscription instance. Run this BEFORE flipping the running
# subscription service at the new DB so in-flight seller/buyer state is
# preserved.
#
# Idempotent: re-running wipes the target subscription_svc rows and
# reloads from source. Safe to re-run while the subscription service is
# stopped; do not run while it is writing (target TRUNCATE will race).
#
# Env:
#   SOURCE_DATABASE_URL       shared cluster URL (default: localhost:5432/ecmarket_dev, ecmarket user)
#   SUBSCRIPTION_DATABASE_URL split cluster URL  (default: localhost:5434/subscription_dev, ecmarket user)
#
# Example:
#   bash infra/scripts/backfill_subscription_split.sh
#
#   # remote cluster:
#   SOURCE_DATABASE_URL='postgres://...@prod-shared/ecmarket' \
#   SUBSCRIPTION_DATABASE_URL='postgres://...@prod-subscription/subscription' \
#     bash infra/scripts/backfill_subscription_split.sh

set -euo pipefail

SOURCE_URL="${SOURCE_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
TARGET_URL="${SUBSCRIPTION_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5434/subscription_dev?sslmode=disable}"

dump_file="$(mktemp -t subscription_backfill.XXXXXX.sql)"
trap 'rm -f "$dump_file"' EXIT

echo "==> checking source cluster has subscription_svc"
psql "$SOURCE_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'subscription_svc';" \
  | grep -q 1 || {
    echo "source cluster at $SOURCE_URL has no subscription_svc schema — nothing to backfill"
    exit 1
  }

echo "==> checking target cluster has subscription_svc (migrations applied)"
psql "$TARGET_URL" -tAc \
  "SELECT 1 FROM pg_namespace WHERE nspname = 'subscription_svc';" \
  | grep -q 1 || {
    echo "target cluster at $TARGET_URL has no subscription_svc schema"
    echo "run: SUBSCRIPTION_DATABASE_URL=\"$TARGET_URL\" bash infra/scripts/migrate.sh up-service subscription"
    exit 1
  }

echo "==> dumping subscription_svc rows from source"
pg_dump "$SOURCE_URL" \
  --schema=subscription_svc \
  --data-only \
  --no-owner \
  --no-privileges \
  --disable-triggers \
  > "$dump_file"

echo "==> wiping target subscription_svc rows"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 <<'SQL'
TRUNCATE
    subscription_svc.buyer_subscriptions,
    subscription_svc.buyer_plans,
    subscription_svc.seller_subscriptions,
    subscription_svc.subscription_plans
RESTART IDENTITY CASCADE;
SQL

echo "==> loading dump into target"
psql "$TARGET_URL" -v ON_ERROR_STOP=1 -f "$dump_file"

echo "==> row counts on target:"
psql "$TARGET_URL" -c "
SELECT 'subscription_plans'      AS table, COUNT(*) FROM subscription_svc.subscription_plans
UNION ALL SELECT 'seller_subscriptions', COUNT(*) FROM subscription_svc.seller_subscriptions
UNION ALL SELECT 'buyer_plans',          COUNT(*) FROM subscription_svc.buyer_plans
UNION ALL SELECT 'buyer_subscriptions',  COUNT(*) FROM subscription_svc.buyer_subscriptions;
"

echo "==> done. stop the subscription service, flip DATABASE_URL to the target, restart."
