SHELL := /bin/sh

APP_CMD ?= ./cmd/api
BIN_DIR ?= bin
BIN_NAME ?= api

DB_USER ?= user
DB_NAME ?= postgres
DB_INIT_SQL ?= /docker-entrypoint-initdb.d/001_init.sql
SCHEMA_DIR ?= internal/infrastructure/database/schema
DB_FK_SQL ?= $(SCHEMA_DIR)/002_add_foreign_keys.sql

.PHONY: help build run test test-cover lint tidy gen mocks postgres db db-up db-down db-wait init db-init db-migrate migrate db-reset db-fk db-fk-verify compose-up compose-down bench bench-cpu bench-mem perf-test e2e

help: ## list available targets
	@awk 'BEGIN {FS = ":.*## "; print "Available targets:"} /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## compile backend binary
	go build -o $(BIN_DIR)/$(BIN_NAME) $(APP_CMD)

run: ## run API locally
	go run $(APP_CMD)

test: ## run all tests
	go test ./...

test-cover: ## run tests with coverage
	go test -cover ./...

BENCH_PATTERN ?= .
BENCH_TIME ?= 1x
BENCH_PKG ?= ./...
PROFILE_DIR ?= bin/profile

bench: ## run all Go benchmarks (BENCH_PATTERN, BENCH_TIME, BENCH_PKG overridable)
	go test -run '^$$' -bench '$(BENCH_PATTERN)' -benchtime=$(BENCH_TIME) -benchmem $(BENCH_PKG)

perf-test: bench ## alias for bench (perf/load-style entry point for hot-path usecases + HTTP handlers)

e2e: ## run full E2E smoke test against a live server (BASE_URL overridable, default http://localhost:8080); writes report to reports/
	@./scripts/e2e.sh $(E2E_ARGS)

bench-cpu: ## run benchmarks with CPU profiling (writes bin/profile/cpu.prof)
	@mkdir -p $(PROFILE_DIR)
	go test -run '^$$' -bench '$(BENCH_PATTERN)' -benchtime=$(BENCH_TIME) -benchmem -cpuprofile=$(PROFILE_DIR)/cpu.prof -o $(PROFILE_DIR)/bench.test $(BENCH_PKG)
	@echo "CPU profile written to $(PROFILE_DIR)/cpu.prof — inspect with: go tool pprof $(PROFILE_DIR)/bench.test $(PROFILE_DIR)/cpu.prof"

bench-mem: ## run benchmarks with memory profiling (writes bin/profile/mem.prof)
	@mkdir -p $(PROFILE_DIR)
	go test -run '^$$' -bench '$(BENCH_PATTERN)' -benchtime=$(BENCH_TIME) -benchmem -memprofile=$(PROFILE_DIR)/mem.prof -o $(PROFILE_DIR)/bench.test $(BENCH_PKG)
	@echo "Mem profile written to $(PROFILE_DIR)/mem.prof — inspect with: go tool pprof $(PROFILE_DIR)/bench.test $(PROFILE_DIR)/mem.prof"

lint: ## run go vet checks
	go vet ./...

tidy: ## tidy go modules
	go mod tidy

gen: ## regenerate OpenAPI artifacts
	go generate ./internal/interfaces

mocks: ## regenerate mockery mocks
	go run github.com/vektra/mockery/v2@latest --name UserRepository --dir internal/application/user --output internal/mocks --filename mock_user_repository.go --outpkg mocks --with-expecter

postgres: db ## alias: start local PostgreSQL and wait until ready

db: ## start local postgres and wait until ready (use before init or run)
	docker compose up -d postgres
	@$(MAKE) db-wait

db-up: ## start local postgres using docker compose
	docker compose up -d postgres

db-down: ## stop local postgres service
	docker compose down

# Wait until Postgres accepts connections (max ~30s).
db-wait: ## block until local postgres accepts connections (expects container running)
	@i=0; \
	while [ $$i -lt 30 ]; do \
		if docker compose exec -T postgres pg_isready -U $(DB_USER) -d $(DB_NAME) >/dev/null 2>&1; then \
			exit 0; \
		fi; \
		i=$$((i + 1)); \
		sleep 1; \
	done; \
	echo "postgres did not become ready in time" >&2; \
	exit 1

init: ## apply SCHEMA_DIR/*.sql to local postgres (sorted by filename; postgres must be running)
	@set -e; \
	for f in $$(ls -1 $(SCHEMA_DIR)/*.sql 2>/dev/null | sort); do \
		echo "Applying $$f..."; \
		cat "$$f" | docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1; \
	done

db-migrate: db init ## start postgres, wait, then apply migrations

db-init: db init ## one-shot DB setup (same as: make db init)

migrate: db-migrate ## alias for db-migrate

db-reset: ## reset local postgres schema using init SQL
	@$(MAKE) db
	docker compose exec -T postgres psql -U $(DB_USER) -d postgres -c "DROP DATABASE IF EXISTS $(DB_NAME);"
	docker compose exec -T postgres psql -U $(DB_USER) -d postgres -c "CREATE DATABASE $(DB_NAME);"
	docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f $(DB_INIT_SQL)

db-fk: ## apply 002_add_foreign_keys.sql to running postgres container
	@test -f "$(DB_FK_SQL)" || (echo "missing SQL file: $(DB_FK_SQL)" >&2; exit 1)
	docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 < "$(DB_FK_SQL)"

db-fk-verify: ## verify expected foreign key constraints exist
	docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -c "SELECT conname FROM pg_constraint WHERE conname IN ('fk_merchants_user_id','fk_invoices_merchant_id','fk_payment_intents_invoice_id','fk_refunds_invoice_id','fk_refunds_merchant_id','fk_wallets_merchant_id','fk_top_ups_merchant_id') ORDER BY conname;"

compose-up: ## run api + postgres via docker compose
	docker compose up -d

compose-down: ## stop api + postgres containers
	docker compose down

##docker compose down -v
##docker compose up -d --build --force-recreate
##make db-init