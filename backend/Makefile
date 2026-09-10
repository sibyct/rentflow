.PHONY: run build test test-integration lint fmt migrate-up migrate-down migrate-create tidy

APP_NAME := api
MIGRATIONS_DIR := migrations
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/propertymanagement?sslmode=disable

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

run:
	go run ./cmd/api

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/api

test:
	go test ./... -race -count=1

test-integration:
	go test ./... -race -count=1 -tags=integration -timeout=5m

lint:
	golangci-lint run ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

migrate-up:
	migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS_DIR) up

migrate-down:
	migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS_DIR) down 1

# Usage: make migrate-create name=add_leases_table
migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
