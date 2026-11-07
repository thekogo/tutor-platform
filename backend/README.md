# Backend

One Go binary serving the REST API (`/api/*`, contract: [`../api/openapi.yaml`](../api/openapi.yaml)),
the MCP endpoint for the Content Admin agent (`/mcp`), and probes (`/healthz`, `/readyz`).
Design: [`../docs/architecture.md`](../docs/architecture.md). Vocabulary: [`../CONTEXT.md`](../CONTEXT.md).

## Layout

| Path | What |
|---|---|
| `cmd/server` | main: config from env, wiring, graceful shutdown, subcommands |
| `cmd/seed` | `seed -file ../content/<course>/graph.yaml` |
| `internal/domain` | pure engine: graph, Mastery, Recommendations, placement (no I/O) |
| `internal/app` | use cases, transactions, authorization, graph cache, serving rules |
| `internal/store` | goose migrations (embedded, run on start) + sqlc queries on pgx/v5 |
| `internal/httpapi` | oapi-codegen strict server (`api.gen.go`), thin handlers, middleware |
| `internal/mcpserver` | MCP tools (official go-sdk), streamable HTTP, bearer auth |
| `internal/auth` | Google OIDC (invite-only), dev login, sessions, invites |
| `internal/seed` | graph.yaml import (shared by `cmd/seed` and `server seed`) |
| `internal/testutil` | real-Postgres test databases and fixtures |

## Commands

```bash
make generate         # sqlc + oapi-codegen (pinned versions via `go run`; sqlc needs cgo/gcc)
make build            # bin/server (CGO_ENABLED=0)
make run              # local server; dev login on, insecure cookies
make seed             # import ../content/$(COURSE)/graph.yaml into DATABASE_URL (idempotent)
make rebuild-mastery  # recompute all Mastery from Answers (after graph or formula changes)
make test             # go test -race ./...
make lint             # gofmt, go vet, golangci-lint v2
make docker           # container image
```

The binary (`/server` in the image) has subcommands:

```
server [serve]                   HTTP server (migrations run automatically)
server seed -file graph.yaml     import a course graph [-archive-missing]
server rebuild-mastery           full Mastery rebuild (ADR 0002)
server migrate                   apply migrations and exit
server healthcheck               GET http://127.0.0.1:<HTTP_ADDR port>/readyz, exit 0/1
```

## Configuration

| Env | Default | Notes |
|---|---|---|
| `DATABASE_URL` | — | required |
| `HTTP_ADDR` | `:8080` | |
| `APP_BASE_URL` | `http://localhost:3000` | web origin: required `Origin` on unsafe `/api` requests, post-login redirect, invite URLs (`<APP_BASE_URL>/login?invite=<token>`) |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `GOOGLE_REDIRECT_URL` | — | Google login is disabled unless all are set; redirect URL is `…/api/auth/google/callback` |
| `SESSION_COOKIE_SECURE` | `true` | set `false` for plain-http local dev |
| `AUTH_DEV_LOGIN` | `false` | enables `POST /api/auth/dev-login` (dev/e2e only) |
| `MCP_TOKEN` | — | bearer token for `/mcp`; empty disables MCP (503) |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Tests

Integration tests run against real Postgres: `TEST_DATABASE_URL` if set (each test creates and drops
its own database, so the role needs `CREATEDB`), otherwise a `postgres:17-alpine` container via
testcontainers-go; skipped if neither is available.

```bash
TEST_DATABASE_URL=postgres://tutor:tutor@localhost:5433/tutor?sslmode=disable make test
```

## Notes on behaviour

- **Mastery** is a projection of Answers (ADR 0002). Only non-excluded Answers to Active Questions count.
  After each Answer the Question's Concepts are recomputed; Flag/retire/resolve/update recompute every
  affected Learner. A Flag pulls the Question (status `flagged`) for everyone at once.
- **Serving:** Active only; Exam Questions never appear in Quizzes, Diagnostic Quizzes, or Placement
  Tests; unanswered Questions first (Primary-Concept matches first), then repeats (least recently
  answered first, stored `is_repeat`). If a Quiz can't be filled with fresh Questions, a Content Request
  is raised for the Concept (at most one open request per Concept, enforced by a partial unique index).
- **Sessions:** Quiz = 10 Questions (difficulty 1-3) on a Concept or Topic; Diagnostic = 4 Questions
  (difficulty 1-2) on a Concept; Exam = the Exam's Active Questions, deadline enforced server-side
  (`410 deadline_passed`); Placement = adaptive, up to 18 probes, one Question per probed Concept.
- **Placement estimate:** probe selection uses a placement-local Mastery computed only from that
  session's Answers with `MinEvidence = 0.5` (one Answer decides Weak vs Mastered). It is not
  stored; persisted Mastery stays a pure projection of Answers.
- **Learning Paths** started from a Recommendation store that Recommendation's reason path; otherwise
  the shortest Prerequisite chain. A path completes when every step is Developing or better.
- **Archived Concepts** are hidden from the graph; their Dependents inherit their Prerequisites, so
  chains stay intact.
- **Graph cache:** each Course's graph is cached in-process and invalidated on graph writes. With
  several replicas, writes on one replica are not seen by others until restart.
