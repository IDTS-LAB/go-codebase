.PHONY: run build test test-unit test-integration lint fmt migrate-up migrate-down sqlc swagger docker-up docker-down docker-dev docker-dev-down clean rename install-tools precommit install-hooks

APP_NAME := go-codebase
BUILD_DIR := bin

GOINSTALL = go install

GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null)
GOIMPORTS := $(shell command -v goimports 2>/dev/null)
GOOSE := $(shell command -v goose 2>/dev/null)
SQLC := $(shell command -v sqlc 2>/dev/null)
SWAG := $(shell command -v swag 2>/dev/null)

install-tools:
ifndef GOLANGCI_LINT
	$(GOINSTALL) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
endif
ifndef GOIMPORTS
	$(GOINSTALL) golang.org/x/tools/cmd/goimports@latest
endif
ifndef GOOSE
	$(GOINSTALL) github.com/pressly/goose/v3/cmd/goose@latest
endif
ifndef SQLC
	$(GOINSTALL) github.com/sqlc-dev/sqlc/cmd/sqlc@latest
endif
ifndef SWAG
	$(GOINSTALL) github.com/swaggo/swag/cmd/swag@latest
endif

run:
	go run ./cmd/api

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api

test:
	go test -v -count=1 ./...

test-unit:
	go test -v -count=1 $(shell go list ./... | grep -v /tests/)

test-integration:
	go test -v -count=1 $(shell go list ./... | grep /tests/)

test-coverage:
	go test -v -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: install-tools
	golangci-lint run ./...

fmt: install-tools
	go fmt ./...
	goimports -w .

migrate-up: install-tools
	goose -dir migrations up

migrate-down: install-tools
	goose -dir migrations down

sqlc: install-tools
	sqlc generate

swagger: install-tools
	swag init -g cmd/api/swagger.go -o docs --parseDependency --parseInternal

precommit: install-tools mod-tidy
	@echo "=== Checking formatting ==="
	@if [ -n "$$(go fmt ./...)" ]; then \
		echo "ERROR: Files not formatted. Run 'make fmt' and commit again."; \
		exit 1; \
	fi
	@echo "=== Linting ==="
	golangci-lint run ./...
	@echo "=== Building ==="
	go build ./...
	@echo "=== Testing ==="
	go test -count=1 ./...
	@echo "=== All checks passed ==="

install-hooks:
	@echo "Installing pre-commit hook..."
	@printf '#!/bin/sh\nmake -C "$(CURDIR)" precommit\n' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed."

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-prod-up:
	docker compose -f docker-compose.prod.yml up -d

docker-prod-down:
	docker compose -f docker-compose.prod.yml down

docker-prod-migrate:
	docker compose -f docker-compose.prod.yml --profile migrate run --rm migrate

docker-dev:
	docker compose -f docker-compose.dev.yml up -d

docker-dev-down:
	docker compose -f docker-compose.dev.yml down

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

mod-tidy:
	go mod tidy

rename:
	./scripts/rename-module.sh $(MODULE)

.DEFAULT_GOAL := run
