.PHONY: help deps-up deps-down migrate seed proto-gen openapi-gen \
       dev-gateway dev-auth dev-catalog dev-inventory dev-order dev-search dev-recommend dev-notification dev-cart dev-inquiry dev-review dev-subscription dev-shipping \
       dev-buyer dev-seller dev-admin \
       build-all lint-go test-go

# Default
help:
	@echo "Usage:"
	@echo "  make deps-up          - Start local dependencies (PG, Redis, Pub/Sub emulator)"
	@echo "  make deps-down        - Stop local dependencies"
	@echo "  make migrate          - Run database migrations"
	@echo "  make seed             - Seed development data"
	@echo "  make dev-<service>    - Run a Go service with hot-reload (air)"
	@echo "  make dev-buyer        - Run buyer Next.js app"
	@echo "  make dev-seller       - Run seller Next.js app"
	@echo "  make dev-admin        - Run admin Next.js app"
	@echo "  make build-all        - Build all Go services"
	@echo "  make lint-go          - Lint all Go code"
	@echo "  make test-go          - Test all Go code"

# ─── Dependencies ──────────────────────────────────────────────
DATABASE_URL ?= postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable

deps-up:
	docker compose -f infra/docker/docker-compose.deps.yaml up -d

deps-down:
	docker compose -f infra/docker/docker-compose.deps.yaml down

# ─── Database ──────────────────────────────────────────────────
# Migrations are per-service (infra/db/migrations/<svc>/) with each service
# tracking its own schema_migrations_<svc> version table. The wrapper script
# runs them in a fixed dependency-respecting order.
migrate:
	DATABASE_URL="$(DATABASE_URL)" bash infra/scripts/migrate.sh up

migrate-down:
	DATABASE_URL="$(DATABASE_URL)" bash infra/scripts/migrate.sh down

migrate-service:
	@[ -n "$(SVC)" ] || { echo "Usage: make migrate-service SVC=<name>"; exit 1; }
	DATABASE_URL="$(DATABASE_URL)" bash infra/scripts/migrate.sh up-service $(SVC)

migrate-create:
	@read -p "Service (auth/catalog/order/...): " svc; \
	read -p "Migration name: " name; \
	dir=infra/db/migrations/$$svc; \
	[ -d $$dir ] || { echo "no such service dir: $$dir"; exit 1; }; \
	migrate create -ext sql -dir $$dir -seq $$name

seed:
	psql "$(DATABASE_URL)" -f infra/db/seeds/dev_tenants.sql

# ─── Go Services (with air hot-reload) ────────────────────────
dev-gateway:
	cd backend/services/gateway && air

dev-auth:
	cd backend/services/auth && air

dev-catalog:
	cd backend/services/catalog && air

dev-inventory:
	cd backend/services/inventory && air

dev-order:
	cd backend/services/order && air

dev-search:
	cd backend/services/search && air

dev-recommend:
	cd backend/services/recommend && air

dev-notification:
	cd backend/services/notification && air

dev-cart:
	cd backend/services/cart && air

dev-inquiry:
	cd backend/services/inquiry && air

dev-review:
	cd backend/services/review && air

dev-subscription:
	cd backend/services/subscription && air

dev-shipping:
	cd backend/services/shipping && air

# ─── Frontend ──────────────────────────────────────────────────
dev-buyer:
	pnpm --filter buyer dev

dev-seller:
	pnpm --filter seller dev

dev-admin:
	pnpm --filter admin dev

# ─── Build & Test ──────────────────────────────────────────────
GO_SERVICES := gateway auth catalog inventory order search recommend notification cart inquiry review subscription shipping

build-all:
	@set -e; for svc in $(GO_SERVICES); do \
		echo "Building $$svc..."; \
		( cd backend/services/$$svc && go build -o ../../../bin/$$svc ./cmd/server ); \
	done

lint-go:
	@fail=0; for svc in $(GO_SERVICES); do \
		echo "Linting $$svc..."; \
		( cd backend/services/$$svc && golangci-lint run ./... ) || fail=1; \
	done; \
	( cd backend/pkg && golangci-lint run ./... ) || fail=1; \
	exit $$fail

test-go:
	@fail=0; for svc in $(GO_SERVICES); do \
		echo "Testing $$svc..."; \
		( cd backend/services/$$svc && go test ./... ) || fail=1; \
	done; \
	( cd backend/pkg && go test ./... ) || fail=1; \
	exit $$fail

# ─── Proto & OpenAPI ──────────────────────────────────────────
proto-gen:
	@set -e; \
	for dir in backend/shared/api backend/services/*/api; do \
		if [ -f $$dir/buf.gen.yaml ]; then \
			echo "  buf generate $$dir"; \
			( cd $$dir && buf generate ); \
		fi; \
	done

openapi-gen:
	pnpm --filter api-client generate
