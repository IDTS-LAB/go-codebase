# Observability & Monitoring Design

## Overview

Add full-stack monitoring with Prometheus metrics, Grafana dashboards, and Alertmanager notifications. The Go application exposes Prometheus metrics, infrastructure exporters collect database/cache metrics, and Grafana provides visual dashboards for admin analysis.

## Architecture

```
Go App (8080)         ── /metrics ──┐
  └─ monitoring module              │
PostgreSQL            ── exporter ──┤
Redis                 ── exporter ──┤── Prometheus ──▶ Alertmanager ──▶ Email / Discord / UI
NATS (8222)          ── /metrics ───┤       │
Jaeger (4318)        ── OTLP ───────┘       └──▶ Grafana (provisioned dashboards)
```

## Components

### 1. Go Application Metrics (`internal/monitoring/`)

Standalone domain module with:

- `domain/` — `MetricsRecorder` interface (IncrementCounter, ObserveHistogram, SetGauge)
- `infrastructure/prometheus/` — Prometheus implementation with:
  - HTTP request counter (method, path, status)
  - HTTP request duration histogram (50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s)
  - Active requests gauge
  - Business event counters (user_registered, login_success, login_failed, email_sent, todo_created)
- `interfaces/http/` — Handler registering `/metrics` endpoint (promhttp.Handler())
- `module.go` — Fx module providing the MetricsRecorder and registering the endpoint

### 2. Middleware (`internal/shared/middleware/metrics.go`)

HTTP middleware implementing the RED metrics pattern — Rate, Errors, Duration. Records request count, status code, and duration for every request using the MetricsRecorder interface.

### 3. Infrastructure Exporters

Added to docker-compose.yml:
- **postgres_exporter** (prometheuscommunity/postgres-exporter) — connections, transactions, cache hit ratio, deadlocks
- **redis_exporter** (oliver006/redis-exporter) — hit rate, memory, connected clients, command rate

### 4. Prometheus Configuration (`deployments/prometheus/`)

Scrape targets:
- `api:8080` — Go app /metrics
- `postgres_exporter:9187`
- `redis_exporter:9121`
- `nats:8222` — NATS monitoring endpoint

Alerting rules (prometheus/alerts.yml):
- `HighErrorRate` — 5xx > 5% over 5m
- `HighLatency` — p99 latency > 1s over 5m
- `ServiceDown` — target unreachable 1m
- `PostgresPoolExhaustion` — connections > 80% of max
- `RedisDown` — exporter unreachable
- `HighMemoryUsage` — Go RSS > 500MB
- `HighCPUUsage` — Go CPU > 80% over 5m

### 5. Alertmanager (`deployments/alertmanager/`)

Three receivers:
- **Prometheus UI** — default route
- **Email** — SMTP via configured credentials
- **Discord** — webhook receiver, Discord webhook URL from env var

### 6. Grafana Provisioning (`deployments/grafana/`)

Auto-provisioned dashboards:
1. **Go App RED** — Request rate rps, error rate %, p50/p95/p99 latency, active requests, goroutines, memory, GC, business events
2. **PostgreSQL** — Connections, tps, cache hit ratio, deadlocks, query duration
3. **Redis** — Hit rate, memory, connected clients, commands/s
4. **Overview** — All services health, top errors, latency heatmap

### 7. Docker Compose Updates

- Add postgres_exporter, redis_exporter services
- Mount prometheus config directory (alerts.yml)
- Mount grafana provisioning directory
- Add alertmanager service

## Config

New section in `configs/config.yaml`:

```yaml
monitoring:
  metrics_path: /metrics
  discord_webhook_url: ""
```
