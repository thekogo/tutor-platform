# Root task runner. Delegates to backend/Makefile and frontend/package.json scripts.
# Run `make` (or `make help`) to list targets.

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

COMPOSE      ?= docker compose
COMPOSE_E2E  := $(COMPOSE) -f docker-compose.yml -f docker-compose.e2e.yml
PNPM         ?= pnpm
PYTHON       ?= python3

COURSE       ?= cu-best-math
GRAPH_YAML   := content/$(COURSE)/graph.yaml
GRAPH_MD     := content/$(COURSE)/graph.md

# Host-side URL of the compose Postgres (used by `seed`, `dev-db`, tests).
DATABASE_URL ?= postgres://tutor:tutor@localhost:5433/tutor?sslmode=disable
export DATABASE_URL

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z0-9_-]+:.*##/ { printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Stack (docker compose)

.PHONY: up
up: ## Build and start db + backend + frontend (waits until healthy)
	$(COMPOSE) up -d --build --wait

.PHONY: down
down: ## Stop the stack (keeps the database volume; use `docker compose down -v` to wipe)
	$(COMPOSE) down

.PHONY: logs
logs: ## Follow logs of all services
	$(COMPOSE) logs -f --tail=200

.PHONY: dev-db
dev-db: ## Start only Postgres on localhost:5433 (for running backend/frontend on the host)
	$(COMPOSE) up -d --wait db
	@echo "DATABASE_URL=$(DATABASE_URL)"

.PHONY: seed
seed: ## Import content/<course>/graph.yaml (COURSE=cu-best-math) into the database at DATABASE_URL
	$(MAKE) -C backend seed

##@ Code

.PHONY: generate
generate: ## Regenerate code from api/openapi.yaml and SQL (backend + frontend)
	$(MAKE) -C backend generate
	cd frontend && $(PNPM) run gen:api

.PHONY: lint
lint: ## Lint backend, frontend and e2e
	$(MAKE) -C backend lint
	cd frontend && $(PNPM) run lint && $(PNPM) run typecheck
	cd e2e && $(PNPM) run typecheck

.PHONY: test
test: ## Unit + integration tests (backend uses TEST_DATABASE_URL if set)
	$(MAKE) -C backend test
	cd frontend && $(PNPM) run test

.PHONY: build
build: ## Build backend binary and frontend production bundle
	$(MAKE) -C backend build
	cd frontend && $(PNPM) run build

.PHONY: graph-check
graph-check: ## Validate the course graph and re-render its Mermaid markdown
	$(PYTHON) scripts/graph_check.py $(GRAPH_YAML) $(GRAPH_MD)

.PHONY: install
install: ## Install frontend and e2e dependencies (pnpm, frozen lockfile)
	cd frontend && $(PNPM) install --frozen-lockfile
	cd e2e && $(PNPM) install --frozen-lockfile

##@ End-to-end

.PHONY: e2e-up
e2e-up: ## Start the e2e stack (dev login enabled) and seed it
	$(COMPOSE_E2E) up -d --build --wait
	$(MAKE) seed

.PHONY: e2e-down
e2e-down: ## Stop the e2e stack and delete its volume
	$(COMPOSE_E2E) down -v

.PHONY: e2e
e2e: e2e-up ## Run Playwright against the e2e stack (stack is left running; `make e2e-down` to stop)
	cd e2e && $(PNPM) install --frozen-lockfile && $(PNPM) exec playwright install --with-deps chromium && $(PNPM) test

##@ CI

.PHONY: ci
ci: ## Run the CI checks locally (no e2e/docker; run `make e2e` separately)
	@echo "==> gofmt"
	@test -z "$$(gofmt -l backend | tee /dev/stderr)"
	@echo "==> go vet"
	cd backend && go vet ./...
	$(MAKE) -C backend lint
	$(MAKE) -C backend test
	@echo "==> codegen drift"
	$(MAKE) generate
	git diff --exit-code -- backend frontend
	cd frontend && $(PNPM) run typecheck && $(PNPM) run lint && $(PNPM) run test && $(PNPM) run build
	cd e2e && $(PNPM) run typecheck
	$(MAKE) graph-check
	git diff --exit-code -- content
