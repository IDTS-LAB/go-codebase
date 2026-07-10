.PHONY: run build test lint fmt migrate-up migrate-down sqlc swagger docker-up docker-down clean rename

APP_NAME := go-codebase
BUILD_DIR := bin

run:
	go run ./cmd/api

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api

test:
	go test -v -count=1 ./...

test-coverage:
	go test -v -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	goimports -w .

migrate-up:
	goose -dir migrations up

migrate-down:
	goose -dir migrations down

sqlc:
	sqlc generate

swagger:
	swag init -g cmd/api/swagger.go -o docs --parseDependency --parseInternal

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

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

mod-tidy:
	go mod tidy

rename:
	./scripts/rename-module.sh $(MODULE)

.DEFAULT_GOAL := run
