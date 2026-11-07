# Backend tasks. Only Go is required (codegen tools run via `go run` at pinned
# versions; sqlc's Postgres parser needs cgo, i.e. a C compiler such as gcc).

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

SQLC_VERSION          ?= v1.30.0
OAPI_CODEGEN_VERSION  ?= v2.6.0
GOLANGCI_LINT_VERSION ?= v2.15.0

COURSE       ?= cu-best-math
GRAPH_YAML   ?= ../content/$(COURSE)/graph.yaml
DATABASE_URL ?= postgres://tutor:tutor@localhost:5433/tutor?sslmode=disable
BIN          ?= bin/server

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: generate
generate: ## Regenerate sqlc store code and the oapi-codegen strict server
	CGO_ENABLED=1 go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION) \
		-config internal/httpapi/oapi-codegen.yaml ../api/openapi.yaml
	gofmt -w internal/store internal/httpapi

.PHONY: build
build: ## Build the server binary (static, CGO disabled) into bin/server
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/server

.PHONY: run
run: ## Run the server locally (needs DATABASE_URL; dev login + insecure cookies)
	AUTH_DEV_LOGIN=$${AUTH_DEV_LOGIN:-true} SESSION_COOKIE_SECURE=$${SESSION_COOKIE_SECURE:-false} \
		DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server

.PHONY: seed
seed: ## Import ../content/$(COURSE)/graph.yaml into DATABASE_URL (idempotent)
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed -file $(GRAPH_YAML)

.PHONY: rebuild-mastery
rebuild-mastery: ## Recompute every Learner's Mastery from the Answer history
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/server rebuild-mastery

.PHONY: test
test: ## Run all tests with -race (integration tests use TEST_DATABASE_URL, else testcontainers)
	go test -race -count=1 ./...

.PHONY: test-unit
test-unit: ## Run only the pure domain tests
	go test -race -count=1 ./internal/domain/...

.PHONY: lint
lint: ## gofmt check, go vet, golangci-lint
	@test -z "$$(gofmt -l . | tee /dev/stderr)"
	go vet ./...
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

.PHONY: fmt
fmt: ## Format code
	gofmt -w .

.PHONY: docker
docker: ## Build the backend container image
	docker build -t tutor-platform/backend:local .
