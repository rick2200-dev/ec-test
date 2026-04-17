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
# Each DB-backed service needs DATABASE_URL pointing at its split
# Postgres instance (Phase 3). Gateway and cart have no DB.
dev-gateway:
	cd backend/services/gateway && air

dev-auth:
	DATABASE_URL=postgres://auth_role:localdev@localhost:5437/auth_dev?sslmode=disable cd backend/services/auth && air

dev-catalog:
	DATABASE_URL=postgres://catalog_role:localdev@localhost:5442/catalog_dev?sslmode=disable cd backend/services/catalog && air

dev-inventory:
	DATABASE_URL=postgres://inventory_role:localdev@localhost:5436/inventory_dev?sslmode=disable cd backend/services/inventory && air

dev-order:
	DATABASE_URL=postgres://order_role:localdev@localhost:5443/order_dev?sslmode=disable cd backend/services/order && air

dev-search:
	DATABASE_URL=postgres://search_role:localdev@localhost:5440/search_dev?sslmode=disable cd backend/services/search && air

dev-recommend:
	DATABASE_URL=postgres://recommend_role:localdev@localhost:5441/recommend_dev?sslmode=disable cd backend/services/recommend && air

dev-notification:
	DATABASE_URL=postgres://notification_role:localdev@localhost:5433/notification_dev?sslmode=disable cd backend/services/notification && air

dev-cart:
	cd backend/services/cart && air

dev-inquiry:
	DATABASE_URL=postgres://inquiry_role:localdev@localhost:5438/inquiry_dev?sslmode=disable cd backend/services/inquiry && air

dev-review:
	DATABASE_URL=postgres://review_role:localdev@localhost:5439/review_dev?sslmode=disable cd backend/services/review && air

dev-subscription:
	DATABASE_URL=postgres://subscription_role:localdev@localhost:5434/subscription_dev?sslmode=disable cd backend/services/subscription && air

dev-shipping:
	DATABASE_URL=postgres://shipping_role:localdev@localhost:5435/shipping_dev?sslmode=disable cd backend/services/shipping && air

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
