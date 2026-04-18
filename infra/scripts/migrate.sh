#!/bin/bash
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"
MIGRATIONS_ROOT="${MIGRATIONS_ROOT:-infra/db/migrations}"

# Service-owned migration directories run in this fixed order because some
# cross-schema references (e.g. catalog/004_create_seller_plan_boost reads
# auth_svc.sellers) still exist in legacy migrations. Keep bootstrap first;
# otherwise follow the historical creation order.
SERVICES=(
  bootstrap
  auth
  catalog
  inventory
  order
  subscription
  inquiry
  review
  shipping
  notification
  recommend
  search
  coupon
  loyalty
  grants
)

# Per-service DB URL override. Services that have been physically split
# out of the shared cluster point at their own DB instance via a
# dedicated env var. Everything else falls back to DATABASE_URL.
db_url_for_service() {
  local svc="$1"
  case "$svc" in
    notification)
      echo "${NOTIFICATION_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5433/notification_dev?sslmode=disable}"
      ;;
    subscription)
      echo "${SUBSCRIPTION_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5434/subscription_dev?sslmode=disable}"
      ;;
    shipping)
      echo "${SHIPPING_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5435/shipping_dev?sslmode=disable}"
      ;;
    inventory)
      echo "${INVENTORY_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5436/inventory_dev?sslmode=disable}"
      ;;
    auth)
      echo "${AUTH_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5437/auth_dev?sslmode=disable}"
      ;;
    inquiry)
      echo "${INQUIRY_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5438/inquiry_dev?sslmode=disable}"
      ;;
    review)
      echo "${REVIEW_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5439/review_dev?sslmode=disable}"
      ;;
    search)
      echo "${SEARCH_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5440/search_dev?sslmode=disable}"
      ;;
    recommend)
      echo "${RECOMMEND_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5441/recommend_dev?sslmode=disable}"
      ;;
    catalog)
      echo "${CATALOG_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5442/catalog_dev?sslmode=disable}"
      ;;
    order)
      echo "${ORDER_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5443/order_dev?sslmode=disable}"
      ;;
    coupon)
      echo "${COUPON_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5444/coupon_dev?sslmode=disable}"
      ;;
    loyalty)
      echo "${LOYALTY_DATABASE_URL:-postgres://ecmarket:localdev@localhost:5445/loyalty_dev?sslmode=disable}"
      ;;
    *)
      echo "$DATABASE_URL"
      ;;
  esac
}

# append the schema_migrations table name to the URL's query string so
# every service keeps its own migration history.
append_migrations_table() {
  local url="$1" svc="$2"
  local sep="?"
  case "$url" in *\?*) sep="&";; esac
  echo "${url}${sep}x-migrations-table=schema_migrations_${svc}"
}

migrate_up_all() {
  for svc in "${SERVICES[@]}"; do
    local dir="${MIGRATIONS_ROOT}/${svc}"
    [ -d "$dir" ] || { echo "skip ${svc}: no migrations"; continue; }
    echo "==> migrate up ${svc}"
    migrate -path "$dir" -database "$(append_migrations_table "$(db_url_for_service "$svc")" "$svc")" up
  done
}

migrate_down_all() {
  # Reverse order for down so cross-schema refs tear down cleanly.
  for ((i=${#SERVICES[@]}-1; i>=0; i--)); do
    local svc="${SERVICES[$i]}"
    local dir="${MIGRATIONS_ROOT}/${svc}"
    [ -d "$dir" ] || continue
    echo "==> migrate down 1 ${svc}"
    migrate -path "$dir" -database "$(append_migrations_table "$(db_url_for_service "$svc")" "$svc")" down 1 || true
  done
}

migrate_service_up() {
  local svc="$1"
  local dir="${MIGRATIONS_ROOT}/${svc}"
  [ -d "$dir" ] || { echo "no migration dir for ${svc}"; exit 1; }
  migrate -path "$dir" -database "$(append_migrations_table "$(db_url_for_service "$svc")" "$svc")" up
}

case "${1:-up}" in
  up)             migrate_up_all ;;
  down)           migrate_down_all ;;
  up-service)     migrate_service_up "${2:?service name required}" ;;
  *)
    echo "Usage: $0 [up|down|up-service <name>]"
    exit 1
    ;;
esac
