FROM golang:1.26-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
