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
)

db_url_with_table() {
  local svc="$1"
  local sep="?"
  case "$DATABASE_URL" in *\?*) sep="&";; esac
  echo "${DATABASE_URL}${sep}x-migrations-table=schema_migrations_${svc}"
}

migrate_up_all() {
  for svc in "${SERVICES[@]}"; do
    local dir="${MIGRATIONS_ROOT}/${svc}"
    [ -d "$dir" ] || { echo "skip ${svc}: no migrations"; continue; }
    echo "==> migrate up ${svc}"
    migrate -path "$dir" -database "$(db_url_with_table "$svc")" up
  done
}

migrate_down_all() {
  # Reverse order for down so cross-schema refs tear down cleanly.
  for ((i=${#SERVICES[@]}-1; i>=0; i--)); do
    local svc="${SERVICES[$i]}"
    local dir="${MIGRATIONS_ROOT}/${svc}"
    [ -d "$dir" ] || continue
    echo "==> migrate down 1 ${svc}"
    migrate -path "$dir" -database "$(db_url_with_table "$svc")" down 1 || true
  done
}

migrate_service_up() {
  local svc="$1"
  local dir="${MIGRATIONS_ROOT}/${svc}"
  [ -d "$dir" ] || { echo "no migration dir for ${svc}"; exit 1; }
  migrate -path "$dir" -database "$(db_url_with_table "$svc")" up
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
