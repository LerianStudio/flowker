# Flowker Readyz Activation Guide

## Overview

Flowker implements the canonical `/readyz` readiness probe that checks **MongoDB** and **PostgreSQL** (audit database) connectivity. The endpoint returns `200 OK` when all dependencies are reachable and `503 Service Unavailable` when any dependency is down. This enables Kubernetes to make accurate routing decisions and prevents traffic from reaching pods with unreachable databases.

**Scope fence:** `/readyz` is an infrastructure probe. It does NOT include synthetic business-logic checks, certificate validity validation, or performance SLIs.

---

## Endpoints

| Endpoint | Purpose | K8s Probe | Auth Required |
|----------|---------|-----------|---------------|
| `/readyz` | Readiness probe — checks all dependencies | `readinessProbe` | No |
| `/health` | Liveness probe — gated by startup self-probe | `livenessProbe` | No |
| `/metrics` | Prometheus scrape — includes readyz metrics | N/A | No |

Both `/readyz` and `/health` are mounted **before** authentication middleware, allowing Kubernetes probes to function without credentials.

---

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DEPLOYMENT_MODE` | `saas` / `byoc` / `local` — SaaS enforces TLS on all DBs | `local` | No |
| `VERSION` | Service version (injected via ldflags or env) | `dev` | No |
| `MONGO_URI` | MongoDB connection string | `mongodb://localhost:27017/flowker` | No |
| `AUDIT_DB_HOST` | PostgreSQL audit database host | `localhost` | No |
| `AUDIT_DB_PORT` | PostgreSQL audit database port | `5432` | No |
| `AUDIT_DB_USER` | PostgreSQL audit database user | `flowker_audit` | No |
| `AUDIT_DB_PASSWORD` | PostgreSQL audit database password | `flowker_audit` | No |
| `AUDIT_DB_NAME` | PostgreSQL audit database name | `flowker_audit` | No |
| `AUDIT_DB_SSL_MODE` | PostgreSQL SSL mode (`disable`, `require`, etc.) | `disable` | No |

---

## Kubernetes Probe Configuration

Copy-paste into your Deployment spec:

```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 2

livenessProbe:
  httpGet:
    path: /health
    port: http
  initialDelaySeconds: 30
  periodSeconds: 10
  failureThreshold: 3
```

**Note:** Drain grace period is 12 seconds (default), calibrated for `periodSeconds=5 * failureThreshold=2 + buffer`.

---

## Response Contract

### `/readyz` Response (200 OK)

```json
{
  "status": "healthy",
  "checks": {
    "mongodb": {
      "status": "up",
      "latency_ms": 3,
      "tls": true
    },
    "postgresql": {
      "status": "up",
      "latency_ms": 2,
      "tls": false
    }
  },
  "version": "1.0.0",
  "deployment_mode": "local"
}
```

### `/readyz` Response (503 Service Unavailable)

```json
{
  "status": "unhealthy",
  "checks": {
    "mongodb": {
      "status": "down",
      "error": "context deadline exceeded"
    },
    "postgresql": {
      "status": "up",
      "latency_ms": 2,
      "tls": false
    }
  },
  "version": "1.0.0",
  "deployment_mode": "local"
}
```

---

## Status Vocabulary

| Status | Meaning | Aggregation Impact |
|--------|---------|-------------------|
| `up` | Dependency reachable, check passed | Healthy |
| `down` | Dependency unreachable or check failed | **Unhealthy → 503** |
| `degraded` | Circuit breaker half-open OR partial failure | **Unhealthy → 503** |
| `skipped` | Optional dependency explicitly disabled | Healthy (ignored) |
| `n/a` | Not applicable in current mode | Healthy (ignored) |

**Aggregation rule:** Top-level `status` is `"healthy"` **if and only if** every check is `up`, `skipped`, or `n/a`. ANY `down` or `degraded` → HTTP 503.

---

## Metrics Reference

All metrics are emitted on every `/readyz` request and at startup self-probe.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `readyz_check_duration_ms` | Histogram | `dep`, `status` | Duration of dependency checks in milliseconds |
| `readyz_check_status` | Counter | `dep`, `status` | Count of check outcomes per dependency |
| `selfprobe_result` | Gauge | `dep` | Last self-probe result (1=up, 0=down) |

**Histogram buckets (ms):** `[1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000]`

### Example Prometheus Queries

```promql
# Dependency check latency P99 (aggregated across instances)
histogram_quantile(0.99, sum by (le) (rate(readyz_check_duration_ms_bucket[5m])))

# Dependency check latency P99 by dependency
histogram_quantile(0.99, sum by (dep, le) (rate(readyz_check_duration_ms_bucket[5m])))

# Failure rate by dependency
rate(readyz_check_status{status="down"}[5m])

# Current self-probe state per dependency
selfprobe_result{dep="mongodb"}
selfprobe_result{dep="postgresql"}

# Detect any failing dependency (min=0 means at least one dep is down)
min(selfprobe_result)
```

---

## Operational Runbook

### `/readyz` returning 503

1. Check which dependency is `down` or `degraded` in the response body
2. Inspect the `error` field for the failing dependency
3. Verify network connectivity to the database
4. Check database credentials and connection strings
5. Review database logs for connection limits or authentication failures

### `/health` returning 503

1. Self-probe failed at startup — pod is alive but not ready
2. Check startup logs for `startup_self_probe_failed` or `self_probe_check` entries
3. Kubernetes will restart the pod automatically (livenessProbe failure)
4. Investigate why dependencies were unreachable at boot time

### Service refusing to start in SaaS mode

1. Error message: `DEPLOYMENT_MODE=saas: TLS required for <dep> but not configured`
2. `ValidateSaaSTLS` detected a non-TLS connection string
3. Update the connection string to use TLS:
   - MongoDB: Use `mongodb+srv://` or add `tls=true` parameter
   - PostgreSQL: Change `sslmode` from `disable` to `require` or higher
4. Restart the service

### In-flight requests killed during deploy

1. Drain grace period too short
2. Increase grace period past `periodSeconds * failureThreshold` (currently 12s)
3. Verify `/readyz` returns 503 immediately after SIGTERM
4. Check `GracefulShutdownHandler` logs for proper drain sequence

---

## Scope Fence (What's NOT in /readyz)

| Excluded | Reason | Where it belongs |
|----------|--------|------------------|
| Synthetic business-logic probes | `/readyz` is infrastructure, not application health | Separate `/biz-check` endpoint if needed |
| Certificate validity/expiry | `/readyz` reports TLS posture, not cert health | External cert-monitoring tool |
| Performance SLIs (p99 latency) | `/readyz` is binary healthy/unhealthy | Telemetry dashboards |
| Per-tenant business rules | Global `/readyz` is tenant-agnostic | Future `/readyz/tenant/:id` endpoint |

---

## Common Errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| 401 on `/readyz` | Endpoint mounted behind auth middleware | Verify `/readyz` is mounted BEFORE auth in `routes.go` |
| Metrics not in `/metrics` | Metrics not registered or registry not exposed | Check `prometheus.MustRegister` calls in `readyz/metrics.go` |
| `selfprobe_result{dep="X"}` stays 0 | `RunSelfProbe` not invoked or dep X unreachable at boot | Check startup logs for `self_probe_check` entries; query `selfprobe_result{dep="mongodb"}` or `selfprobe_result{dep="postgresql"}` individually |
| `/readyz` returns stale results | Cache layer in front (FORBIDDEN) | Remove any caching middleware on `/readyz` |
| Service starts with non-TLS in SaaS mode | `ValidateSaaSTLS` not called or bypassed | Verify `ValidateSaaSTLS(cfg)` is called in bootstrap before connections |

---

## TLS Detection Logic

TLS detection uses `url.Parse()` (not substring matching) for accuracy:

| Dependency | TLS Detection Method |
|------------|---------------------|
| MongoDB | Scheme `mongodb+srv://` OR query param `tls=true` OR `ssl=true` |
| PostgreSQL | Query param `sslmode` ∈ {`require`, `verify-ca`, `verify-full`} |

**Note:** PostgreSQL `sslmode=allow` and `sslmode=prefer` are NOT considered TLS-enabled because they can fall back to cleartext connections.

Example connection strings:

```bash
# MongoDB with TLS
MONGO_URI="mongodb+srv://user:pass@cluster.mongodb.net/flowker"
MONGO_URI="mongodb://user:pass@host:27017/flowker?tls=true"

# PostgreSQL with TLS
AUDIT_DB_SSL_MODE="require"
AUDIT_DB_SSL_MODE="verify-full"
```

---

## Graceful Shutdown Sequence

1. **SIGTERM received** → `drainingState.Store(true)`
2. **`/readyz` returns 503** immediately (even if deps are healthy)
3. **K8s stops routing** new traffic to the pod
4. **Grace period (12s)** allows in-flight requests to complete
5. **`server.Shutdown()`** called
6. **Dependencies closed** (MongoDB, PostgreSQL connections)
7. **Process exits**

---

## Files Reference

| File | Purpose |
|------|---------|
| `internal/adapters/http/in/readyz/handler.go` | `/readyz` handler, response types, status vocabulary, and `IsDrainingFunc` variable |
| `internal/adapters/http/in/readyz/checker.go` | MongoDB and PostgreSQL health checker implementations |
| `internal/adapters/http/in/readyz/metrics.go` | Prometheus metrics registration |
| `internal/adapters/http/in/health/handler.go` | `/health` handler (gated by `SelfProbeOKFunc`) |
| `internal/bootstrap/selfprobe.go` | Startup self-probe implementation |
| `internal/bootstrap/shutdown.go` | Graceful shutdown handler with draining state |
| `internal/bootstrap/tls_detection.go` | TLS detection via url.Parse |
| `internal/bootstrap/tls_enforcement.go` | `ValidateSaaSTLS()` for SaaS mode |
