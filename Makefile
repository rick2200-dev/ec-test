.PHONY: help deps-up deps-down migrate seed proto-gen openapi-gen \
       dev-gateway dev-auth dev-catalog dev-inventory dev-order dev-search dev-recommend dev-notification \
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
	docker compose -f docker/docker-compose.deps.yaml up -d

deps-down:
	docker compose -f docker/docker-compose.deps.yaml down

# ─── Database ──────────────────────────────────────────────────
migrate:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name

seed:
	psql "$(DATABASE_URL)" -f db/seeds/dev_tenants.sql

# ─── Go Services (with air hot-reload) ────────────────────────
dev-gateway:
	cd services/gateway && air

dev-auth:
	cd services/auth && air

dev-catalog:
	cd services/catalog && air

dev-inventory:
	cd services/inventory && air

dev-order:
	cd services/order && air

dev-search:
	cd services/search && air

dev-recommend:
	cd services/recommend && air

dev-notification:
	cd services/notification && air

# ─── Frontend ──────────────────────────────────────────────────
dev-buyer:
	pnpm --filter buyer dev

dev-seller:
	pnpm --filter seller dev

dev-admin:
	pnpm --filter admin dev

# ─── Build & Test ──────────────────────────────────────────────
build-all:
	@for svc in gateway auth catalog inventory order search recommend notification; do \
		echo "Building $$svc..."; \
		cd services/$$svc && go build -o ../../bin/$$svc ./cmd/server && cd ../..; \
	done

lint-go:
	@for svc in gateway auth catalog inventory order search recommend notification; do \
		echo "Linting $$svc..."; \
		cd services/$$svc && golangci-lint run ./... && cd ../..; \
	done
	cd pkg && golangci-lint run ./...

test-go:
	@for svc in gateway auth catalog inventory order search recommend notification; do \
		echo "Testing $$svc..."; \
		cd services/$$svc && go test ./... && cd ../..; \
	done
	cd pkg && go test ./...

# ─── Proto & OpenAPI ──────────────────────────────────────────
proto-gen:
	buf generate proto

openapi-gen:
	pnpm --filter api-client generate
