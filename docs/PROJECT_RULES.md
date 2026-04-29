# Flowker Project Rules

This document defines the specific standards for Flowker development, following Lerian Studio's Ring standards.

> **Reference**: This file is used by `/dev-refactor` and code review agents to validate compliance.

---

## Table of Contents

| # | Section | Description |
|---|---------|-------------|
| 1 | [Project Overview](#project-overview) | Project type and stack |
| 2 | [Version](#version) | Go version requirements |
| 3 | [Core Dependency: lib-commons](#core-dependency-lib-commons-mandatory) | Required foundation library |
| 4 | [Frameworks & Libraries](#frameworks--libraries) | Required packages and versions |
| 5 | [Data Stores](#data-stores) | Dual-database architecture (MongoDB + PostgreSQL) |
| 6 | [Authentication](#authentication) | Plugin Auth + API Key dual model |
| 7 | [Executor Catalog & Providers](#executor-catalog--providers) | Static catalog vs dynamic configuration |
| 8 | [Configuration](#configuration) | Environment variable handling |
| 9 | [Observability](#observability) | OpenTelemetry integration |
| 10 | [Bootstrap](#bootstrap) | Application initialization |
| 11 | [HTTP Responses](#http-responses) | Response standardization |
| 12 | [Pagination](#pagination) | Cursor-based pagination |
| 13 | [Middleware Order](#middleware-order) | Critical middleware sequence |
| 14 | [Data Transformation](#data-transformation-toentityfromentity-mandatory) | ToEntity/FromEntity patterns |
| 15 | [UUID Fields in Models](#uuid-fields-in-models) | UUID type usage |
| 16 | [Error Handling](#error-handling) | Business vs Technical errors, wrapping rules |
| 17 | [Context Cancellation Checks](#context-cancellation-checks-mandatory) | Check ctx.Err() before processing |
| 18 | [Input Normalization Order](#input-normalization-order-mandatory) | Normalize-Validate-Store pattern |
| 19 | [Sentinel Errors for Constructors](#sentinel-errors-for-constructors-mandatory) | Return errors, never panic |
| 20 | [Function Design](#function-design-mandatory) | Single responsibility principle |
| 21 | [Whitespace Style](#whitespace-style-wsl_v5) | Code formatting conventions |
| 22 | [Testing](#testing) | Table-driven tests, deterministic data, edge cases |
| 23 | [Shared Test Helpers](#shared-test-helpers) | Centralized test utilities |
| 24 | [Logging](#logging) | Structured logging with libLog.Any |
| 25 | [Linting](#linting) | golangci-lint configuration |
| 26 | [API Documentation](#api-documentation) | Swagger/OpenAPI generation |
| 27 | [Architecture Patterns](#architecture-patterns) | Hexagonal architecture |
| 28 | [Directory Structure](#directory-structure) | Project layout (Lerian pattern) |
| 29 | [File Naming Conventions](#file-naming-conventions-mandatory) | Go snake_case file naming |
| 30 | [Domain Models](#domain-models-mandatory) | Rich Domain Models (NOT anemic) |
| 31 | [Forbidden Practices](#forbidden-practices) | What NOT to do |

---

## Project Overview

**Flowker** is a workflow orchestration platform for financial validation.

| Attribute | Value |
|-----------|-------|
| **Language** | Go 1.25.8 |
| **Framework** | Fiber v2.52+ |
| **Primary Database** | MongoDB (workflows, executions, configurations) |
| **Audit Database** | PostgreSQL (hash-chained immutable audit log) |

---

## Version

- **Minimum**: Go 1.25.8
- **Recommended**: Latest stable release

---

## Core Dependency: lib-commons (MANDATORY)

All Flowker code **MUST** use `lib-commons/v4` as the foundation library. This is the current major version in production; do not downgrade to v2 or v3.

### Required Imports (lib-commons v4)

```go
import (
    libCommons "github.com/LerianStudio/lib-commons/v4/commons"
    libZap "github.com/LerianStudio/lib-commons/v4/commons/zap"                 // Logger initialization (bootstrap only)
    libLog "github.com/LerianStudio/lib-commons/v4/commons/log"                 // Logger interface (services, routes, handlers)
    libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
    libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
    libMongo "github.com/LerianStudio/lib-commons/v4/commons/mongo"
)
```

> Companion library for authentication: `github.com/LerianStudio/lib-auth/v2/auth/middleware` (see [Authentication](#authentication)).

### What lib-commons Provides

| Package | Purpose | Where Used |
|---------|---------|------------|
| `commons` | Core utilities, config loading, tracking context | Everywhere |
| `commons/zap` | Logger initialization/configuration | **Bootstrap files only** |
| `commons/log` | Logger interface (`libLog.Logger`) + structured fields (`libLog.Any`, `libLog.String`) | Services, routes, consumers, handlers |
| `commons/mongo` | MongoDB connection management, pagination | Bootstrap, repositories |
| `commons/opentelemetry` | OpenTelemetry initialization and helpers (`HandleSpanError`, `SetSpanAttributesFromStruct`) | Bootstrap, services |
| `commons/net/http` | HTTP utilities, telemetry middleware, response wrappers, pagination | Routes, handlers |

---

## Frameworks & Libraries

### Required Versions (Minimum)

| Library | Minimum Version | Purpose |
|---------|-----------------|---------|
| `lib-commons/v4` | v4.6.0-beta.7 | Core infrastructure |
| `lib-auth/v2` | v2.x | Access Manager plugin authentication |
| `fiber/v2` | v2.52.12 | HTTP framework |
| `go.mongodb.org/mongo-driver` | v1.17.9 | MongoDB driver |
| `github.com/jackc/pgx/v5` | v5.x | PostgreSQL driver (audit DB) |
| `go.opentelemetry.io/otel` | v1.43.0 | Telemetry |
| `zap` | v1.27.1 | Logging implementation |
| `testify` | v1.11.1 | Testing |
| `gomock` (`go.uber.org/mock`) | v0.6.0 | Mock generation |
| `validator/v10` | v10.30.2 | Input validation |

### Testing

| Library | Use Case |
|---------|----------|
| testify | Assertions |
| GoMock (`go.uber.org/mock`) | Interface mocking (MANDATORY for all mocks) |

---

## Data Stores

Flowker operates a **dual-database architecture**. Each database serves a distinct purpose, and code must not blur the boundary.

| Database | Purpose | Driver | Config Env |
|----------|---------|--------|------------|
| **MongoDB** | Operational state (workflows, executions, configurations) | `go.mongodb.org/mongo-driver` | `MONGO_URI`, `MONGO_DB_NAME`, `MONGO_TLS_CA_CERT` |
| **PostgreSQL** | Immutable, hash-chained audit trail | `github.com/jackc/pgx/v5` | `AUDIT_DB_HOST` (and related PostgreSQL vars) |

### MongoDB Collections (primary)

| Collection | Domain |
|------------|--------|
| `workflows` | Workflow definitions (Draft/Active/Archived) |
| `workflow_executions` | Execution records with step results |
| `provider_configurations` | Provider credentials and settings |
| `executor_configurations` | Executor-specific configuration entries |
| Dashboard collections | Aggregated views for analytics |

### PostgreSQL: Audit Trail

- `AUDIT_DB_HOST` is **mandatory** — bootstrap fails if unset (compliance requirement).
- Repository: `internal/adapters/postgresql/audit/repository.go`.
- The audit log is **hash-chained**: each entry references the previous entry's hash, and `VerifyAuditHashChainQuery` validates integrity.
- All mutating command handlers receive an `AuditWriter` (`command.AuditWriter`) and MUST record domain events (create, update, activate, deactivate, delete).
- Audit data is **write-once**: no updates, no deletes.

### Rules

1. **Do not store audit data in MongoDB** — use the PostgreSQL audit path exclusively.
2. **Do not cross-reference by foreign key between databases** — audit references workflow/execution IDs as opaque values.
3. **All commands that mutate domain state** MUST record an audit entry via `AuditWriter`.
4. **Audit failures are fatal** during bootstrap: the service refuses to start without audit connectivity.

---

## Authentication

Flowker uses a **dual authentication model**. Every protected route MUST be wrapped with `AuthGuard`, which selects the concrete middleware based on configuration.

| Endpoint Type | Primary Auth | Fallback | Examples |
|---------------|--------------|----------|----------|
| **Management** | Plugin Auth (Access Manager via `lib-auth/v2`) | API Key (if `API_KEY_ENABLED=true`) | Workflow CRUD, Provider/Executor config, Audit queries |
| **Execution** | API Key (forced) | Plugin Auth | `POST /v1/webhooks/:id`, `POST /v1/workflows/:id/execute` |

### Auth Priority Rules

1. If `PLUGIN_AUTH_ENABLED=true`, **plugin auth is primary** for management routes (resource/action authorization via Access Manager).
2. If `API_KEY_ENABLED=true`, **API key** is either a fallback OR forced (for execution routes that require M2M auth).
3. At least one of `PLUGIN_AUTH_ENABLED` or `API_KEY_ENABLED` must be effectively configured, otherwise `AuthGuard` returns nil and bootstrap fails.

### Usage Pattern

```go
import (
    httpMiddleware "github.com/LerianStudio/flowker/internal/adapters/http/in/middleware"
    authMiddleware "github.com/LerianStudio/lib-auth/v2/auth/middleware"
)

authClient := authMiddleware.NewAuthClient(cfg.PluginAuthAddress, cfg.PluginAuthEnabled, &logger)
authGuard := httpMiddleware.NewAuthGuard(httpMiddleware.AuthGuardConfig{
    APIKey:            cfg.APIKey,
    APIKeyEnabled:     cfg.APIKeyEnabled,
    PluginAuthEnabled: cfg.PluginAuthEnabled,
    AppName:           "flowker",
}, authClient)

// Management route (plugin auth if enabled, else API key)
router.Post("/v1/workflows", authGuard.Protect("workflows", "manage"), handler.Create)

// Execution route (force API key if enabled)
router.Post("/v1/webhooks/:id", authGuard.With("webhooks", "execute", true), handler.Trigger)
```

### Rules

- **Never** implement custom JWT validation in Flowker. Delegate to `lib-auth/v2`.
- **Never** hardcode API keys — use `API_KEY` env var loaded via the Config struct.
- Legacy `OIDC_*` environment variables are deprecated; `ValidateAccessManagerConfig` emits warnings if they are set.
- Bootstrap MUST call `ValidateAccessManagerConfig(cfg, logger)` before initializing routes.

---

## Executor Catalog & Providers

Flowker distinguishes **Providers** (static catalog, compiled into the binary) from **Executor Configurations** (dynamic records, stored in MongoDB).

| Concept | Type | Storage | Description | Example |
|---------|------|---------|-------------|---------|
| **Provider** | Static catalog entry | In-code (`pkg/executor/catalog.go` + `pkg/executors/*/provider.go`) | External service grouping | S3, Midaz, Tracer, HTTP |
| **Executor** | Runnable operation within a provider | In-code (registered on catalog) | Specific action | `s3.PutObject`, `midaz.CreateTransaction` |
| **Provider Configuration** | Dynamic credential record | MongoDB (`provider_configurations`) | Tenant/environment credentials | `midaz-production` credentials |
| **Executor Configuration** | Dynamic per-executor setting | MongoDB (`executor_configurations`) | Rate limits, timeouts, overrides | Midaz timeout = 10s |

### Catalog Bootstrap

```go
import (
    "github.com/LerianStudio/flowker/pkg/executor"
    "github.com/LerianStudio/flowker/pkg/executors"
    "github.com/LerianStudio/flowker/pkg/templates"
    "github.com/LerianStudio/flowker/pkg/triggers"
)

executorCatalog := executor.NewCatalog()

if err := executors.RegisterDefaults(executorCatalog); err != nil { return err }
if err := triggers.RegisterDefaults(executorCatalog); err != nil { return err }
if err := templates.RegisterDefaults(executorCatalog); err != nil { return err }
```

### Package Layout

| Path | Purpose |
|------|---------|
| `pkg/executor/` | Catalog, runner, runtime, executor/provider interfaces |
| `pkg/executors/` | Concrete providers: `http/`, `midaz/`, `s3/`, `tracer/` |
| `pkg/executors/http/auth/` | OIDC discovery + token fetching for HTTP-backed executors |
| `pkg/triggers/` | Trigger registrations (e.g., webhook, cron) |
| `pkg/templates/` | Workflow templates used by `CreateWorkflowFromTemplate` |
| `pkg/transformation/` | Input/output transformation DSL |
| `pkg/condition/` | Condition/branching evaluator |
| `pkg/circuitbreaker/` | Circuit breaker manager used by the executor runtime |
| `pkg/webhook/` | Webhook route registry (populated from active workflows at startup) |

### Rules

1. **Add new providers in `pkg/executors/<name>/`**, register them through `RegisterDefaults`.
2. **Never** look up provider credentials from environment variables inside an executor — credentials flow in via `ProviderConfiguration`.
3. **Dynamic data (credentials, rate limits)** belongs in MongoDB; **static capability metadata** belongs in the catalog.

---

## Configuration

All services **MUST** use `libCommons.SetConfigFromEnvVars` for configuration loading.

### Configuration Struct Pattern

```go
// internal/bootstrap/config.go
package bootstrap

const ApplicationName = "flowker"

type Config struct {
    // Application
    EnvName       string `env:"ENV_NAME"`
    ServerAddress string `env:"SERVER_ADDRESS"`
    LogLevel      string `env:"LOG_LEVEL"`

    // OpenTelemetry
    OtelServiceName         string `env:"OTEL_RESOURCE_SERVICE_NAME"`
    OtelLibraryName         string `env:"OTEL_LIBRARY_NAME"`
    OtelServiceVersion      string `env:"OTEL_RESOURCE_SERVICE_VERSION"`
    OtelDeploymentEnv       string `env:"OTEL_RESOURCE_DEPLOYMENT_ENVIRONMENT"`
    OtelColExporterEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
    EnableTelemetry         bool   `env:"ENABLE_TELEMETRY"`

    // MongoDB
    MongoURI       string `env:"MONGO_URI"`
    MongoDBName    string `env:"MONGO_DB_NAME"`
    MongoTLSCACert string `env:"MONGO_TLS_CA_CERT"`

    // Audit database (PostgreSQL)
    AuditDBHost string `env:"AUDIT_DB_HOST"`

    // Authentication
    APIKey             string `env:"API_KEY"`
    APIKeyEnabled      bool   `env:"API_KEY_ENABLED"`
    PluginAuthEnabled  bool   `env:"PLUGIN_AUTH_ENABLED"`
    PluginAuthAddress  string `env:"PLUGIN_AUTH_ADDRESS"`

    // HTTP
    CORSAllowedOrigins string `env:"CORS_ALLOWED_ORIGINS"`

    // Swagger
    SwaggerTitle       string `env:"SWAGGER_TITLE"`
    SwaggerDescription string `env:"SWAGGER_DESCRIPTION"`
    SwaggerVersion     string `env:"SWAGGER_VERSION"`
    SwaggerHost        string `env:"SWAGGER_HOST"`
    SwaggerBasePath    string `env:"SWAGGER_BASE_PATH"`
    SwaggerLeftDelim   string `env:"SWAGGER_LEFT_DELIM"`
    SwaggerRightDelim  string `env:"SWAGGER_RIGHT_DELIM"`
    SwaggerSchemes     string `env:"SWAGGER_SCHEMES"`

    // Feature flags
    SkipLibCommonsTelemetry bool `env:"SKIP_LIB_COMMONS_TELEMETRY"`
    FaultInjectionEnabled   bool `env:"FAULT_INJECTION_ENABLED"`
}
```

### Mandatory Validations

Bootstrap MUST fail fast on misconfiguration:

```go
// API_KEY_ENABLED=true requires API_KEY set
if cfg.APIKeyEnabled && cfg.APIKey == "" {
    return nil, fmt.Errorf("API_KEY_ENABLED=true requires API_KEY to be set")
}

// Plugin auth validation
if err := ValidateAccessManagerConfig(cfg, logger); err != nil {
    return nil, err
}

// Telemetry
if cfg.EnableTelemetry && cfg.OtelColExporterEndpoint == "" {
    return nil, fmt.Errorf("ENABLE_TELEMETRY=true requires OTEL_EXPORTER_OTLP_ENDPOINT to be set")
}

// Audit DB is mandatory (compliance)
if cfg.AuditDBHost == "" {
    return nil, fmt.Errorf("AUDIT_DB_HOST is required: audit trail is mandatory for compliance")
}
```

### What NOT to Do

```go
// FORBIDDEN: Manual os.Getenv calls scattered across code
uri := os.Getenv("MONGO_URI") // DON'T do this

// CORRECT: All configuration in Config struct, loaded once in bootstrap
type Config struct {
    MongoURI string `env:"MONGO_URI"` // Centralized
}
```

---

## Observability

All services **MUST** integrate OpenTelemetry using lib-commons.

### Service Method Instrumentation Checklist (MANDATORY)

**Every service method MUST implement these steps:**

| # | Step | Code Pattern | Purpose |
|---|------|--------------|---------|
| 1 | Extract tracking from context | `logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)` | Get logger/tracer injected by middleware |
| 2 | Create child span | `ctx, span := tracer.Start(ctx, "service.{domain}.{operation}")` | Create traceable operation |
| 3 | Defer span end | `defer span.End()` | Ensure span closes even on panic |
| 4 | Use structured logger | `logger.Log(ctx, libLog.LevelInfo, msg, libLog.Any("key", val))` | Logs correlated with trace |
| 5 | Handle business errors | `libOtel.HandleSpanBusinessErrorEvent(span, msg, err)` | Expected errors (validation, not found) |
| 6 | Handle technical errors | `libOtel.HandleSpanError(span, msg, err)` | Unexpected errors (DB, network) |
| 7 | Pass ctx downstream | All calls receive `ctx` with span | Trace propagation |

### Error Handling Classification

| Error Type | Examples | Handler Function | Span Status |
|------------|----------|------------------|-------------|
| **Business Error** | Validation failed, Resource not found, Conflict | `HandleSpanBusinessErrorEvent` | OK (adds event) |
| **Technical Error** | DB connection failed, Timeout, Network error | `HandleSpanError` | ERROR (records error) |

### Span Naming Conventions

| Layer | Pattern | Examples |
|-------|---------|----------|
| HTTP Handler | `handler.{resource}.{action}` | `handler.workflow.create` |
| Service / Command | `command.{domain}.{operation}` | `command.workflow.activate` |
| Query | `query.{domain}.{operation}` | `query.workflow.get_by_id` |
| Repository | `repository.{entity}.{operation}` | `repository.workflow.find_by_id` |

### Tracing with lib-commons (Required)

**Always use lib-commons wrappers** for OpenTelemetry operations.

**Allowed patterns:**

```go
import (
    libCommons "github.com/LerianStudio/lib-commons/v4/commons"
    libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
    "go.opentelemetry.io/otel/trace" // OK: Only for trace.Span type
)

// Creating spans
logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)
ctx, span := tracer.Start(ctx, "command.workflow.activate")
defer span.End()

// Setting span attributes (use SetSpanAttributesFromStruct)
_ = libOtel.SetSpanAttributesFromStruct(&span, "input_data", inputStruct)

// Handling errors
libOtel.HandleSpanError(span, "operation failed", err)              // Technical
libOtel.HandleSpanBusinessErrorEvent(span, "validation failed", err) // Business
```

### Instrumentation Anti-Patterns (FORBIDDEN)

| Anti-Pattern | Correct Pattern |
|--------------|-----------------|
| `import "go.opentelemetry.io/otel"` | Use `libCommons.NewTrackingFromContext(ctx)` |
| `otel.Tracer("name")` | Use tracer from `NewTrackingFromContext(ctx)` |
| `import "go.opentelemetry.io/otel/attribute"` | Use `libOtel.SetSpanAttributesFromStruct` |
| `import "go.opentelemetry.io/otel/codes"` | Use `libOtel.HandleSpanError` |
| `span.SetAttributes(attribute.String(...))` | Use `libOtel.SetSpanAttributesFromStruct` |
| `span.SetStatus(codes.Error, msg)` | Use `libOtel.HandleSpanError` |

---

## Bootstrap

All services **MUST** follow the bootstrap pattern for initialization.

### Directory Structure

```text
/internal
  /bootstrap
    config.go            # Config struct + InitServers()
    fiber.server.go      # HTTP server with graceful shutdown
    service.go           # Service struct + Run() method
    database.go          # MongoDB manager
    audit_database.go    # PostgreSQL audit manager
```

### main.go Pattern

`InitServers` returns `(*Service, error)` — the caller MUST handle the error.

```go
package main

import (
    "log"

    "github.com/LerianStudio/flowker/internal/bootstrap"
)

func main() {
    svc, err := bootstrap.InitServers()
    if err != nil {
        log.Fatalf("failed to initialize services: %v", err)
    }

    svc.Run()
}
```

### InitServers Responsibilities

In order:

1. Load `Config` via `libCommons.SetConfigFromEnvVars`.
2. Initialize zap logger via `libZap.New`.
3. Validate authentication configuration (`ValidateAccessManagerConfig`).
4. Validate telemetry configuration.
5. Initialize telemetry (`libOtel.NewTelemetry`).
6. Connect to MongoDB with a bounded `context.WithTimeout(30s)`.
7. Build the executor catalog and register defaults (executors, triggers, templates).
8. Initialize the PostgreSQL audit pipeline (fail-fast if `AUDIT_DB_HOST` is empty).
9. Initialize domain components (provider configs, workflows, executor configs, executions, dashboard, webhook).
10. Populate the webhook registry from active workflows (paginated read).
11. Wire `AuthGuard` and HTTP routes.
12. Return `(*Service, error)`.

---

## HTTP Responses

All HTTP responses **MUST** use `libHTTP` wrappers for consistent response format across all Lerian services.

### Response Methods

| Method | HTTP Status | When to Use |
|--------|-------------|-------------|
| `libHTTP.OK(c, data)` | 200 | Successful GET, PUT, PATCH |
| `libHTTP.Created(c, data)` | 201 | Successful POST (resource created) |
| `libHTTP.NoContent(c)` | 204 | Successful DELETE |
| `libHTTP.WithError(c, err)` | 4xx/5xx | Error responses |

### Forbidden Patterns

```go
// FORBIDDEN - Direct Fiber responses
c.JSON(status, data)           // Don't use
c.Status(code).JSON(err)       // Don't use
c.SendString(text)             // Don't use

// CORRECT - Use libHTTP wrappers
libHTTP.OK(c, data)
libHTTP.Created(c, data)
libHTTP.WithError(c, err)
```

### Handler Example

```go
import (
    libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
    "github.com/gofiber/fiber/v2"
)

func (h *WorkflowHandler) Create(c *fiber.Ctx) error {
    ctx := c.UserContext()

    var input model.CreateWorkflowInput
    if err := c.BodyParser(&input); err != nil {
        return libHTTP.WithError(c, err)
    }

    result, err := h.workflowService.Create(ctx, &input)
    if err != nil {
        return libHTTP.WithError(c, err)
    }

    return libHTTP.Created(c, result)
}
```

---

## Pagination

### Cursor-Based Pagination (Required)

All list/search endpoints **MUST** use cursor-based pagination for consistent results during navigation.

**Why cursor-based?**

- Consistent results when data changes during navigation
- Efficient for large datasets (no offset scanning)
- Better performance for real-time data

### Pagination Model

```go
// Filter/Input for list operations
type ListFilter struct {
    Limit     int    // Max items per page (1-100, default: 10)
    Cursor    string // Base64 encoded cursor (empty for first page)
    SortBy    string // Field to sort by (e.g., "created_at", "name")
    SortOrder string // "ASC" or "DESC"
}

// Result/Output for list operations
type ListResult[T any] struct {
    Items      []T    `json:"items"`
    NextCursor string `json:"nextCursor"` // Base64 encoded cursor for next page
    HasMore    bool   `json:"hasMore"`    // Indicates if there are more results
}
```

### Cursor Structure

The cursor contains all information needed to resume pagination consistently:

```go
// pkg/net/http/cursor.go
type Cursor struct {
    ID         string `json:"id"`  // ID of the last item returned
    SortValue  string `json:"sv"`  // Value of the sort field for the last item
    SortBy     string `json:"sb"`  // Field used for sorting
    SortOrder  string `json:"so"`  // Sort direction: "ASC" or "DESC"
    PointsNext bool   `json:"pn"`  // Direction indicator (true = next page)
}
```

### API Response Format

```json
{
    "items": [...],
    "nextCursor": "eyJpZCI6Ind...",
    "hasMore": true
}
```

### Usage Pattern

```go
// First page
GET /v1/workflows?limit=10

// Next page (use nextCursor from previous response)
GET /v1/workflows?limit=10&cursor=eyJpZCI6Ind...
```

### Implementation Notes

1. **Do NOT use offset/page-based pagination** - it causes inconsistent results
2. **Limit range**: 1-100 items per page (default: 10)
3. **Empty cursor** = first page
4. **Empty nextCursor** = no more results
5. **Cursor is opaque** - clients should not decode/modify it

---

## Middleware Order

The order of middleware registration is critical for proper telemetry, logging, and auth. Follow the exact order used in `internal/adapters/http/in/routes.go`:

```go
func NewRouter(
    lg libLog.Logger,
    tl *libOtel.Telemetry,
    cfg *RouteConfig,
    authGuard *httpMiddleware.AuthGuard,
    // ... handlers ...
) (*fiber.App, error) {
    f := fiber.New(fiber.Config{
        DisableStartupMessage: true,
        ErrorHandler: func(ctx *fiber.Ctx, err error) error {
            return libHTTP.HandleFiberError(ctx, err)
        },
    })

    tlMid := libHTTP.NewTelemetryMiddleware(tl)

    // 1. FIRST - injects tracer/logger into context
    f.Use(tlMid.WithTelemetry(tl))

    // 2. Panic recovery (must be early to catch panics from later middleware)
    f.Use(recover.New(recover.Config{EnableStackTrace: false}))

    // 3. CORS
    f.Use(cors.New(corsCfg))

    // 4. OpenTelemetry Fiber metrics (skip if SkipLibCommonsTelemetry is set)
    f.Use(otelfiber.Middleware(/* ... */))

    // 5. Client IP extraction (respects X-Forwarded-For)
    f.Use(middleware.ClientIPMiddleware())

    // 6. HTTP request logging (conditional)
    if !cfg.SkipLibCommonsTelemetry {
        f.Use(libHTTP.WithHTTPLogging(libHTTP.WithCustomLogger(lg)))
    }

    // 7. Fault injection (conditional, for chaos testing)
    if cfg.FaultInjectionEnabled {
        f.Use(middleware.FaultInjection(/* ... */))
    }

    // --- route definitions go here, each wrapped with authGuard.Protect / .With ---

    // LAST - closes root spans after the response is flushed
    f.Use(tlMid.EndTracingSpans)

    return f, nil
}
```

**Why order matters:**

- `WithTelemetry` must be first so tracer/logger are available for all subsequent middleware.
- `recover.New` must be early to catch panics from any middleware below.
- `ClientIPMiddleware` must run before any handler that logs or authorizes based on client IP.
- `EndTracingSpans` must be last to properly close spans after the response is sent.

---

## Data Transformation: ToEntity/FromEntity (MANDATORY)

All database models **MUST** implement transformation methods to/from domain entities.

```go
// ToEntity converts database model to domain entity
func (m *WorkflowMongoDBModel) ToEntity() *model.Workflow {
    return &model.Workflow{
        ID:        m.ID,
        Name:      m.Name,
        CreatedAt: m.CreatedAt,
    }
}

// FromEntity converts domain entity to database model
func (m *WorkflowMongoDBModel) FromEntity(w *model.Workflow) {
    m.ID = w.ID
    m.Name = w.Name
    m.CreatedAt = w.CreatedAt
}
```

---

## UUID Fields in Models

ID fields representing UUIDs must use `uuid.UUID` type (not `string`):

```go
import "github.com/google/uuid"

type Entity struct {
    ID        uuid.UUID  `json:"id" swaggertype:"string" format:"uuid"`
    ParentID  *uuid.UUID `json:"parentId,omitempty" swaggertype:"string" format:"uuid"`
}
```

**Rules:**

- Use `uuid.UUID` for ID fields, not `string`
- Add `swaggertype:"string" format:"uuid"` tags for proper OpenAPI documentation
- Use pointer (`*uuid.UUID`) for optional fields
- JSON unmarshaling handles string-to-UUID conversion automatically

---

## Error Handling

### Business vs Technical Errors (CRITICAL DISTINCTION)

Error handling differs by error type. **Business errors** are expected conditions; **technical errors** indicate infrastructure failures.

| Error Type | Examples | Wrapping Rule | Span Handler |
|------------|----------|---------------|--------------|
| **Business** | Validation failed, Not found, Conflict, Unauthorized | Return **directly** without wrapping | `HandleSpanBusinessErrorEvent` |
| **Technical** | DB failure, Network timeout, Connection refused | **Wrap with context** using `fmt.Errorf` | `HandleSpanError` |

### Business Errors - Return Directly

Business errors are expected and have well-defined constants. Do NOT wrap them — the error message is already descriptive:

```go
// CORRECT - business error returned directly
if errors.Is(err, constant.ErrEntityNotFound) {
    libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)

    return nil, err  // Return directly, no wrapping
}
```

### Technical Errors - Wrap with Context

Technical errors are unexpected infrastructure failures. ALWAYS wrap with `fmt.Errorf` and `%w` to provide stack context:

```go
// CORRECT - technical error wrapped with context
if err != nil {
    libOtel.HandleSpanError(span, "Failed to fetch workflow", err)

    return nil, fmt.Errorf("failed to fetch workflow: %w", err)
}
```

### Complete Error Handling Pattern

```go
func (s *ActivateWorkflowCommand) Execute(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
    logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)
    ctx, span := tracer.Start(ctx, "command.workflow.activate")
    defer span.End()

    workflow, err := s.repository.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, constant.ErrEntityNotFound) {
            // Business error - return directly
            libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)

            return nil, err
        }

        // Technical error - wrap with context
        libOtel.HandleSpanError(span, "Failed to get workflow", err)

        return nil, fmt.Errorf("failed to get workflow: %w", err)
    }

    if err := workflow.Activate(); err != nil {
        // Business error (invalid state transition) - return directly
        libOtel.HandleSpanBusinessErrorEvent(span, "Invalid state transition", err)

        return nil, err
    }

    if err := s.repository.Update(ctx, workflow); err != nil {
        // Technical error - wrap with context
        libOtel.HandleSpanError(span, "Failed to update workflow", err)

        return nil, fmt.Errorf("failed to update workflow: %w", err)
    }

    return workflow, nil
}
```

### Error Wrapping Rules Summary

```go
// ALWAYS use %w (not %v) to preserve error chain
return fmt.Errorf("failed to create workflow: %w", err)   // CORRECT
return fmt.Errorf("failed to create workflow: %v", err)   // WRONG - breaks errors.Is()

// Check specific errors with errors.Is
if errors.Is(err, ErrWorkflowNotFound) {
    return nil, err  // Business error - return directly
}
```

### Forbidden

```go
// NEVER use panic for business logic
panic(err) // FORBIDDEN

// NEVER ignore errors returned by functions.
// If the error cannot be handled, at least log it.
result, _ := doSomething()           // FORBIDDEN
_ = repo.Update(ctx, entity)         // FORBIDDEN (silently drops persistence failures)

// When an error truly cannot be propagated (e.g., best-effort persist
// in a cleanup path), log it so failures are observable:
if err := repo.Update(ctx, entity); err != nil {
    logger.Log(ctx, libLog.LevelError, "Failed to persist state (best-effort)",
        libLog.Any("entity.id", entity.ID()),
        libLog.Any("error.message", err.Error()),
    )
}

// NEVER wrap business errors (adds noise, no value)
return nil, fmt.Errorf("not found: %w", constant.ErrEntityNotFound)  // WRONG
return nil, constant.ErrEntityNotFound                                // CORRECT
```

---

## Context Cancellation Checks (MANDATORY)

**CRITICAL:** Check for context cancellation **at the very start** of service methods, **before** any validation or processing. This prevents wasted CPU cycles when the client has already timed out or cancelled the request.

```go
// WRONG - validates before checking cancellation (wastes CPU)
func (s *Service) Execute(ctx context.Context, input *Input) (*Output, error) {
    if err := input.Validate(); err != nil {
        return nil, err
    }

    if err := ctx.Err(); err != nil {
        return nil, err  // Too late!
    }
    // ...
}

// CORRECT - check cancellation FIRST
func (s *Service) Execute(ctx context.Context, input *Input) (*Output, error) {
    if err := ctx.Err(); err != nil {
        return nil, err  // Check BEFORE any work
    }

    if err := input.Validate(); err != nil {
        return nil, err
    }

    // ... rest of operation
}
```

**When to check:**

- **First line** of service methods (before validation)
- Before expensive operations (DB queries, external calls)
- In loops processing multiple items
- After long-running operations before continuing

---

## Input Normalization Order (MANDATORY)

Always follow this order when processing inputs in service methods:

```go
func (s *Service) Create(ctx context.Context, input *CreateInput) (*Output, error) {
    // 1. Check context cancellation
    if err := ctx.Err(); err != nil {
        return nil, err
    }

    // 2. Normalize input (trim whitespace, uppercase, etc.)
    input.Name = strings.TrimSpace(input.Name)

    // 3. Apply defaults (for optional fields)
    input.ApplyDefaults()

    // 4. Validate (after normalization and defaults)
    if err := input.Validate(); err != nil {
        return nil, err
    }

    // 5. Business logic
    // ...
}
```

**Order matters:** Validation MUST happen AFTER normalization and defaults are applied. Validating before normalizing can cause false rejections (e.g., rejecting `"  test  "` as invalid when trimmed it would be valid).

### Whitespace-Only Validation

Reject strings that contain only whitespace characters:

```go
// WRONG - only checks empty string
if name == "" {
    return ErrNameRequired
}

// CORRECT - rejects whitespace-only strings
if strings.TrimSpace(name) == "" {
    return ErrNameRequired
}
```

**Apply to:** Name fields, description fields, any user-provided text that is required.

---

## Sentinel Errors for Constructors (MANDATORY)

Constructors that receive dependencies **MUST** validate them and return errors instead of panicking:

```go
// Define sentinel errors for nil dependencies
var (
    ErrNilRepository = errors.New("repository cannot be nil")
    ErrNilClock      = errors.New("clock cannot be nil")
)

// WRONG - panics on nil
func NewService(repo Repository) *Service {
    if repo == nil {
        panic("repository cannot be nil")  // Don't panic
    }

    return &Service{repo: repo}
}

// CORRECT - returns error on nil
func NewService(repo Repository) (*Service, error) {
    if repo == nil {
        return nil, ErrNilRepository
    }

    return &Service{repo: repo}, nil
}
```

**Rules:**

- Use sentinel errors (e.g., `ErrNilRepository`, `ErrNilClock`) for nil dependency validation
- Constructor signature: `NewX(dep) (*X, error)` (NOT `NewX(dep) *X`)
- Test nil cases with `require.ErrorIs(t, err, ErrNilRepository)`
- Callers in bootstrap **MUST** propagate the error (return it from `InitServers`, do not panic)

```go
// Bootstrap - propagating constructor errors
cmd, err := command.NewCreateWorkflowCommand(repo, catalog, validator, clock, auditWriter)
if err != nil {
    logger.Log(ctx, libLog.LevelError, "Failed to create workflow command",
        libLog.Any("error.message", err.Error()))

    return nil, err
}
```

---

## Function Design (MANDATORY)

**Single Responsibility Principle (SRP):** Each function MUST have exactly ONE responsibility.

### Rules

| Rule | Description |
|------|-------------|
| **One responsibility per function** | A function should do ONE thing |
| **Max 20-30 lines** | If longer, break into smaller functions |
| **One level of abstraction** | Don't mix high-level and low-level operations |
| **Descriptive names** | Function name should describe its single responsibility |

---

## Whitespace Style (wsl_v5)

This project uses the `wsl_v5` linter (configured in `.golangci.yml`) to enforce whitespace conventions.

**Empty lines required before:**

- `return` statements (unless single-line block)
- `if`, `for`, `switch`, `select` blocks
- Assignments after different statement types

**Empty lines NOT allowed:**

- At the start of blocks (after `{`)
- At the end of blocks (before `}`)
- Multiple consecutive empty lines

```go
// WRONG - missing empty line before return
func example() error {
    result := doSomething()
    return result
}

// CORRECT - empty line before return
func example() error {
    result := doSomething()

    return result
}

// WRONG - empty line at start of block
func example() {

    doSomething()
}

// CORRECT - no empty line at start
func example() {
    doSomething()
}

// WRONG - cuddled if after assignment
func example() {
    result := getValue()
    if result > 0 {
        // ...
    }
}

// CORRECT - empty line before if
func example() {
    result := getValue()

    if result > 0 {
        // ...
    }
}
```

Run `make lint` to check for violations. Some issues can be auto-fixed with `golangci-lint run --fix`.

---

## Testing

### Table-Driven Tests (MANDATORY)

```go
func TestExecuteWorkflow(t *testing.T) {
    tests := []struct {
        name    string
        input   ExecuteWorkflowInput
        want    *WorkflowResult
        wantErr error
    }{
        {
            name:  "valid workflow",
            input: ExecuteWorkflowInput{ID: "wf-123"},
            want:  &WorkflowResult{Status: "completed"},
        },
        {
            name:    "workflow not found",
            input:   ExecuteWorkflowInput{ID: "invalid"},
            wantErr: ErrWorkflowNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ExecuteWorkflow(tt.input)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.want.Status, got.Status)
        })
    }
}
```

### Deterministic Test Data (MANDATORY)

**CRITICAL:** Never use `uuid.New()`, `time.Now()`, or any non-deterministic values in tests or test helpers. **New test code MUST follow this rule.** Existing occurrences are being migrated incrementally — do not introduce new ones.

```go
// WRONG - non-deterministic, hard to debug
workflow := &model.Workflow{
    ID:        uuid.New(),    // Random each run - FORBIDDEN
    CreatedAt: time.Now(),    // Different each run - FORBIDDEN
}

// CORRECT - deterministic, reproducible
workflow := &model.Workflow{
    ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),  // Fixed UUID
    CreatedAt: testutil.NewDefaultMockClock().Now(),                     // Fixed time
}
```

**Benefits:**
- Tests are reproducible across runs
- Easier to debug failures (same values every time)
- Consistent expected values in assertions
- Prevents flaky tests from timing issues

### Edge Case Coverage (MANDATORY)

| AC Type | Required Edge Cases | Minimum Count |
|---------|---------------------|---------------|
| Input validation | nil, empty string, boundary values, invalid format | 3+ |
| CRUD operations | not found, duplicate key, concurrent modification | 3+ |
| Business logic | zero value, negative numbers, boundary conditions | 3+ |
| Error handling | context timeout, connection refused, invalid response | 2+ |

### Boundary Value Tests

Always test boundary conditions for validation limits:

```go
func TestNewWorkflow(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        expectErr bool
    }{
        {
            name:      "name at max length is valid",
            input:     strings.Repeat("a", MaxNameLength),
            expectErr: false,
        },
        {
            name:      "name exceeds max length fails",
            input:     strings.Repeat("a", MaxNameLength+1),
            expectErr: true,
        },
    }
    // ...
}
```

**Always test:**
- Exactly at the limit (should pass)
- One over the limit (should fail)
- Zero/empty values
- Whitespace-only strings

### No-Mutation Assertions

Verify that failed operations do not partially mutate state:

```go
func TestWorkflow_Update_InvalidInput(t *testing.T) {
    wf := createValidWorkflow(t)

    // Capture original state
    originalName := wf.Name()
    originalStatus := wf.Status()

    // Attempt invalid update
    err := wf.Update(invalidInput)

    // Verify error occurred
    require.Error(t, err)

    // Verify NO partial mutation happened
    assert.Equal(t, originalName, wf.Name(), "name should not mutate on error")
    assert.Equal(t, originalStatus, wf.Status(), "status should not mutate on error")
}
```

### Slice Indexing Safety

Always use `require.Len` before indexing slices to prevent panics:

```go
// WRONG - can panic if slice is empty
assert.Equal(t, expectedID, workflows[0].ID)

// CORRECT - validate length first
require.Len(t, workflows, 1, "expected exactly one workflow")
assert.Equal(t, expectedID, workflows[0].ID)
```

### Error Comparison

Use `errors.Is` or `require.ErrorIs` instead of string matching:

```go
// WRONG - fragile string matching
assert.Contains(t, err.Error(), "not found")

// CORRECT - use sentinel errors
require.ErrorIs(t, err, constant.ErrEntityNotFound)
```

### Mock Generation (GoMock - MANDATORY)

```go
//go:generate mockgen -source=repository.go -destination=mocks/mock_repository.go -package=mocks
```

---

## Shared Test Helpers

Test helper functions must be placed in `internal/testutil/` package, not duplicated in each test file:

```go
// internal/testutil/helpers.go
package testutil

import "github.com/google/uuid"

// Ptr returns a pointer to any value. Generic helper for tests.
func Ptr[T any](v T) *T {
    return &v
}

// UUIDPtr returns a pointer to the given UUID.
func UUIDPtr(u uuid.UUID) *uuid.UUID {
    return &u
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string {
    return &s
}

// Int64Ptr returns a pointer to the given int64.
func Int64Ptr(i int64) *int64 {
    return &i
}
```

### Deterministic Time

Use `MockClock` for deterministic timestamps:

```go
// internal/testutil/mock_clock.go
clock := testutil.NewDefaultMockClock()  // Fixed: 2024-01-15 10:30:00 UTC
```

### No Local Duplicate Helpers

Do not create local helper functions when equivalent exists in `testutil`:

```go
// WRONG - local helper duplicating testutil.Ptr
func statusPtr(s model.WorkflowStatus) *model.WorkflowStatus { return &s }

// CORRECT - use generic helper
testutil.Ptr(model.WorkflowStatusActive)
testutil.Ptr("some string")
```

Usage in tests:

```go
import "github.com/LerianStudio/flowker/internal/testutil"

// In test:
input := &Input{
    AccountID: testutil.UUIDPtr(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")),
    Name:      testutil.StringPtr("test"),
    Status:    testutil.Ptr(model.WorkflowStatusDraft),
}
```

---

## Logging

**HARD GATE:** All Go services MUST use lib-commons structured logging via `logger.Log(ctx, level, msg, fields...)` with `libLog.Any`, `libLog.String`, etc. Never use `fmt.Printf`-style interpolation.

### FORBIDDEN Logging Patterns

| Pattern | Why FORBIDDEN |
|---------|---------------|
| `fmt.Println()` | No structure, no trace correlation |
| `fmt.Printf()` | No structure, no trace correlation |
| `log.Println()` | Standard library lacks trace correlation |
| `log.Printf()` | Standard library lacks trace correlation |
| `log.Fatal()` | Exits without graceful shutdown |
| `logger.Infof("msg: %s", val)` | String interpolation - not searchable/indexable |
| `logger.Errorf("failed: %v", err)` | String interpolation - not searchable/indexable |

### Required Pattern: `logger.Log` + `libLog.Any`

All logging **MUST** use `logger.Log(ctx, level, message, fields...)` with typed field helpers from `libLog`:

```go
import (
    libCommons "github.com/LerianStudio/lib-commons/v4/commons"
    libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
)

logger, _, _, _ := libCommons.NewTrackingFromContext(ctx)

// Log operation start
logger.Log(ctx, libLog.LevelInfo, "Creating workflow",
    libLog.Any("operation", "command.workflow.create"),
    libLog.Any("workflow.name", input.Name),
)

// Log success
logger.Log(ctx, libLog.LevelInfo, "Workflow created successfully",
    libLog.Any("operation", "command.workflow.create"),
    libLog.Any("workflow.id", result.ID().String()),
)

// Log errors (include error.message field)
logger.Log(ctx, libLog.LevelError, "Failed to create workflow",
    libLog.Any("operation", "command.workflow.create"),
    libLog.Any("error.message", err.Error()),
)
```

### Persistent Fields with `.With`

When the same field(s) apply to multiple log calls, use `.With` to create a scoped logger:

```go
logger.With(libLog.String("env.var", name)).Log(ctx, libLog.LevelWarn,
    "Deprecated OIDC env var is set but no longer used")
```

### Log Field Naming Convention

Field names **MUST** follow dot notation with lowercase:

| Field | Correct | Incorrect |
|-------|---------|-----------|
| Workflow ID | `workflow.id` | `workflowId`, `workflow_id` |
| Workflow name | `workflow.name` | `workflowName`, `name` |
| Error message | `error.message` | `error`, `err`, `errorMessage` |
| Operation | `operation` | `op`, `action` |
| Env var name | `env.var` | `envVar`, `env_var` |

### Operation Field Convention

The `operation` field **MUST** match the span name for correlation between logs and traces:

| Layer | Operation Value |
|-------|-----------------|
| Handler | `handler.workflow.create` |
| Service/Command | `command.workflow.create` |
| Query | `query.workflow.get` |
| Repository | `repository.workflow.find_by_id` |

### Log Message Guidelines

| Event | Message Pattern | Example |
|-------|-----------------|---------|
| Operation start | Present participle | "Creating workflow" |
| Operation success | Past tense + "successfully" | "Workflow created successfully" |
| Operation failure | "Failed to" + verb | "Failed to create workflow" |

### Level Selection

| Level | When to Use |
|-------|-------------|
| `libLog.LevelDebug` | Verbose diagnostic detail (disabled in production) |
| `libLog.LevelInfo` | Normal operational milestones (start/success) |
| `libLog.LevelWarn` | Degraded condition that did not fail (e.g., deprecated env var set) |
| `libLog.LevelError` | Operation failed |

> **Note**: There is no `FatalLog`; bootstrap propagates errors via `return nil, err` and `main.go` calls `log.Fatalf` only as a last resort to surface startup failure.

---

## Linting

The project uses `golangci-lint` v2 with a strict configuration. The full set of enabled linters lives in `.golangci.yml`. Key rules:

```yaml
# .golangci.yml (excerpt)
version: "2"
run:
  tests: false
linters:
  enable:
    - bodyclose
    - depguard
    - dogsled
    - dupword
    - errcheck
    - errchkjson
    - gocognit
    - gocyclo
    - gosec
    - govet
    - ineffassign
    - loggercheck
    - misspell
    - nakedret
    - nilerr
    - nolintlint
    - prealloc
    - predeclared
    - reassign
    - revive
    - staticcheck
    - thelper
    - tparallel
    - unconvert
    - unparam
    - unused
    - usestdlibvars
    - wastedassign
    - wsl_v5
  settings:
    gocognit: { min-complexity: 25 }
    gocyclo:  { min-complexity: 25 }
    wsl_v5:
      allow-first-in-block: true
      allow-whole-block: false
      branch-max-lines: 2
    depguard:
      rules:
        main:
          deny:
            - pkg: io/ioutil
              desc: Deprecated since Go 1.16
formatters:
  enable:
    - gofmt
    - goimports
```

**Rules:**

- **Cyclomatic/cognitive complexity** is capped at 25 (`gocyclo`, `gocognit`).
- **`wsl_v5`** enforces whitespace style (see [Whitespace Style](#whitespace-style-wsl_v5)).
- **`loggercheck`** validates structured logging key/value pairs.
- **`depguard`** forbids `io/ioutil` and similar deprecated packages.
- **Tests are excluded from some linters** via `exclusions.rules` (e.g., `errcheck`, `gosec` for `_test.go`).

Run `make lint` to apply checks with auto-fix where possible.

---

## API Documentation

This project uses **swaggo/swag** for OpenAPI/Swagger documentation generation.

**IMPORTANT:** Always use the Makefile to generate documentation:

```bash
make generate-docs
```

**Rules:**

- Documentation is generated in `api/` directory (not `docs/`)
- Never run `swag init` directly - always use `make generate-docs`
- Add swagger annotations to handler functions (see existing handlers for examples)
- Use `swaggertype` and `format` tags for proper type mapping (e.g., `swaggertype:"string" format:"uuid"` for `uuid.UUID` fields)

**Generated files:**

- `api/docs.go` - Go embed file
- `api/swagger.json` - OpenAPI 3.0 JSON spec
- `api/swagger.yaml` - OpenAPI 3.0 YAML spec

Access documentation at: `http://localhost:4000/swagger/index.html`

---

## Architecture Patterns

### Hexagonal Architecture (Ports & Adapters)

```text
/internal
  /bootstrap         # Application initialization
  /services          # Business logic
    /command         # Write operations
    /query           # Read operations
  /adapters          # Infrastructure implementations
    /http/in         # HTTP handlers + routes
    /mongodb         # MongoDB repositories
    /postgresql      # PostgreSQL repositories (audit)
```

### Interface-Based Abstractions

```go
// Define interface in the package that USES it
type WorkflowRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error)
    Save(ctx context.Context, workflow *model.Workflow) error
}

type UseCase struct {
    WorkflowRepo WorkflowRepository  // Depend on interface
}
```

---

## Directory Structure

```text
/cmd
  /app                     # Main application entry (main.go)
/internal
  /bootstrap               # Initialization
    config.go
    fiber.server.go
    service.go
    database.go            # MongoDB manager
    audit_database.go      # PostgreSQL audit manager
  /services                # Business logic
    /command               # Write operations (use cases)
    /query                 # Read operations (use cases)
  /adapters                # Infrastructure implementations
    /http/in               # HTTP handlers + routes
      /audit               # Audit query endpoints
      /catalog             # Executor catalog endpoints
      /dashboard           # Dashboard endpoints
      /execution           # Workflow execution endpoints
      /executor_configuration
      /health              # Liveness/readiness probes
      /middleware          # AuthGuard, ClientIP, FaultInjection, APIKey
      /provider_configuration
      /webhook             # Webhook trigger endpoints
      /workflow            # Workflow CRUD
      routes.go
      routes_cors.go
      swagger.go
    /mongodb               # MongoDB repositories
    /postgresql/audit      # PostgreSQL audit repository
  /testutil                # Shared test helpers (Ptr, MockClock, etc.)
/pkg
  /constant                # Constants and error codes
  /model                   # Shared domain models
  /net/http                # HTTP utilities (cursor pagination)
  /executor                # Executor catalog, runner, runtime, interfaces
  /executors               # Concrete providers: http, midaz, s3, tracer
  /triggers                # Trigger registrations (webhook, cron)
  /templates               # Workflow templates
  /transformation          # Input/output transformation DSL
  /condition               # Condition/branching evaluator
  /circuitbreaker          # Circuit breaker manager
  /webhook                 # Webhook route registry
  /pagination              # Pagination utilities
/api                       # Generated OpenAPI/Swagger specs
/docs                      # Project documentation (this file lives here)
```

---

## File Naming Conventions (MANDATORY)

All Go source files **MUST** use **snake_case** naming.

### Rules

| Type | Pattern | Example |
|------|---------|---------|
| **Regular files** | `snake_case.go` | `workflow.go`, `user_repository.go` |
| **Test files** | `*_test.go` | `workflow_test.go` |
| **Mock files** | `*_mock.go` | `workflow_mock.go` |

### Forbidden

```go
// FORBIDDEN: camelCase or PascalCase
WorkflowRepository.go
workflowRepository.go

// CORRECT: snake_case
workflow_repository.go
workflow.go
```

---

## Domain Models (MANDATORY)

**CRITICAL:** Flowker **MUST** use **Rich Domain Models**, NOT **Anemic Domain Models**.

### Why This Matters

Anemic Domain Models (structs with only data, no behavior) generate "useless" boilerplate code whose sole purpose is validating object state consistency. This pattern was identified as a problem in the Tracer project and **MUST be avoided** in Flowker.

### Rich Domain Model Pattern

| Aspect | Anemic Model | Rich Domain Model |
|--------|--------------|-------------------|
| **Validation** | External functions | Built into constructor |
| **State changes** | Direct field assignment | Methods with validation |
| **Consistency** | Not guaranteed | Always consistent |
| **Behavior** | In separate services | In the model itself |

### Implementation Pattern

```go
// FORBIDDEN: Anemic Model (data bag + external validation)
type Workflow struct {
    ID     string
    Name   string
    Status string
    Steps  []Step
}

// External validation - generates useless boilerplate
func ValidateWorkflow(w *Workflow) error {
    if w.Name == "" {
        return errors.New("name is required")
    }
    if len(w.Steps) == 0 {
        return errors.New("at least one step is required")
    }
    return nil
}

// Usage requires remembering to validate
w := &Workflow{Name: "test"}
if err := ValidateWorkflow(w); err != nil {  // Easy to forget!
    return err
}
```

```go
// CORRECT: Rich Domain Model (validation in constructor)
type Workflow struct {
    id     string  // private fields
    name   string
    status WorkflowStatus
    steps  []Step
}

// Constructor validates and returns consistent object
func NewWorkflow(name string, steps []Step) (*Workflow, error) {
    if name == "" {
        return nil, errors.New("name is required")
    }
    if len(steps) == 0 {
        return nil, errors.New("at least one step is required")
    }

    return &Workflow{
        id:     uuid.New().String(),
        name:   name,
        status: WorkflowStatusDraft,
        steps:  steps,
    }, nil
}

// Getters expose data
func (w *Workflow) ID() string { return w.id }
func (w *Workflow) Name() string { return w.name }
func (w *Workflow) Status() WorkflowStatus { return w.status }

// Methods encapsulate state changes with validation
func (w *Workflow) Activate() error {
    if w.status != WorkflowStatusDraft {
        return errors.New("can only activate draft workflows")
    }
    w.status = WorkflowStatusActive

    return nil
}
```

### When to Apply

| Model Type | Pattern to Use |
|------------|----------------|
| **Domain entities** (Workflow, Provider, Execution) | Rich Domain Model with constructor validation |
| **Input DTOs** (CreateWorkflowInput) | Use `validator` tags + struct validation |
| **Output DTOs** (WorkflowOutput) | Simple struct (read-only, no validation needed) |
| **Database models** (WorkflowMongoDBModel) | Simple struct with ToEntity/FromEntity |

### Validate-Before-Mutate (MANDATORY)

Methods that change state **MUST** validate ALL inputs before applying ANY changes. Failed validations must leave the object unchanged (no partial mutations):

```go
// WRONG - partial mutation on validation failure
func (w *Workflow) Update(name string, nodes []Node) error {
    w.name = strings.TrimSpace(name)  // Mutated!

    if w.name == "" {
        return ErrNameRequired  // BUG: name was mutated even though validation failed
    }

    w.nodes = nodes  // Mutated!

    if len(nodes) == 0 {
        return ErrNodesRequired  // BUG: both fields mutated before validation completed
    }

    return nil
}

// CORRECT - validate first, then mutate atomically
func (w *Workflow) Update(name string, nodes []Node) error {
    // Normalize inputs (does not mutate state)
    normalizedName := strings.TrimSpace(name)

    // Validate ALL invariants before ANY mutation
    if normalizedName == "" {
        return ErrNameRequired
    }

    if len(nodes) == 0 {
        return ErrNodesRequired
    }

    // All validations passed - now mutate atomically
    w.name = normalizedName
    w.nodes = cloneNodes(nodes)  // Defensive copy
    w.updatedAt = time.Now()

    return nil
}
```

### Validation Rules

1. **Domain entities**: Validation in constructor (`NewWorkflow(...)`)
2. **Input DTOs**: Use `validate` tags with `validator/v10`
3. **State transitions**: Validate-Before-Mutate pattern (validate all, then apply all)
4. **Immutable after creation**: Consider making fields private with getters
5. **Defensive copies**: Always copy slices and maps to prevent external mutation

### Benefits

- **Impossible to create invalid objects** - validation at construction
- **Self-documenting** - behavior is in the model, not scattered
- **No forgotten validations** - constructor enforces rules
- **No partial mutations** - failed operations leave state unchanged
- **Easier testing** - model is self-contained unit
- **Less boilerplate** - no external validation functions

---

## Forbidden Practices

1. **Direct database access from handlers** - Always go through services
2. **Business logic in repositories** - Repositories are for data access only
3. **Hardcoded configuration** - Use environment variables via Config struct
4. **Ignoring errors** - All errors must be handled:

   ```go
   // FORBIDDEN
   result, _ := someFunction()
   _ = anotherFunction()

   // CORRECT
   result, err := someFunction()
   if err != nil {
       return fmt.Errorf("someFunction failed: %w", err)
   }

   if err := anotherFunction(); err != nil {
       logger.Log(ctx, libLog.LevelError, "anotherFunction failed",
           libLog.Any("error.message", err.Error()))
   }
   ```

5. **Missing context propagation** - Always pass context through layers
6. **Missing tracing** - All service operations must have tracing spans
7. **Tests without mocks** - Service tests must mock dependencies
8. **Cyclomatic complexity > 25** - Refactor complex functions
9. **Direct Fiber responses** - Use libHTTP wrappers (`libHTTP.OK`, `libHTTP.Created`, etc.)
10. **Direct OTel imports in app code** - Use lib-commons wrappers (`SetSpanAttributesFromStruct`, `HandleSpanError`)
11. **`fmt.Println`/`log.Printf`/`logger.Infof`** - Use structured `logger.Log(ctx, level, msg, libLog.Any(...))`
12. **panic for business logic** - Only for truly unrecoverable errors (avoid in bootstrap too — return errors)
13. **offset/page-based pagination** - Use cursor-based pagination only
14. **Anemic Domain Models** - Use Rich Domain Models with validation in constructors (see [Domain Models](#domain-models-mandatory))
15. **Wrapping business errors** - Return business errors directly, only wrap technical errors (see [Error Handling](#error-handling))
16. **Skipping context cancellation check** - Always check `ctx.Err()` at the start of service methods (see [Context Cancellation](#context-cancellation-checks-mandatory))
17. **Panicking in constructors** - Use sentinel errors and return `(*T, error)` (see [Sentinel Errors](#sentinel-errors-for-constructors-mandatory))
18. **String interpolation in logs** - Use `libLog.Any`/`libLog.String` fields (see [Logging](#logging))
19. **Non-deterministic test data** - Never use `uuid.New()` or `time.Now()` in tests (see [Testing](#testing))
20. **Partial mutations on error** - Validate all inputs before applying any state changes (see [Domain Models](#domain-models-mandatory))
21. **Storing audit data in MongoDB** - Audit trail goes exclusively to PostgreSQL via `AuditWriter` (see [Data Stores](#data-stores))
22. **Custom JWT validation** - Delegate to `lib-auth/v2` via `AuthGuard` (see [Authentication](#authentication))
23. **Looking up provider credentials from env vars inside executors** - Credentials flow via `ProviderConfiguration` (see [Executor Catalog](#executor-catalog--providers))
24. **Using `lib-commons/v2` or `lib-commons/v3`** - Flowker is on `lib-commons/v4` (see [Core Dependency](#core-dependency-lib-commons-mandatory))

---

## Checklist

Before submitting code, verify:

- [ ] All configuration loaded via `libCommons.SetConfigFromEnvVars`
- [ ] Logger initialized with `libZap.New` (bootstrap only)
- [ ] All service methods have proper span instrumentation (`NewTrackingFromContext` + `tracer.Start` + `defer span.End()`)
- [ ] No `fmt.Println`, `log.Printf`, `logger.Infof`, or `panic` in business logic
- [ ] Business errors returned directly, technical errors wrapped with `fmt.Errorf`
- [ ] Context cancellation checked at the start of service methods (`ctx.Err()`)
- [ ] Input normalization before validation (trim, defaults, then validate)
- [ ] Constructors return `(*T, error)` with sentinel errors for nil dependencies
- [ ] Bootstrap propagates constructor errors (returns `(nil, err)`, no panic)
- [ ] Table-driven tests with deterministic data (no `uuid.New()` or `time.Now()`)
- [ ] Boundary value tests and no-mutation assertions for domain models
- [ ] Mocks generated with GoMock (`go.uber.org/mock`)
- [ ] Structured logging uses `logger.Log(ctx, level, msg, libLog.Any(...))`
- [ ] HTTP responses use `libHTTP.OK()`, `libHTTP.Created()`, `libHTTP.WithError()`
- [ ] List endpoints use cursor-based pagination
- [ ] Middleware registered in correct order (Telemetry → Recover → CORS → Otel → ClientIP → Logging → … → EndTracingSpans)
- [ ] Protected routes wrapped with `authGuard.Protect` or `authGuard.With`
- [ ] Mutating commands record an audit entry via `AuditWriter` (PostgreSQL)
- [ ] UUID fields use `uuid.UUID` type with swagger tags
- [ ] Domain entities use Rich Domain Model with Validate-Before-Mutate pattern
- [ ] New executors registered in `pkg/executors/` and wired via `RegisterDefaults`
- [ ] All imports of `lib-commons` point to `/v4/` (no `/v2/` or `/v3/`)
