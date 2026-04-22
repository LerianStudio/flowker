# AGENTS.md — Flowker Quick-Start for AI Agents

## What Is This?

Flowker is a **workflow orchestration platform for financial validation** written in Go. It enables financial institutions to validate transactions through external providers (KYC, AML, fraud detection) **before** writing to core banking ledgers, ensuring compliance with complete audit trails.

## Quick Facts

| Aspect | Detail |
|--------|--------|
| Language | Go 1.25+ |
| Module | `github.com/LerianStudio/flowker` |
| License | Elastic License 2.0 |
| Architecture | Hexagonal + CQRS |
| HTTP Framework | Fiber v2 |
| Databases | MongoDB (replica set) for domain data, PostgreSQL for audit trail |
| Default Port | 4021 |

## Get Running

```bash
make dev-setup     # Install tools, copy .env.example -> .env
make dev           # Start Mongo + Audit PG + Go app locally
make test          # Run all tests (unit, integration, e2e)
make lint          # Lint all code
```

## Project Structure (What Goes Where)

```
internal/
  adapters/http/in/         -> HTTP handlers and routes (Fiber)
  adapters/mongodb/         -> MongoDB repositories (workflow, execution, executor_configuration, etc.)
  adapters/postgresql/      -> PostgreSQL repository (audit trail)
  bootstrap/                -> Config, DI, server lifecycle
  services/command/         -> Write use cases (one file per operation)
  services/query/           -> Read use cases (one file per operation)

pkg/
  model/                    -> Domain models with Swagger annotations
  constant/errors.go        -> Error codes (FLK-0001 through FLK-0612)
  errors.go                 -> Typed error structs + ValidateBusinessError factory
  executor/                 -> Executor catalog, provider, runner contracts
  executors/                -> Built-in executors (HTTP, S3, Midaz, Tracer)
  circuitbreaker/           -> Resilience for external-provider calls
  condition/                -> Workflow edge-condition evaluator
  transformation/           -> Data-transformation language for workflows
  triggers/                 -> Trigger definitions (webhook, schedule)
  webhook/                  -> Webhook route registry
  net/http/                 -> Middleware, cursor pagination, Fiber helpers
```

## Key Conventions

1. **Rich Domain Models**: Validation lives in constructors/methods, not external functions
2. **Error handling**: Business errors return directly; technical errors wrap with `%w`
3. **Provider vs Executor**: Providers are static catalog entries; Executors are dynamic DB configs
4. **File naming**: `snake_case.go`, one handler or operation per file
5. **Imports**: stdlib, external, internal (blank-line separated)
6. **Context**: Always first param
7. **IDs**: `string` (UUID format)
8. **Commit messages**: `type(scope): description` — types: feature|fix|refactor|style|test|docs|build

## Key Files to Read First

| File | Why |
|------|-----|
| `internal/bootstrap/config.go` | Composition root, all env vars, init sequence |
| `internal/adapters/http/in/routes.go` | All API routes registered here |
| `pkg/model/workflow.go` | Core workflow domain model |
| `pkg/constant/errors.go` | All error codes |
| `pkg/errors.go` | Error types + ValidateBusinessError factory |
| `.env.example` | All environment variables |
| `docs/PROJECT_RULES.md` | Coding standards (DO NOT overwrite) |

## What NOT To Do

- Do NOT overwrite `docs/PROJECT_RULES.md`
- Do NOT use anemic domain models (external validation functions)
- Do NOT panic — return errors
- Do NOT put domain logic in handlers or repositories
- Do NOT nest metadata values

## Deeper References

- **[CLAUDE.md](CLAUDE.md)** — Deep technical reference (architecture, patterns, auth model, database)
- **[llms-full.txt](llms-full.txt)** — Complete reference with all env vars, API endpoints, error codes, models
- **[llms.txt](llms.txt)** — Concise overview following llmstxt.org spec
- **[docs/PROJECT_RULES.md](docs/PROJECT_RULES.md)** — Coding standards and conventions
