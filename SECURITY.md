# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue. Report privately via GitHub's
[security advisory form](https://github.com/thekogo/tutor-platform/security/advisories/new)
or by email to the maintainer (see the GitHub profile of @thekogo).

Include what you found, steps to reproduce, and the impact you believe it has. You can expect an acknowledgement within 7 days. Please give us a reasonable window to fix the issue before disclosing it.

## Supported versions

Only the latest commit on `main` is supported.

## Scope highlights

Of particular interest:

- Authentication and session handling (Google OIDC, invite tokens, session cookies, `POST /api/auth/dev-login` being reachable when it should not be)
- Authorization: one Learner reading another Learner's Answers or Mastery; non-Owners reaching `/api/admin/*`
- Answer keys or Explanations exposed before a Learner answers
- MCP endpoint (`/mcp`) access without a valid `MCP_TOKEN`
- SQL injection, XSS through Markdown/KaTeX rendering, CSRF / Origin check bypass

## Deployment hardening (for operators)

- `AUTH_DEV_LOGIN=false` and `NEXT_PUBLIC_DEV_LOGIN=false` in production. Never deploy with `docker-compose.e2e.yml`.
- `SESSION_COOKIE_SECURE=true`, serve over HTTPS only, `APP_BASE_URL` set to the public HTTPS origin.
- `MCP_TOKEN`: at least 32 random bytes (`openssl rand -hex 32`), never committed, rotated if leaked. Expose `/mcp` only where your agent needs it.
- Use a strong Postgres password and do not publish the database port publicly.
- Keep dependencies current (Dependabot PRs; CI runs govulncheck, `pnpm audit` and CodeQL).
