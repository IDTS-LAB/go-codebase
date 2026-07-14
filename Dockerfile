# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Install code generation tools (pinned for reproducibility)
# sqlc and swag outputs are gitignored, so they must be generated in-image
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 && \
    go install github.com/swaggo/swag/cmd/swag@v1.16.6

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate gitignored code: sqlc repositories and swagger docs
RUN sqlc generate && \
    swag init -g cmd/api/swagger.go -o docs --parseDependency --parseInternal

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/api

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# Copy binary and required files
COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./server"]
