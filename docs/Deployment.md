# Deployment

## Docker

### Local Development

```bash
# Build and start all services (dev stack with observability)
make docker-up

# Stop all services
make docker-down

# Build the image only
make docker-build
```

### Production Compose

```bash
# Copy and edit environment variables
cp .env.example .env
# Edit .env with production values (especially DB_PASSWORD, JWT_SECRET, SMTP_*)

# Start production stack
docker compose -f docker-compose.prod.yml up -d

# Run migrations
docker compose -f docker-compose.prod.yml --profile migrate run --rm migrate
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

Set all required environment variables (see `.env.example`). Critical secrets:

- `JWT_SECRET` — must be changed from default in production
- `DB_PASSWORD` — strong database password
- `SMTP_PASSWORD` / `SENDGRID_API_KEY` — email provider credentials

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

## CI/CD

### GitHub Actions

Two workflows are defined in `.github/workflows/`:

#### CI (`ci.yml`)

Runs on every push/PR to `main` and `development`:
- Lint with `golangci-lint`
- Run tests with race detection against PostgreSQL and Redis services
- Build the Go binary
- Build the Docker image

#### CD (`cd.yml`)

Runs on pushes to `main` and version tags:
- Builds and pushes Docker image to GitHub Container Registry (`ghcr.io/idts-lab/go-codebase`)
- Tags images with branch, tag, and short SHA
- Staging deployment job (triggered on `main` push)
- Production deployment job (triggered on `v*.*.*` tags)

### Required Secrets

No repository secrets are required for CI or image push (uses `GITHUB_TOKEN`). Add your own secrets if deployment jobs need SSH keys, kubeconfigs, or cloud credentials.

## Kubernetes

Base manifests are in `k8s/base/` with staging/production overlays in `k8s/overlays/`.

### Deploy with Kustomize

```bash
# Staging
kubectl apply -k k8s/overlays/staging

# Production
kubectl apply -k k8s/overlays/production
```

### Before deploying

1. Update `k8s/base/secret.yaml` with real credentials, or use an external secret manager.
2. Update ingress hosts in overlays to your actual domains.
3. Ensure cert-manager is installed for TLS, or remove the `cert-manager.io/cluster-issuer` annotation.

## Health Checks

- **Health**: `GET /health` - Returns 200 if service is running
- **Readiness**: `GET /ready` - Returns 200 if service is ready to accept traffic

## Monitoring

- **Swagger UI**: http://localhost:8080/swagger/index.html - Interactive API docs
- **Jaeger**: http://localhost:16686 - Distributed tracing
- **Prometheus**: http://localhost:9090 - Metrics
- **Grafana**: http://localhost:3000 - Dashboards
