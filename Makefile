.PHONY: help deps-up deps-down migrate seed proto-gen openapi-gen \
       dev-gateway dev-auth dev-catalog dev-inventory dev-order dev-search dev-recommend dev-notification dev-cart dev-inquiry dev-review \
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
migrate:
	migrate -path infra/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path infra/db/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir infra/db/migrations -seq $$name

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

# ─── Frontend ──────────────────────────────────────────────────
dev-buyer:
	pnpm --filter buyer dev

dev-seller:
	pnpm --filter seller dev

dev-admin:
	pnpm --filter admin dev

# ─── Build & Test ──────────────────────────────────────────────
build-all:
	@for svc in gateway auth catalog inventory order search recommend notification cart inquiry review; do \
		echo "Building $$svc..."; \
		cd backend/services/$$svc && go build -o ../../../bin/$$svc ./cmd/server && cd ../../..; \
	done

lint-go:
	@for svc in gateway auth catalog inventory order search recommend notification cart inquiry review; do \
		echo "Linting $$svc..."; \
		cd backend/services/$$svc && golangci-lint run ./... && cd ../../..; \
	done
	cd backend/pkg && golangci-lint run ./...

test-go:
	@for svc in gateway auth catalog inventory order search recommend notification cart inquiry review; do \
		echo "Testing $$svc..."; \
		cd backend/services/$$svc && go test ./... && cd ../../..; \
	done
	cd backend/pkg && go test ./...

# ─── Proto & OpenAPI ──────────────────────────────────────────
proto-gen:
	buf generate backend/proto

openapi-gen:
	pnpm --filter api-client generate
