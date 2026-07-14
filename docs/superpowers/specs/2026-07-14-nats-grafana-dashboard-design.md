# NATS Grafana Dashboard Design

## Overview

Add a NATS monitoring dashboard to Grafana and instrument the Go application's NATSMessenger to expose per-subject message metrics. The dashboard shows NATS server health, per-subject message throughput, and recent message payloads — all via existing Prometheus infrastructure plus a lightweight HTTP debug endpoint.

## Architecture

```
NATS Server (:8222)    ── /metrics ──┐
  (server-level metrics)              │
                                      ├── Prometheus ──▶ Grafana
Go API App (:8080)                   │
  ├─ NATSMessenger    ── /metrics ───┘     + Infinity datasource
  │  (per-subject metrics)                  └──▶ /debug/nats (JSON)
  └─ /debug/nats endpoint
       (in-memory ring buffer, last 100 messages)
```

Grafana data sources used:
- **Prometheus** — NATS server metrics (`nats_*`) and app-instrumented per-subject metrics
- **Infinity** — fetches recent message payloads from `GET /debug/nats`

## Components

### 1. NATSMessenger Instrumentation (`internal/infrastructure/messaging/nats.go`)

Add four Prometheus metrics, registered via `promauto` (same pattern as `internal/monitoring/`):

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `nats_published_total` | CounterVec | `subject` | Per-subject publish count |
| `nats_received_total` | CounterVec | `subject` | Per-subject receive count |
| `nats_publish_bytes_total` | CounterVec | `subject` | Total bytes published per subject |
| `nats_received_bytes_total` | CounterVec | `subject` | Total bytes received per subject |

Instrument `Publish()` and `Subscribe()` to increment counters on each call.

### 2. NATS Debug Endpoint (in a new handler file)

New file: `internal/infrastructure/messaging/debug.go`

- In-memory ring buffer (capacity 100) of recent published messages
- Each entry: subject, timestamp, truncated payload (max 1KB, returned as string; binary payloads base64-encoded), direction (publish/subscribe)
- Exposed as `GET /debug/nats` returning JSON array sorted newest-first
- Only active when NATS is configured **and** `nats_debug_endpoint` config is true (disabled by default)

### 3. Route Registration (`cmd/api/main.go`)

Register `GET /debug/nats` on the API router, gated behind `NATS_DEBUG_ENDPOINT=true` (or `nats.debug_endpoint` in config). Disabled by default to avoid exposure in production.

### 4. Grafana Dashboard (`deployments/grafana/dashboards/nats.json`)

Provisioned dashboard with three rows:

**Row 1 — NATS Server Health**
| Panel | Type | Query |
|-------|------|-------|
| Active Connections | Stat | `nats_connections` |
| Subscriptions | Stat | `nats_subscriptions` |
| Uptime | Stat | `nats_uptime_seconds` |
| Messages In/Out | Time series | `rate(nats_messages_sent_total[5m])`, `rate(nats_messages_received_total[5m])` |
| Bytes In/Out | Time series | `rate(nats_in_bytes_total[5m])`, `rate(nats_out_bytes_total[5m])` |

**Row 2 — Per-Subject Activity**
| Panel | Type | Query |
|-------|------|-------|
| Publish Rate by Subject | Bar gauge | `rate(nats_published_total[5m])` |
| Receive Rate by Subject | Bar gauge | `rate(nats_received_total[5m])` |
| Data Volume by Subject | Stacked time series | `rate(nats_publish_bytes_total[5m])` |

**Row 3 — Recent Messages**
| Panel | Type | Data Source |
|-------|------|-------------|
| Recent Message Payloads | Table | Infinity — `GET /debug/nats`, columns: time, subject, payload |

### 5. Infinity Datasource Provisioning

Add Infinity datasource to `deployments/grafana/datasources/datasources.yml` pointing to the Go API's `/debug/nats` endpoint.

**Note:** The Infinity datasource is a community plugin (`yesoreyeram-infinity-datasource`). It must be installed in Grafana (add `GF_INSTALL_PLUGINS=yesoreyeram-infinity-datasource` to Grafana's environment in docker-compose).

## Config

Add to `configs/config.yaml`:

```yaml
nats:
  url: nats://localhost:4222
  debug_endpoint: false   # enables /debug/nats HTTP endpoint
```

The debug endpoint is available only when both `NATS_URL` is set and `debug_endpoint` is true.

## Testing

- Unit tests for the ring buffer logic
- Unit tests for Prometheus counter incrementation in `Publish()` and `Subscribe()`
- Integration test: start NATS, publish/subscribe messages, verify `/debug/nats` returns them
- Manual: start docker-compose, verify dashboard panels populate

## Files Changed

| File | Change |
|------|--------|
| `internal/infrastructure/messaging/nats.go` | Add Prometheus counters, instrument Publish/Subscribe |
| `internal/infrastructure/messaging/debug.go` | New — ring buffer + HTTP handler |
| `internal/shared/config/config.go` | Add `DebugEndpoint bool` to `NATSConfig` |
| `configs/config.yaml` | Add `nats.debug_endpoint: false` |
| `.env` | Add `NATS_DEBUG_ENDPOINT=false` |
| `cmd/api/main.go` | Register `/debug/nats` route |
| `docker-compose.yml` | Add `NATS_DEBUG_ENDPOINT` env to API; add `GF_INSTALL_PLUGINS` to Grafana |
| `deployments/grafana/dashboards/nats.json` | New — NATS dashboard |
| `deployments/grafana/datasources/datasources.yml` | Add Infinity datasource |
| `deployments/grafana/dashboards/dashboard.yml` | Add nats.json to provisioning |
