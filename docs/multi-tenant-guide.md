# Multi-Tenant Activation Guide

## Overview

Flowker supports multi-tenant mode using database-per-tenant isolation. When enabled, each tenant has isolated MongoDB and PostgreSQL databases, with `tenantId` extracted from JWT tokens and resolved via the Tenant Manager API.

## Components

| Component | Service | Module | Resources |
|-----------|---------|--------|-----------|
| Flowker API | flowker | manager | MongoDB, PostgreSQL (audit) |

## Environment Variables

### Multi-Tenant Configuration (14 canonical env vars)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `MULTI_TENANT_ENABLED` | bool | `false` | No | Enable multi-tenant mode |
| `MULTI_TENANT_URL` | string | - | When enabled | Tenant Manager API URL |
| `MULTI_TENANT_REDIS_HOST` | string | - | When enabled | Redis host for Pub/Sub event-driven tenant discovery |
| `MULTI_TENANT_REDIS_PORT` | string | `6379` | No | Redis port for Pub/Sub |
| `MULTI_TENANT_REDIS_PASSWORD` | string | - | No | Redis password for Pub/Sub |
| `MULTI_TENANT_REDIS_TLS` | bool | `false` | No | Enable TLS for Pub/Sub Redis connection |
| `MULTI_TENANT_MAX_TENANT_POOLS` | int | `100` | No | Max concurrent tenant connection pools |
| `MULTI_TENANT_IDLE_TIMEOUT_SEC` | int | `300` | No | Idle pool timeout in seconds |
| `MULTI_TENANT_TIMEOUT` | int | `30` | No | HTTP client timeout for Tenant Manager API calls |
| `MULTI_TENANT_CIRCUIT_BREAKER_THRESHOLD` | int | `5` | No | Circuit breaker failure threshold |
| `MULTI_TENANT_CIRCUIT_BREAKER_TIMEOUT_SEC` | int | `30` | No | Circuit breaker timeout in seconds |
| `MULTI_TENANT_SERVICE_API_KEY` | string | - | When enabled | API key for Tenant Manager /settings endpoint |
| `MULTI_TENANT_CACHE_TTL_SEC` | int | `120` | No | In-memory cache TTL for tenant config |
| `MULTI_TENANT_CONNECTIONS_CHECK_INTERVAL_SEC` | int | `30` | No | PostgreSQL settings revalidation interval |

## How to Activate

### 1. Configure Environment Variables

Add the following to your `.env` or deployment configuration:

```bash
# Enable multi-tenant mode
MULTI_TENANT_ENABLED=true

# Tenant Manager connection
MULTI_TENANT_URL=http://tenant-manager:8080
MULTI_TENANT_SERVICE_API_KEY=your-api-key

# Redis for event-driven tenant discovery
MULTI_TENANT_REDIS_HOST=redis
MULTI_TENANT_REDIS_PORT=6379
MULTI_TENANT_REDIS_TLS=false

# Optional: tune connection pools
MULTI_TENANT_MAX_TENANT_POOLS=100
MULTI_TENANT_IDLE_TIMEOUT_SEC=300
MULTI_TENANT_CACHE_TTL_SEC=120
```

### 2. Ensure Tenant Manager is Running

Flowker requires the Tenant Manager API to be available at `MULTI_TENANT_URL`. The Tenant Manager provides:
- Tenant database credentials via `/settings` endpoint
- Event-driven tenant discovery via Redis Pub/Sub

### 3. Start Flowker

```bash
# With docker-compose
make up

# Or locally
make run
```

### 4. Verify Activation

Check the startup logs for:

```
INFO  Multi-tenant mode enabled  url=http://tenant-manager:8080
INFO  TenantMiddleware initialized  postgres=true mongodb=true
INFO  Event-driven tenant discovery started
```

## How to Verify

### 1. Check Logs

When multi-tenant mode is active, you should see:
- `Multi-tenant mode enabled` at startup
- `Tenant middleware initialized` with resource managers
- Per-request logs with `tenant_id` field

### 2. Test with JWT

Make a request with a valid JWT containing `tenant_id` claim:

```bash
curl -H "Authorization: Bearer <jwt-with-tenant-id>" \
     http://localhost:4021/v1/workflows
```

The request should be routed to the tenant's isolated database.

### 3. Check Metrics

Multi-tenant metrics are exposed at `/metrics`:

```bash
curl http://localhost:4021/metrics | grep tenant_
```

Expected metrics:
- `tenant_connections_total{tenant_id="..."}` - Connection count per tenant
- `tenant_connection_errors_total{tenant_id="...", error_type="..."}` - Connection errors
- `tenant_consumers_active{tenant_id="..."}` - Active consumers (if applicable)
- `tenant_messages_processed_total{tenant_id="..."}` - Processed messages (if applicable)

## How to Deactivate

Set `MULTI_TENANT_ENABLED=false` (or remove the variable):

```bash
MULTI_TENANT_ENABLED=false
```

Flowker will start in single-tenant mode with the original static database connections.

## Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `tenant mongodb connection missing from context` | JWT missing `tenant_id` claim or TenantMiddleware not registered | Ensure JWT contains `tenant_id` and TenantMiddleware is in the route chain |
| `tenant postgres connection missing from context` | Same as above | Same as above |
| `failed to resolve tenant settings` | Tenant Manager unreachable or invalid API key | Check `MULTI_TENANT_URL` and `MULTI_TENANT_SERVICE_API_KEY` |
| `circuit breaker open` | Tenant Manager has been failing | Check Tenant Manager health; wait for circuit breaker to close |
| `tenant not found` | Tenant not provisioned in Tenant Manager | Provision the tenant via Tenant Manager API |

## Backward Compatibility

When `MULTI_TENANT_ENABLED=false` (default):
- No tenant middleware is registered
- No JWT parsing or tenant resolution occurs
- All database connections use the original static constructors
- All existing tests pass unchanged
- `go test ./...` works without any multi-tenant infrastructure

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Request + JWT                        │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Auth Guard (validates JWT)                  │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│  TenantMiddleware (extracts tenantId, resolves DB connections)   │
│                                                                  │
│  - Parses tenantId from JWT claims                               │
│  - Calls Tenant Manager /settings for tenant config              │
│  - Resolves per-tenant MongoDB + PostgreSQL connections          │
│  - Stores connections in request context                         │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Route Handlers                            │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Repositories                             │
│                                                                  │
│  - tmcore.GetMBContext(ctx) → tenant's MongoDB                   │
│  - tmcore.GetPGContext(ctx) → tenant's PostgreSQL                │
└─────────────────────────────────────────────────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    ▼                         ▼
        ┌───────────────────┐     ┌───────────────────┐
        │ tenant-123-flowker│     │ tenant-456-flowker│
        │    (MongoDB)      │     │    (MongoDB)      │
        └───────────────────┘     └───────────────────┘
                    │                         │
        ┌───────────────────┐     ┌───────────────────┐
        │ tenant-123-audit  │     │ tenant-456-audit  │
        │   (PostgreSQL)    │     │   (PostgreSQL)    │
        └───────────────────┘     └───────────────────┘
```

## Dependencies

| Package | Import Path | Purpose |
|---------|-------------|---------|
| tmmiddleware | `lib-commons/v5/commons/tenant-manager/middleware` | TenantMiddleware with WithPG/WithMB |
| tmclient | `lib-commons/v5/commons/tenant-manager/client` | Tenant Manager HTTP client |
| tmcore | `lib-commons/v5/commons/tenant-manager/core` | GetPGContext, GetMBContext helpers |
| tmmongo | `lib-commons/v5/commons/tenant-manager/mongo` | MongoDB Manager |
| tmpostgres | `lib-commons/v5/commons/tenant-manager/postgres` | PostgreSQL Manager |
| tmredis | `lib-commons/v5/commons/tenant-manager/redis` | Tenant Pub/Sub Redis client |
| tmevent | `lib-commons/v5/commons/tenant-manager/event` | Event-driven tenant discovery |
