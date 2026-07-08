# Deployment

## Docker

### Build and Run

```bash
# Build and start all services
make docker-up

# Stop all services
make docker-down
```

### Services

| Service | Port | Description |
|---------|------|-------------|
| API | 8080 | Go API server |
| PostgreSQL | 5432 | Database |
| Redis | 6379 | Cache |
| NATS | 4222 | Messaging |
| Jaeger | 16686 | Tracing UI |
| Prometheus | 9090 | Metrics |
| Grafana | 3000 | Dashboards |

## Production

### Environment Variables

Set all required environment variables (see `.env.example`).

### Database

```bash
make migrate-up
make seed
```

### Build

```bash
make build
```

### Run

```bash
./bin/go-codebase
```

## Health Checks

- **Health**: `GET /health` - Returns 200 if service is running
- **Readiness**: `GET /ready` - Returns 200 if service is ready to accept traffic

## Monitoring

- **Swagger UI**: http://localhost:8080/swagger/index.html - Interactive API docs
- **Jaeger**: http://localhost:16686 - Distributed tracing
- **Prometheus**: http://localhost:9090 - Metrics
- **Grafana**: http://localhost:3000 - Dashboards (admin/admin)
