#!/bin/sh
# Seeds the local Pub/Sub emulator with every topic + subscription the
# services expect. Runs as a one-shot init container against the emulator
# REST API — idempotent (PUT returns 200 on existing resources).
#
# Keep the lists in sync with:
#   grep -rohE '"[a-z]+-events[a-z-]*"' backend/services/ --include='*.go'
set -eu

PROJECT_ID="${PUBSUB_PROJECT_ID:-local-project}"
HOST="${PUBSUB_EMULATOR_HOST:-pubsub-emulator:8085}"

TOPICS="
cart-events
inquiry-events
inventory-events
order-events
payout-events
product-events
review-events
shipping-events
user-events
"

# <topic> <subscription> pairs — one per line, separated by space.
SUBS="
inquiry-events   inquiry-events-notification
inventory-events inventory-events-notification
order-events     order-events-inventory
order-events     order-events-notification
order-events     order-events-recommend
order-events     order-events-shipping
product-events   product-events-recommend
product-events   product-events-search
review-events    review-events-notification
shipping-events  shipping-events-notification
shipping-events  shipping-events-order
user-events      user-events-recommend
"

echo "waiting for Pub/Sub emulator at ${HOST}"
i=0
while ! curl -fsS "http://${HOST}/v1/projects/${PROJECT_ID}/topics" >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -gt 60 ]; then
    echo "emulator not reachable after 60s" >&2
    exit 1
  fi
  sleep 1
done

for t in $TOPICS; do
  [ -z "$t" ] && continue
  echo "  create topic ${t}"
  curl -fsS -X PUT "http://${HOST}/v1/projects/${PROJECT_ID}/topics/${t}" \
    >/dev/null 2>/tmp/err \
    || grep -q 'ALREADY_EXISTS' /tmp/err \
    || { echo "failed to create topic ${t}"; cat /tmp/err; exit 1; }
done

echo "$SUBS" | while read -r topic sub; do
  [ -z "$topic" ] && continue
  echo "  create subscription ${sub} -> ${topic}"
  curl -fsS -X PUT -H 'Content-Type: application/json' \
    -d "{\"topic\":\"projects/${PROJECT_ID}/topics/${topic}\"}" \
    "http://${HOST}/v1/projects/${PROJECT_ID}/subscriptions/${sub}" \
    >/dev/null 2>/tmp/err \
    || grep -q 'ALREADY_EXISTS' /tmp/err \
    || { echo "failed to create subscription ${sub}"; cat /tmp/err; exit 1; }
done

echo "pubsub emulator seeded"
