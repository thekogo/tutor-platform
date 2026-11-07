# Contributing

## Workflow: trunk-based

- `main` is always releasable. Branch protection requires the **CI OK** check and one review.
- Work on short-lived branches (hours to a couple of days): `feat/learning-path-api`, `fix/flag-dialog`.
- Rebase on `main` often, keep PRs small and focused. Squash-merge; the PR title becomes the commit message.
- Unfinished features hide behind config, not behind long-lived branches.

## Commits: Conventional Commits

```
<type>(<scope>): <summary in imperative mood>
```

Types: `feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`. Breaking changes: `feat(api)!: ...` plus a `BREAKING CHANGE:` footer.

Scopes: `api`, `backend`, `domain`, `store`, `mcp`, `auth`, `web`, `e2e`, `content`, `ci`, `deploy`, `docs`.

Examples: `feat(domain): weight repeat Answers at 0.3`, `fix(web): show Explanation after Flag`, `docs(adr): 0004 exam timing`.

Optional local hooks (gofmt, ESLint on staged files, commit message check): install [lefthook](https://lefthook.dev) and run `lefthook install`.

## Domain language

[CONTEXT.md](CONTEXT.md) is the ubiquitous language. Use its terms in code, API, UI copy and PRs (Learner, not user; Answer, not attempt; Flag, not report). If a change introduces or changes a domain concept:

1. Update CONTEXT.md in the same PR.
2. If the decision has lasting trade-offs (hard to reverse, surprising, or a real alternative was rejected), add an ADR in `docs/adr/` (`NNNN-short-title.md`: context, decision, consequences).

## Contract-first API changes

`api/openapi.yaml` is the source of truth for REST.

1. Edit `api/openapi.yaml` first (and nothing else) and get agreement on the shape.
2. Regenerate both sides: `make generate` (oapi-codegen strict server in backend, openapi-typescript types in frontend).
3. Implement the handler and the UI against the generated code.
4. Commit the spec and the generated code together. CI fails if generated code drifts from the spec.

Same for SQL: edit `backend/internal/store/queries/*.sql` or add a migration, then `make -C backend generate` (sqlc). Never edit an applied migration; add a new one.

MCP tools are part of the contract with the Owner's agent: document changes in `docs/architecture.md` (MCP tools table).

## Test pyramid

| Layer | Where | What |
|---|---|---|
| Domain unit tests (most) | `backend/internal/domain` | Pure, table-driven, fast; high coverage of mastery, recommendation, graph rules |
| Integration | `backend/internal/store`, `internal/app` | Real Postgres (`TEST_DATABASE_URL`), transactions, authorization |
| HTTP / MCP handler tests | `internal/httpapi`, `internal/mcpserver` | Request/response mapping, auth, error codes, no answer keys leaked |
| Frontend unit/component | `frontend/` (Vitest + Testing Library, MSW) | Components, hooks, rendering of Mastery Levels |
| E2E (few) | `e2e/` (Playwright) | Critical journeys only: login, dashboard, quiz + flag, mastery map, admin access |

Write the test at the lowest layer that can catch the bug. E2E locators: prefer roles and labels; use `data-testid` where Thai copy would make selectors brittle.

## Definition of done

- [ ] Behaviour matches CONTEXT.md vocabulary; CONTEXT.md / ADR updated if the domain changed
- [ ] API changes made spec-first; `make generate` committed; no drift
- [ ] Tests added at the right layer; `make test` and `make lint` pass
- [ ] E2E journey updated if a critical user flow changed
- [ ] Migrations are additive and reversible where possible
- [ ] No secrets, tokens or real Learner data in code, fixtures or logs
- [ ] Answer keys and Explanations are never sent before the Learner answers
- [ ] New env vars documented in `.env.example` (and README if user-facing)
- [ ] UI changes: screenshots in the PR; Thai copy reviewed
- [ ] CI green (`CI OK`)

## Local checks before pushing

```sh
make ci        # mirrors CI (minus Docker builds and e2e)
make e2e       # when touching user-visible flows
```
