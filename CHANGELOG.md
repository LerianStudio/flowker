# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Features
- Initial Flowker project setup
- Hexagonal architecture with CQRS pattern
- MongoDB integration with replica set support
- PostgreSQL-backed, hash-chained audit trail
- Dual authentication model: Access Manager plugin (lib-auth/v2) + API Key
- Executor catalog with HTTP, Midaz, S3, and Tracer providers
- Workflow templates, triggers (including webhooks), and transformation DSL
- Circuit breaker and condition evaluator for the execution runtime
- OpenTelemetry observability (traces, metrics, logs)
- Swagger/OpenAPI documentation
- Health and version endpoints
- Provider Configuration CRUD with full lifecycle management
  - Status transitions: unconfigured → configured → tested → active → disabled
  - Support for multiple authentication types (none, api_key, bearer, basic, oidc)
- Cursor-based pagination across list endpoints

### Refactoring
- Flat CQRS services package with thin facades
- Individual command/query files for better maintainability
- Clock interface injection for improved testability

### Testing
- Unit tests for command/query packages with gomock
- Integration tests for Provider Configuration and Workflow APIs

### Maintenance
- Project scaffolding based on lib-commons v4
- Docker Compose configuration for local development
