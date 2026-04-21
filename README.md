# Flowker

## Overview

Flowker is a workflow orchestration platform for financial validation. It enables financial institutions to validate transactions through external providers (KYC, AML, fraud detection) **before** writing to core banking ledgers, ensuring compliance and providing complete audit trails.

## Quick Start

1. **Clone the repository:**
    ```bash
    git clone https://github.com/LerianStudio/flowker.git
    cd flowker
    ```

2. **Setup environment variables and tooling (first-time only):**
    ```bash
    make dev-setup
    ```
    Installs tooling and copies `.env.example` to `.env`. Adjust values before the next step if needed.

    If you skip this step, `make dev` will still copy `.env.example` to `.env` on its own when `.env` is missing — `dev-setup` exists to also install the supporting toolchain (swag, gosec, golangci-lint, …).

3. **Run the local dev stack:**
    ```bash
    make dev
    ```
    `make dev` starts MongoDB (replica set) and Audit PostgreSQL via `docker-compose.dev.yml`, generates Swagger docs, then runs the Go app locally with the env vars the dev profile needs. Tear down with `make clear`.

4. **Access the API:**
   - API base URL: `http://localhost:4021`
   - Swagger UI: `http://localhost:4021/swagger/index.html`
   - Liveness: `http://localhost:4021/health/live` · Readiness: `http://localhost:4021/health/ready`

Prefer a containerized stack? `make up` starts everything via `docker-compose.yml` (production-like).

## Authentication

Flowker uses a dual authentication model driven by environment flags. The middleware that protects each route family is selected at startup by `AuthGuard` based on those flags:

| Route family | Auth | Relevant flags |
|---|---|---|
| `/v1/workflows/*`, `/v1/executions/*`, `/v1/catalog/*`, `/v1/dashboards/*`, `/v1/audit-events/*`, `/v1/executors/*`, `/v1/provider-configurations/*` | Access Manager (OIDC/JWT via `lib-auth/v2`) when `PLUGIN_AUTH_ENABLED=true`; otherwise falls back to the API-key middleware (header `X-API-Key`) | `PLUGIN_AUTH_ENABLED`, `PLUGIN_AUTH_ADDRESS`, `API_KEY_ENABLED`, `API_KEY` |
| `/v1/webhooks/*` | Infrastructure API key (header `X-API-Key`) plus the per-webhook `X-Webhook-Token` validated by the handler when the registered webhook defines one | `API_KEY_ENABLED`, `API_KEY` |
| `/health/*`, `/swagger/*`, `/version` | Public | — |

Effective behaviour by flag combination:

| `PLUGIN_AUTH_ENABLED` | `API_KEY_ENABLED` | Management routes | Webhook routes |
|---|---|---|---|
| `true` | any | Access Manager | API key (+ per-webhook token) |
| `false` | `true` | API key | API key (+ per-webhook token) |
| `false` | `false` | **Unauthenticated** | **Unauthenticated** (handler still checks `X-Webhook-Token` when the webhook defines one) |

The fully-unauthenticated mode is intended for local development only; `make dev` sets both flags to `false`.

## Swagger Documentation

Flowker ships Swagger docs generated from the code annotations by `swaggo/swag` and validated by `openapi-generator-cli`. Once the server is running the UI is available at `http://localhost:4021/swagger/index.html`; the raw artifacts live under `api/` (`docs.go`, `swagger.json`, `swagger.yaml`, `openapi.yaml`).

The docs are **not** regenerated automatically — run `make generate-docs` whenever annotations change, and `make validate-api-docs` in CI-style checks.

## Environment Variables

`make dev-setup` copies `.env.example` to `.env` on first run. Key variables:

| Variable | Purpose |
|---|---|
| `MONGO_URI`, `MONGO_DB_NAME` | MongoDB connection (replica set) |
| `AUDIT_DB_HOST`, `AUDIT_DB_PORT`, `AUDIT_DB_USER`, `AUDIT_DB_PASSWORD`, `AUDIT_DB_NAME` | Audit trail PostgreSQL |
| `PLUGIN_AUTH_ENABLED`, `PLUGIN_AUTH_ADDRESS` | Access Manager (OIDC) toggle + service URL |
| `API_KEY_ENABLED`, `API_KEY` | Webhook API-key auth toggle + value |
| `ENABLE_TELEMETRY`, `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry OTLP export |
| `CORS_ALLOWED_ORIGINS` | Comma-separated CORS allowlist |
| `SERVER_ADDRESS` | Listen address (defaults to `:4021` in `make dev`) |

See `.env.example` for the full list with defaults.

## Common Commands

```bash
# Development setup
make dev-setup          # Install tools, copy .env.example → .env, run tests
make setup-git-hooks    # Install pre-commit hooks

# Local development (recommended)
make dev                # Start Mongo + Audit PG containers, then run Go app locally
make clear              # Stop dev stack and remove volumes/networks

# Containerized run (production-like)
make build              # go build ./...
make run                # Run the compiled binary with .env
make up                 # docker compose up (production-like)
make down               # docker compose down
make rebuild-up         # Rebuild image and restart services

# Testing (build tags required — see Testing section)
make test               # Run unit + integration tests
make test-unit          # Only unit tests
make test-integration   # Only integration tests
make cover-html         # Generate HTML coverage report (artifacts/coverage.html)

# Code quality
make lint               # Run golangci-lint with auto-fix
make format             # gofmt
make tidy               # go mod tidy
make sec                # gosec

# Documentation
make generate-docs      # Regenerate Swagger/OpenAPI artifacts under api/
make validate-api-docs  # Generate + validate via openapi-generator-cli
```

## Testing

Flowker uses Go **build tags** to separate test suites. Without a tag no tests run — this is intentional so each suite can be opted into independently:

```bash
go test -tags=unit ./...                   # unit tests only
go test -tags=integration ./...            # integration tests (hits Mongo + Postgres)
go test -tags=e2e ./...                    # end-to-end tests (boots the HTTP server)
go test -tags=unit,integration,e2e ./...   # full suite
```

`make test` and the CI workflows wrap these with the expected dependencies (Docker for integration/e2e).

## Architecture

Flowker follows Hexagonal Architecture (Ports & Adapters) with a CQRS split inside the services layer.

```
internal/
├── adapters/
│   ├── http/in/        # HTTP handlers, routes, middleware (Fiber)
│   ├── mongodb/        # MongoDB repositories (workflows, executors, providers, executions)
│   └── postgresql/     # PostgreSQL repository (audit trail)
├── bootstrap/          # Configuration, server setup, dependency injection
├── services/
│   ├── command/        # Write operations (CQRS commands)
│   └── query/          # Read operations (CQRS queries)
└── testutil/           # Shared test helpers

pkg/
├── circuitbreaker/     # Resilience for external-provider calls
├── clock/              # Clock abstraction for deterministic tests
├── condition/          # Workflow edge-condition evaluator
├── constant/           # Error codes, pagination constants
├── executor/           # Executor catalog and contracts
├── executors/          # Built-in executors (HTTP, S3, Midaz, Tracer)
├── model/              # Rich domain models (Swagger annotations)
├── net/http/           # HTTP utilities (cursor pagination, Fiber helpers)
├── pagination/         # Shared cursor parsing/encoding for repositories
├── templates/          # Workflow template definitions
├── transformation/     # Data-transformation language used in workflows
├── triggers/           # Trigger definitions (webhook, schedule, …)
└── webhook/            # Webhook route registry consumed by the HTTP layer
```

## Technology Stack

- **Language**: Go 1.25.8
- **Web framework**: Fiber v2.52.12
- **Datastores**: MongoDB (replica set) for domain data · PostgreSQL for audit trail
- **AuthN**: `lib-auth/v2` (Access Manager, OIDC/JWT) + API key middleware
- **Observability**: OpenTelemetry (traces, metrics, logs via `lib-commons/v4`)
- **Testing**: `stretchr/testify`, `go.uber.org/mock`, Testcontainers for integration suites

## Documentation

Product, technical and planning documents live under `docs/pre-dev/` — PRD, TRD, data model, API design, feature map, dependency map, and task breakdown.

Additional contributor guidance lives in `CLAUDE.md` (project rules, conventions, rich-domain-model requirements).

## License

Flowker is distributed under the [Elastic License 2.0](./LICENSE). Your use of the software is subject to the terms of the [LICENSE](./LICENSE) file; the canonical text is published at <https://www.elastic.co/licensing/elastic-license>.

Flowker is **source-available**, not "open source" in the OSI sense: the source is openly published and auditable, but the LICENSE file — not this README — is the authoritative statement of your rights and obligations.
