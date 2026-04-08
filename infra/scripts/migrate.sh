#!/bin/bash
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable}"

case "${1:-up}" in
  up)
    migrate -path infra/db/migrations -database "$DATABASE_URL" up
    ;;
  down)
    migrate -path infra/db/migrations -database "$DATABASE_URL" down 1
    ;;
  *)
    echo "Usage: $0 [up|down]"
    exit 1
    ;;
esac
