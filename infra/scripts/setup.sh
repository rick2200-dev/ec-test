#!/bin/bash
set -euo pipefail

echo "=== EC Marketplace Local Setup ==="

# Check prerequisites
command -v go >/dev/null 2>&1 || { echo "ERROR: go is required"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "ERROR: node is required"; exit 1; }
command -v pnpm >/dev/null 2>&1 || { echo "ERROR: pnpm is required"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "ERROR: docker is required"; exit 1; }

echo "[1/5] Installing Node.js dependencies..."
pnpm install

echo "[2/5] Starting local dependencies..."
docker compose -f infra/docker/docker-compose.deps.yaml up -d

echo "[3/5] Waiting for PostgreSQL..."
until docker compose -f infra/docker/docker-compose.deps.yaml exec -T postgres pg_isready -U ecmarket -d ecmarket_dev >/dev/null 2>&1; do
  sleep 1
done

echo "[4/5] Running database migrations..."
if command -v migrate >/dev/null 2>&1; then
  migrate -path infra/db/migrations -database "postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable" up
else
  echo "WARN: golang-migrate not found. Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
fi

echo "[5/5] Seeding development data..."
PGPASSWORD=localdev psql -h localhost -U ecmarket -d ecmarket_dev -f infra/db/seeds/dev_tenants.sql 2>/dev/null || echo "WARN: psql not found or seed failed. Run manually: make seed"

echo ""
echo "=== Setup complete! ==="
echo "Start a service:  make dev-gateway"
echo "Start frontend:   make dev-buyer"
