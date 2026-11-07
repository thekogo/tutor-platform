# Tutor Platform

A tutoring platform that models a course as a **knowledge graph**, tracks where each Learner is weak, and points them to the Concept they should study next. When a Learner struggles, the platform follows the Prerequisite graph back to the **Root Cause** and recommends studying that first, always showing why.

The first Course is **CU-BEST Math** (entrance exam for graduate study at Chulalongkorn University). The UI is in Thai.

Content (Concepts, Lessons, Questions) is written by the Owner's AI agent (Claude) through an **MCP server**; the platform itself never calls an LLM.

- Domain language: [CONTEXT.md](CONTEXT.md) (read this first; the code uses these terms)
- Architecture, data model, quality bar: [docs/architecture.md](docs/architecture.md)
- Decisions: [docs/adr/](docs/adr/)
- REST contract (source of truth): [api/openapi.yaml](api/openapi.yaml)
- Course graph: [content/cu-best-math/graph.yaml](content/cu-best-math/graph.yaml) ([rendered](content/cu-best-math/graph.md))

## Stack

| Part | Tech |
|---|---|
| Backend | Go 1.26, one binary: REST `/api`, MCP `/mcp`, probes `/healthz` `/readyz`; pgx + sqlc, goose migrations (run on start) |
| Frontend | Next.js (App Router, TypeScript, Tailwind), proxies `/api` to the backend |
| Database | PostgreSQL 17 |
| E2E | Playwright (`e2e/`) |

```
api/       REST contract          backend/   Go API + MCP server
frontend/  Next.js web app        content/   seed knowledge graphs
e2e/       Playwright journeys    scripts/   dev utilities (graph_check.py)
docs/      architecture + ADRs    .github/   CI, CodeQL, Dependabot
```

## Prerequisites

- Docker with Compose v2 (for the quick start)
- For local development without Docker: Go 1.26, Node 22, pnpm 10 (`corepack enable`), Python 3 + PyYAML (graph check)
- GNU make

## Quick start (docker compose)

```sh
cp .env.example .env        # safe dev defaults; dev login enabled
make up                     # builds and starts db, backend, frontend; waits until healthy
make seed                   # imports content/cu-best-math/graph.yaml
```

Open http://localhost:3000. With `AUTH_DEV_LOGIN=true` and `NEXT_PUBLIC_DEV_LOGIN=true` the login page offers a dev login (no Google needed). **The first person to log in becomes the Owner.**

| Service | URL |
|---|---|
| Web app | http://localhost:3000 |
| Backend REST | http://localhost:8080/api |
| MCP | http://localhost:8080/mcp |
| Probes | http://localhost:8080/healthz, http://localhost:8080/readyz |
| Postgres | `postgres://tutor:tutor@localhost:5433/tutor` (host port 5433 to avoid clashing with a local 5432) |

`make logs` follows logs, `make down` stops (data kept), `docker compose down -v` wipes the database.

## Local development without Docker for the apps

Run only Postgres in Docker and the apps on your host (hot reload):

```sh
cp .env.example .env
make dev-db                                   # Postgres on localhost:5433
set -a; source .env; set +a                   # export env into your shell

# terminal 1: backend (migrations run on start)
make -C backend seed                          # once, after the first start
cd backend && go run ./cmd/server

# terminal 2: frontend
cd frontend && pnpm install && pnpm dev       # BACKEND_URL=http://localhost:8080
```

Frontend-only work without a backend: `NEXT_PUBLIC_API_MOCKING=true pnpm dev` (mocked API).

## Make targets

Run `make` to list everything. The main ones:

| Target | What it does |
|---|---|
| `make up` / `down` / `logs` | Docker compose stack |
| `make dev-db` | Only Postgres on :5433 |
| `make seed` | Import the course graph (`$(MAKE) -C backend seed`) into `DATABASE_URL` |
| `make generate` | Regenerate code from `api/openapi.yaml` (oapi-codegen, openapi-typescript) and SQL (sqlc) |
| `make lint` | golangci-lint, ESLint, tsc |
| `make test` | Go tests (integration tests use `TEST_DATABASE_URL`) + Vitest |
| `make build` | Backend binary + `next build` |
| `make graph-check` | Validate the course graph DAG and re-render `graph.md` |
| `make e2e` | Start the e2e stack (dev login on), seed it, run Playwright |
| `make e2e-down` | Stop the e2e stack and delete its data |
| `make ci` | What CI runs, locally (except Docker builds and e2e) |

## Seeding

`make seed` imports `content/cu-best-math/graph.yaml` (Course, Topics, Concepts, Prerequisites) into the database at `DATABASE_URL` (default `postgres://tutor:tutor@localhost:5433/tutor`). It is idempotent: slugs in `graph.yaml` are stable external IDs.

Questions and Lessons are not in the seed: the Owner's Claude agent creates them through MCP, and every Question goes through an independent verification pass before it becomes Active (ADR 0003).

After editing a graph, run `make graph-check` (validates the DAG, re-renders `graph.md`) and commit both files.

## Connecting Claude to the MCP server

The backend exposes MCP over Streamable HTTP at `/mcp`, authenticated with `Authorization: Bearer $MCP_TOKEN`. The token acts as **Content Admin** with Owner visibility, so treat it like a password: generate it with `openssl rand -hex 32` and set `MCP_TOKEN` in `.env`.

**Claude Code:**

```sh
claude mcp add --transport http tutor-platform http://localhost:8080/mcp \
  --header "Authorization: Bearer $MCP_TOKEN"
```

Or add it to a project `.mcp.json` (don't commit the token, use an env var):

```json
{
  "mcpServers": {
    "tutor-platform": {
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "headers": { "Authorization": "Bearer ${MCP_TOKEN}" }
    }
  }
}
```

**Claude Desktop / claude.ai:** remote connectors must be reachable over HTTPS, so deploy behind TLS (or tunnel) and add the URL as a custom connector with the bearer token.

Then ask Claude, for example: "Use get_course_graph for cu-best-math, then write 5 Draft Questions for integer-operations with create_questions." See the tool list in [docs/architecture.md](docs/architecture.md#backend).

## Testing

The testing pyramid (details in [CONTRIBUTING.md](CONTRIBUTING.md)):

```sh
make test                   # Go unit + integration, Vitest
make lint                   # golangci-lint (backend/.golangci.yml), ESLint, tsc
make e2e                    # Playwright journeys against docker compose (dev login on)
make e2e-down               # clean up
```

Integration tests need Postgres: `make dev-db` and export `TEST_DATABASE_URL` (see `.env.example`). The e2e stack uses its own compose project (`tutor-platform-e2e`) and volume, but the same host ports, so stop the dev stack first (`make down`).

E2E details: `e2e/` is a standalone pnpm project. `cd e2e && pnpm install && pnpm exec playwright install chromium`, then `pnpm test`, `pnpm test:ui`, or `pnpm report`. Configure with `E2E_BASE_URL`, `E2E_MCP_URL`, `E2E_MCP_TOKEN`.

## CI

GitHub Actions ([.github/workflows/ci.yml](.github/workflows/ci.yml)) runs path-filtered jobs: backend (gofmt, vet, golangci-lint, `go test -race` against a Postgres service, govulncheck, codegen drift), frontend (generated types drift, tsc, ESLint, Vitest, `next build`, `pnpm audit`), content graph check, Docker image builds, and Playwright e2e. `CI OK` is the single required status check. CodeQL scans Go and TypeScript weekly and on PRs.

## Contributing and security

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).
