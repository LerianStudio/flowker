# CLAUDE.md

> **Note**: AGENTS.md is a symlink to this file. Edit CLAUDE.md only; changes propagate automatically.

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Flowker** is a workflow orchestration platform for financial validation. It enables financial institutions to validate transactions through external providers (KYC, AML, fraud detection) **before** writing to core banking ledgers, ensuring compliance and providing complete audit trails.

**Language**: Go 1.25.5
**Framework**: Fiber v2.52.10
**Database**: MongoDB

### Provider vs Executor Terminology

| Concept | Type | Storage | Description | Example |
|---------|------|---------|-------------|---------|
| **Provider** | Static catalog | In-code (`pkg/executor/catalog.go` pattern) | External service grouping | S3, Midaz, Tracer, KYC |
| **Executor** | Dynamic config | MongoDB (`executor_configurations`) | Specific operation within a provider | S3.PutObject, Midaz.CreateTransaction |

Providers group executors. A Provider (e.g., S3) has multiple Executors (e.g., PutObject, GetObject). Providers are registered statically in code; Executors are configured dynamically via API/DB.

## Common Commands

```bash
# Development setup
make dev-setup          # Install tools, setup .env, run tests
make setup-git-hooks    # Install pre-commit hooks

# Building and running
make build              # Build the component
make run                # Run with .env configuration
make up                 # Start services with Docker Compose
make down               # Stop services
make rebuild-up         # Rebuild and restart (useful during development)

# Testing
make test               # Run all tests with verbose output
make cover-html         # Generate HTML coverage report in artifacts/coverage.html
go test -v ./internal/services/command/...   # Run tests for specific package

# Code quality
make lint               # Run golangci-lint with auto-fix
make format             # Format code (go fmt)
make tidy               # Update dependencies (go mod tidy)
make sec                # Security checks with gosec

# Documentation
make generate-docs      # Generate Swagger/OpenAPI documentation
make validate-api-docs  # Validate API documentation

# Docker
make logs               # Show logs for all services
make logs-api           # Show logs for main service
make ps                 # List container status
```

## Architecture

### Hexagonal Architecture (Ports & Adapters)

```
internal/
├── adapters/
│   ├── http/in/        # HTTP handlers and routes (Fiber)
│   ├── grpc/in/        # gRPC service handlers
│   └── mongodb/        # MongoDB repository implementations
├── bootstrap/          # Configuration, server setup, dependency injection
└── services/
    ├── command/        # Write operations (CQRS commands)
    └── query/          # Read operations (CQRS queries)

pkg/
├── model/              # Domain models with Swagger annotations
├── proto/              # Protocol Buffer definitions
├── constant/           # Error codes, pagination constants
└── net/http/           # HTTP utilities (cursor pagination)
```

### CQRS Pattern

- **Commands** (`internal/services/command/`): CreateExample, UpdateExample, DeleteExample
- **Queries** (`internal/services/query/`): GetExampleByID, GetAllExamples

### Repository Pattern

Each repository follows this structure in `internal/adapters/mongodb/`:
- `example.go` - Interface definition
- `example.mongodb.go` - MongoDB implementation
- `example.mongodb.mock.go` - Mock for testing

## Key Patterns

### Rich Domain Models (MANDATORY)

**CRITICAL:** Use **Rich Domain Models**, NOT **Anemic Domain Models**.

```go
// ❌ FORBIDDEN: Anemic Model (data bag + external validation)
type Workflow struct {
    ID, Name, Status string
}
func ValidateWorkflow(w *Workflow) error { ... }  // External validation = bad

// ✅ CORRECT: Rich Domain Model (validation in constructor)
type Workflow struct {
    id, name string  // private fields
    status   WorkflowStatus
}

func NewWorkflow(name string, steps []Step) (*Workflow, error) {
    if name == "" {
        return nil, errors.New("name is required")
    }
    return &Workflow{id: uuid.New().String(), name: name, status: Draft}, nil
}

func (w *Workflow) Name() string { return w.name }  // Getters for access
func (w *Workflow) Activate() error { ... }         // Methods for state changes
```

See `docs/PROJECT_RULES.md` for complete guidelines.

### Error Handling

Use custom error types from `pkg/errors.go`:
- `EntityNotFoundError` - For missing entities
- `ValidationError` - For validation failures

### Validation

Uses `go-playground/validator/v10` with struct tags for **Input DTOs only**:
```go
// Input DTOs use validator tags
type CreateExampleInput struct {
    Name string `json:"name" validate:"required,max=256"`
    Age  int    `json:"age" validate:"required"`
}
```

### Swagger Annotations

Add to handlers in `internal/adapters/http/in/`:
```go
// @Summary      Create an Example
// @Tags         Example
// @Accept       json
// @Produce      json
// @Param        example  body  model.CreateExampleInput  true  "Example Input"
// @Success      200  {object}  model.ExampleOutput
// @Router       /v1/example [post]
```

## Commit Message Format

Git hooks enforce this format (protected branches: main, develop, release/*):

```
type(scope): description

type: feature|fix|refactor|style|test|docs|build
scope: 1-20 characters
description: 1-100 characters
```

Examples:
- `feature(auth): implement JWK authentication`
- `fix(workflow): resolve execution state persistence`

## Database

- **MongoDB**: Document-oriented database for all operations (replica set)
- **Connection**: Configured via `MONGODB_URI` environment variable
- **Collections**: workflows, executor_configurations, workflow_executions, audit_entries

## Authentication

Flowker uses a **dual authentication model**:

| Endpoint Type | Auth Method | Examples |
|---------------|-------------|----------|
| **Management** | OIDC/JWT (external IdP) | Workflow CRUD, Executor CRUD, Audit queries |
| **Execution** | API Key | ExecuteWorkflow, GetExecutionStatus |

- **OIDC**: JWT tokens validated via external IdP's JWKS endpoint (e.g., Keycloak)
- **API Key**: Created via Management API, used for M2M workflow execution

## Testing

- Framework: `stretchr/testify` for assertions
- Mocking: `go.uber.org/mock`
- Test files: `*_test.go` alongside implementations
- Coverage ignore list: `scripts/coverage_ignore.txt`

## Key Dependencies

- `github.com/LerianStudio/lib-commons/v2` - Core utilities (MongoDB, logging, telemetry)
- `github.com/go-playground/validator/v10` - Input validation
- `go.mongodb.org/mongo-driver` - MongoDB driver
- `go.opentelemetry.io/otel` - Observability (tracing, metrics, logging)

## Documentation

Detailed planning and requirements are in `docs/pre-dev/`:
- `prd.md` - Product Requirements
- `trd.md` - Technical Requirements
- `data-model.md` - Entity definitions and relationships
- `api-design.md` - API specifications
- `tasks.md` - Implementation task breakdown
