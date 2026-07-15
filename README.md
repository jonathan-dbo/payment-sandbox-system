# Payment Sandbox Backend

This repository contains the Go backend for the payment sandbox assessment app.

## Prerequisites

- Go 1.21+
- Docker + Docker Compose (optional, for PostgreSQL)

## Environment Setup

1. Copy the example environment file:

```bash
cp .env.example .env
```

2. Update at least `JWT_SECRET` in `.env` to a non-default value.

Notes:
- Ensure DB variables are valid before running the API.

## Run the Backend

Use the `Makefile` as the canonical entry point for common development tasks.

Run directly:

```bash
go run ./cmd/api
```

Or via Makefile:

```bash
make run
```

Server starts on `http://localhost:8080`.

Health check:

```bash
curl http://localhost:8080/health
```

## Run with Local PostgreSQL (Docker)

One-shot local database setup (starts Postgres, waits until ready, applies all `*.sql` in `internal/infrastructure/database/schema/` sorted by name):

```bash
make db init
```

Equivalent shortcuts:

```bash
make db-init
# or
make db-migrate
```

(`make migrate` runs the same steps as `db-migrate`. `make postgres` only starts Postgres and waits — use before `make init` if you split the steps.)

Then run the API:

```bash
make run
```

Run API + DB together in containers:

```bash
make compose-up
```

Reset local DB (drop/create + schema init):

```bash
make db-reset
```

Container runtime env strategy:
- `docker-compose.yml` loads `.env.example` via `env_file` for both `api` and `postgres`.
- Most new env keys only need to be added in `.env.example` (not in compose).
- Keep explicit compose `environment` entries only for container-specific overrides (for example, `DB_HOST=postgres` for the API container).
- Override values by exporting shell env vars before `docker compose up -d` when needed.

## Testing

Run all tests:

```bash
go test -v ./...
```

Or:

```bash
make test
```

Run tests with coverage:

```bash
go test -cover ./...
```

Or:

```bash
make test-cover
```

Run race + coverage:

```bash
go test -v -race -cover ./...
```

## Performance Tests (Benchmarks)

Hot paths are covered by Go benchmark tests (`Benchmark*` functions), not a separate load-testing tool:

- Usecase-level: `internal/application/payment/payment_usecase_bench_test.go`, `internal/application/refund/refund_usecase_bench_test.go`, `internal/application/topup/topup_usecase_bench_test.go`
- HTTP-level (via `httptest` against the real router): `internal/infrastructure/http/router_bench_test.go`

Run all benchmarks:

```bash
go test -bench . -benchmem ./...
```

Or via Makefile:

```bash
make bench
```

`make bench` accepts overrides:

```bash
make bench BENCH_PATTERN=BenchmarkHTTPHealth BENCH_TIME=200x BENCH_PKG=./internal/infrastructure/http/...
```

- `BENCH_PATTERN` — regex passed to `-bench` (default `.`, i.e. run everything)
- `BENCH_TIME` — passed to `-benchtime` (default `1x`; use e.g. `200x` or `2s` for more stable numbers)
- `BENCH_PKG` — package path to benchmark (default `./...`)

`perf-test` is an alias for `bench`:

```bash
make perf-test
```

### Profiling

Generate a CPU or memory profile from a benchmark run:

```bash
make bench-cpu BENCH_PATTERN=BenchmarkHTTPCreateInvoice BENCH_TIME=200x BENCH_PKG=./internal/infrastructure/http/...
make bench-mem BENCH_PATTERN=BenchmarkHTTPCreateInvoice BENCH_TIME=200x BENCH_PKG=./internal/infrastructure/http/...
```

Profiles are written to `bin/profile/` (`cpu.prof` / `mem.prof`) alongside the compiled test binary. Inspect with:

```bash
go tool pprof bin/profile/bench.test bin/profile/cpu.prof
```

Notes:
- Benchmarks silence structured logging (`shared.LogEvent`) and gin's request logger so I/O doesn't distort timing/allocation numbers.
- `BenchmarkHTTPLogin` is dominated by bcrypt password hashing (~100ms+/op) by design — this reflects real login cost, not a bug.
- Low `-benchtime` iteration counts (e.g. `20x`) can be noisy; prefer `100x`+ or a duration (`1s`, `2s`) for representative numbers.

## API Documentation (OpenAPI / Swagger)

OpenAPI spec file:
- `internal/api/api.yaml`

End-to-end manual testing walkthrough (Swagger UI / Postman / curl, covering every flow and error case):
- `E2E_TESTING_GUIDE.md`

Served by the app:
- Raw spec: `http://localhost:8080/api.yaml`
- Swagger UI: `http://localhost:8080/docs/index.html`

Regenerate generated OpenAPI interfaces:

```bash
go generate ./internal/interfaces
```

Or:

```bash
make gen
```

## Make Targets

Use `make help` to list targets and descriptions.
