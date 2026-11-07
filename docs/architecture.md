# Architecture

Vocabulary follows [CONTEXT.md](../CONTEXT.md). Decisions with lasting trade-offs live in [docs/adr](./adr).

## Shape

```
                 ┌──────────────┐  REST /api (cookie session)   ┌─────────────────────────────┐
 Learner ───────▶│  Next.js web │──────────────────────────────▶│                             │
 (browser)       └──────────────┘                               │   Go server (one binary)    │
                                                                │                             │
 Owner's AI  ──── MCP (streamable HTTP /mcp, bearer token) ────▶│  transport: httpapi | mcp   │
 agent (Claude)                                                 │  app: use-case services     │
                                                                │  domain: graph, mastery,    │
                                                                │          recommendation     │
                                                                │  store: pgx + sqlc          │
                                                                └──────────────┬──────────────┘
                                                                               │
                                                                        ┌──────▼──────┐
                                                                        │ PostgreSQL  │
                                                                        └─────────────┘
```

- **One Go binary** serves the REST API (`/api/*`), the MCP endpoint (`/mcp`), and health probes (`/healthz`, `/readyz`). Both transports call the same `app` services; neither contains business logic.
- **The platform never calls an LLM.** All content generation and the independent verification pass are done by the Owner's AI agent through MCP tools (ADR 0003).
- **Contract-first.** `api/openapi.yaml` is the source of truth for REST. The backend implements it (oapi-codegen strict server); the frontend consumes generated types (openapi-typescript). Changing the API means changing the spec first.

## Repository layout

```
api/openapi.yaml            REST contract (source of truth)
backend/
  cmd/server/               main: config, wiring, graceful shutdown
  cmd/seed/                 imports content/<course>/graph.yaml
  internal/domain/          PURE Go, no I/O: graph, mastery, recommendation, placement
  internal/app/             use-case services; transaction boundaries; authorization
  internal/store/           sqlc queries + pgx; migrations (goose, embedded)
  internal/httpapi/         oapi-codegen generated server + handlers, auth middleware
  internal/mcpserver/       MCP tools (modelcontextprotocol/go-sdk) → app services
  internal/auth/            Google OIDC, sessions, invites
frontend/                   Next.js (App Router, TypeScript, Tailwind)
content/<course>/graph.yaml seed knowledge graphs
e2e/                        Playwright end-to-end tests against docker compose
deploy/                     Dockerfiles, compose overrides
scripts/                    dev utilities (graph_check.py)
```

## Backend

**Layers and dependency rule:** `httpapi`/`mcpserver` → `app` → `domain` + `store`. `domain` imports nothing from the project and does no I/O; it is exhaustively unit-tested. `store` returns plain structs; `app` maps them into `domain` types.

**Mastery is a projection** (ADR 0002). Every Answer is stored. After each Answer, `app` recomputes Mastery for the affected Concepts from that Learner's Answers and upserts the `mastery` cache table. A full rebuild command recomputes everything (used after graph edits or calculation changes).

**Domain engine** (`internal/domain`) — see the Go doc comments for exact rules:
- `Graph`: validated DAG of Concepts and Prerequisites; ancestors/descendants queries.
- `ComputeMastery`: recency- and difficulty-weighted accuracy; Primary Concept weight 1.0, secondary 0.5, repeat Answers 0.3; levels Unassessed / Weak / Developing / Mastered via `MasteryConfig`.
- `Recommend`: Root Cause search (skips below Mastered Concepts; Unassessed Prerequisites yield a Diagnose recommendation), Priority = Weak descendants unblocked × Topic Exam Weight (default 1).
- `NextPathStep`: walks a Learning Path, skipping Concepts already Developing or better.
- `NextPlacementConcept`: adaptive Placement Test probe selection.

**Question serving rules:** only Active Questions; Exam Questions never appear in Quizzes; prefer Questions the Learner hasn't answered; when a Concept's pool is exhausted, serve repeats and raise a Content Request.

**Auth:** Google OIDC, invite-only. First login with no Learners → that person becomes Owner. Others need a valid invite token. Sessions are opaque random tokens stored hashed in Postgres, sent as `HttpOnly; Secure; SameSite=Lax` cookie. State-changing requests require a matching `Origin`. `AUTH_DEV_LOGIN=true` enables `POST /api/auth/dev-login` for local dev and e2e only.

**MCP auth:** `Authorization: Bearer <MCP_TOKEN>` (env), acts as Content Admin with Owner visibility.

**MCP tools** (Content Admin):

| Tool | Purpose |
|---|---|
| `get_course_graph` | Course, Topics, Concepts, Prerequisites |
| `import_course_graph` | Upsert a whole graph (same shape as `graph.yaml`); validates DAG |
| `upsert_topic`, `upsert_concept`, `archive_concept` | Graph edits |
| `set_prerequisites` | Replace a Concept's Prerequisites; rejects cycles |
| `upsert_lesson` | Markdown Lesson for a Concept |
| `create_questions` | Batch-create Draft Questions (stem, choices, key, Explanation, tags, Primary Concept, Difficulty, Format, Source) |
| `list_questions_for_verification` | Draft Questions **without** answer key, for the independent verifier |
| `submit_verification` | Verifier's chosen answer; match → Active, mismatch → stays Draft with note |
| `update_question`, `retire_question` | Fix or retire |
| `create_exam` | Fixed paper from Active Questions, time limit |
| `list_flags`, `resolve_flag` | Fix Flagged Questions (`key_changed` drops prior Answers from Mastery) |
| `list_content_requests`, `fulfill_content_request` | Pool exhaustion loop |
| `get_pool_stats`, `get_mastery_overview` | Read-only stats; counts and Concept names, never served Question content |

## Data model (PostgreSQL)

`learners(id, email, name, is_owner, created_at)` · `invites(token_hash, created_by, expires_at, used_by)` · `sessions(token_hash, learner_id, expires_at)`
`courses` · `topics(course_id, slug, name, exam_weight NULL)` · `concepts(topic_id, slug, name, archived_at)` · `prerequisites(concept_id, prerequisite_id)`
`lessons(concept_id, body_md, updated_at)`
`questions(id, course_id, stem_md, explanation_md, difficulty 1-3, format, source, status, verification_note)` · `question_choices(question_id, label, body_md, is_correct)` · `question_concepts(question_id, concept_id, is_primary)`
`exams(course_id, name, time_limit_s)` · `exam_questions(exam_id, question_id, position)`
`study_sessions(id, learner_id, course_id, kind quiz|diagnostic|placement|exam, concept_id?, topic_id?, exam_id?, started_at, finished_at, deadline)` · `session_questions(session_id, question_id, position)`
`answers(id, learner_id, question_id, session_id, choice_id, correct, is_repeat, answered_at, excluded)` — append-only except `excluded`
`mastery(learner_id, concept_id, score, evidence, level, updated_at)` — cache
`learning_paths(learner_id, course_id, root_concept_id, target_concept_id, path concept_id[], started_at, completed_at, abandoned_at)`
`flags(question_id, learner_id, reason, created_at, resolved_at)` · `content_requests(concept_id, reason, created_at, fulfilled_at)`

Slugs from `graph.yaml` are stable external IDs; DB uses UUIDs internally.

## Frontend

Next.js App Router, TypeScript strict, Tailwind. Thai UI copy. Server Components fetch via a typed client generated from `api/openapi.yaml`; mutations through client components. Math: `react-markdown` + `remark-math` + `rehype-katex`. Mastery Map: `@xyflow/react` + `dagre` layout, Concepts colored by Mastery Level, grouped by Topic.

Pages: login · dashboard (top Recommendation with its reason path, other Root Causes, Mastery Map preview) · `/courses/[id]/map` · `/concepts/[id]` (Lesson + start Quiz) · `/sessions/[id]` (Quiz/Diagnostic/Placement one-at-a-time with immediate feedback + Flag; Exam timed, feedback at end) · `/sessions/[id]/result` · `/learning-path` · `/reports/formats` · `/admin` (Owner: invites, Flags, Content Requests, learners).

## Quality & SDLC

- **Branching:** trunk-based, short-lived branches, PRs with CI required; Conventional Commits.
- **CI (GitHub Actions):** backend `gofmt`/`go vet`/`golangci-lint`, unit tests with `-race`, integration tests against Postgres service container; spec drift check (regenerate and `git diff --exit-code`); frontend `tsc`, ESLint, Vitest, `next build`; Playwright e2e on docker compose; graph validation (`scripts/graph_check.py`); Docker image builds; dependency scanning (`govulncheck`, `npm audit`).
- **Testing pyramid:** domain unit tests (table-driven, high coverage) → store/app integration tests (real Postgres) → HTTP handler tests → a few e2e journeys (login → placement → recommendation → quiz → flag).
- **Config:** 12-factor env vars; `.env.example` documents all. No secrets in repo.
- **Observability:** `log/slog` JSON logs with request IDs, `/healthz` (liveness) and `/readyz` (DB ping).
- **Security:** invite-only OIDC, hashed session/invite tokens, Origin check, parameterized SQL only, answer keys never sent before answering, MCP token scoped to server-to-server.
