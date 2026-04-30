# Flowker Changelog

## [1.0.2](https://github.com/LerianStudio/flowker/releases/tag/v1.0.2)

- Implemented canonical /readyz endpoint for health checks.
- Upgraded lib-commons to v5 and lib-auth to v2.7.0.
- Triggered gitops update on all tags.
- Allowed "ci" scope in PR titles.
- Removed trailing blank lines in Docker configuration.

Contributors: @bedatty, @lerian-studio, @lffranca

[Compare changes](https://github.com/LerianStudio/flowker/compare/v1.0.1...v1.0.2)

---

## [1.0.1](https://github.com/LerianStudio/flowker/releases/tag/v1.0.1)

- Improvements:
  - Bumped shared-workflows to version 1.26.3 to enhance CI processes.
  - Triggered workflows to ensure updated configurations are applied.

Contributors: @bedatty,

[Compare changes](https://github.com/LerianStudio/flowker/compare/v1.0.0...v1.0.1)

---

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

