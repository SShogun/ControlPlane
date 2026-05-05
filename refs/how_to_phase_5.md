# How To Build Phase 5: Tests, CI, Security, and Production Habits

Phase 5 is where Control Plane Notebook stops being "it works on my machine" and starts becoming a backend project you can trust. You do not need to implement everything at once. Use this file as a study map and checklist: read the resource for a topic, implement the matching small task, then run the commands before moving on.

## 1. What Phase 5 Means

Phase 5 is not a feature phase. It is a reliability phase.

The goal is to prove that the app works, protect the important web flows, and make failures visible. For this project, Phase 5 means adding tests, CI, validation, CSRF protection, safer cookies, better config handling, rate limits where useful, structured logs, and beginner-friendly documentation for running all of it.

Do Phase 5 in this order:

1. Unit tests for pure helpers and handler behavior.
2. Store/integration tests with PostgreSQL.
3. CI that runs build and tests automatically.
4. Validation and form error handling.
5. Security hardening: CSRF, cookies, secrets, auth/session safety.
6. Observability: logs first, then metrics/tracing later.
7. Documentation updates.

## 2. Resource Library

Use these resources as your "before implementing" reading list. Every checklist item later in this file points back to one or more of these resources.

### Go Testing Basics

- Go `testing` package: https://pkg.go.dev/testing
- Go `httptest` package: https://pkg.go.dev/net/http/httptest
- Go blog, subtests and table-driven tests: https://go.dev/blog/subtests
- Go fuzzing docs: https://go.dev/doc/security/fuzz/

What to learn:

- Test files end in `_test.go`.
- Test functions are named `TestSomething(t *testing.T)`.
- Table-driven tests let you test many inputs with one test.
- `httptest.NewRecorder()` captures a handler response.
- `httptest.NewRequest()` creates fake HTTP requests.

### Database and Store Testing

- `pgxpool` docs: https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool
- Docker Compose docs: https://docs.docker.com/compose/
- Docker Compose services and healthchecks: https://docs.docker.com/reference/compose-file/services/
- GitHub Actions PostgreSQL service containers: https://docs.github.com/actions/using-containerized-services/creating-postgresql-service-containers

What to learn:

- Store tests should run against a real PostgreSQL database when possible.
- Migrations should prepare the schema before tests run.
- Tests should create their own data and clean up after themselves.
- CI can start PostgreSQL as a service container.

### SQL and sqlc

- sqlc documentation: https://docs.sqlc.dev/
- sqlc getting started: https://docs.sqlc.dev/en/stable/tutorials/getting-started.html

What to learn:

- SQL files are source code.
- `sqlc generate` should be repeatable.
- CI should catch generated-code drift if you rely on generated query code.

### GitHub Actions and CI

- GitHub Actions docs: https://docs.github.com/en/actions
- Workflow syntax: https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions
- Go setup action: https://github.com/actions/setup-go
- Official golangci-lint action: https://github.com/golangci/golangci-lint-action

What to learn:

- A workflow lives in `.github/workflows/*.yml`.
- CI should run on `push` and `pull_request`.
- Start simple: `go test ./...` and `go build ./...`.
- Add linting only after build and tests are stable.

### Sessions and Cookies

- `scs` GitHub repo: https://github.com/alexedwards/scs
- Alex Edwards SCS article: https://www.alexedwards.net/blog/scs-session-manager
- OWASP Session Management Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html

What to learn:

- `LoadAndSave` must wrap requests before handlers use sessions.
- Login should renew the session token.
- Cookies should be `HttpOnly`.
- Production cookies should be `Secure`.
- Session lifetime should be intentional, not accidental.

### CSRF Protection

- OWASP CSRF Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
- `gorilla/csrf` package docs: https://pkg.go.dev/github.com/gorilla/csrf
- `gorilla/csrf` GitHub repo: https://github.com/gorilla/csrf
- `nosurf` package docs: https://pkg.go.dev/github.com/justinas/nosurf

What to learn:

- CSRF matters because browsers automatically send session cookies.
- Protect all state-changing routes: `POST`, `PUT`, `PATCH`, `DELETE`.
- Server-rendered forms need hidden CSRF token fields.
- CSRF secrets must be stable across restarts and must not be hardcoded.

Recommended direction for this project:

- Use one CSRF middleware globally or around web routes.
- Add a CSRF token/field to `templateData`.
- Render the token inside every POST form.
- Keep CSRF secret in an environment variable.

### Validation and Safe HTML

- Go `html/template` docs: https://pkg.go.dev/html/template
- OWASP XSS Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html
- OWASP Input Validation Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html

What to learn:

- Validation means rejecting bad input before writing it to the database.
- Escaping means safely displaying data in HTML.
- `html/template` escapes output by default.
- Do not switch to `template.HTML` unless you fully trust and sanitize that content.

Recommended direction for this project:

- Start with a small `internal/validator` package.
- Validate title length, body length, required fields, and route IDs.
- Keep validation errors in form structs or `templateData.Errors`.

### Rate Limiting

- Go `x/time/rate` docs: https://pkg.go.dev/golang.org/x/time/rate
- OWASP Authentication Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html

What to learn:

- Rate limiting protects expensive or sensitive endpoints.
- Login is the first endpoint to protect.
- A simple in-memory limiter is fine for learning.
- Distributed production deployments need shared rate limit state.

Recommended direction for this project:

- Add a small login rate limiter later, after CSRF and validation.
- Keep it simple: limit repeated failed login attempts by IP address.

### Logging, Metrics, and Tracing

- Go `log/slog` docs: https://pkg.go.dev/log/slog
- Prometheus `promhttp` docs: https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/promhttp
- OpenTelemetry Go docs: https://opentelemetry.io/docs/languages/go/
- OpenTelemetry Go getting started: https://opentelemetry.io/docs/languages/go/getting-started/

What to learn:

- Logs tell you what happened.
- Metrics tell you how often and how slowly things happen.
- Traces show request flow across systems.
- For this beginner app, start with structured logs before adding metrics/tracing packages.

Recommended direction for this project:

- Keep using `slog`.
- Add request ID to logs.
- Log server errors, auth failures, approval decisions, and database failures.
- Add Prometheus/OpenTelemetry only after tests and security basics are in place.

## 3. Phase 5 Implementation Checklist

Each item includes the resources you should read before implementing it.

### A. Test Foundation

- [ ] Read: Go `testing`, Go `httptest`, Go blog on subtests.
- [ ] Create your first test file, probably `cmd/web/helpers_test.go`.
- [ ] Test simple helpers first, such as `readIDParam` behavior for good and bad IDs.
- [ ] Add table-driven tests for validation rules once validation exists.
- [ ] Run `go test ./...` locally.
- [ ] Keep tests small and boring at first.

Done means:

- `go test ./...` runs successfully.
- At least one web helper or handler has a real test.

### B. Handler Tests

- [ ] Read: `httptest` docs.
- [ ] Create fake dependencies for `Application`, especially a fake store.
- [ ] Test that `GET /login` returns `200 OK`.
- [ ] Test that unauthenticated `GET /notebooks` redirects to `/login`.
- [ ] Test that `POST /login` with bad credentials redirects back to `/login`.
- [ ] Test that a handler renders expected status codes, not exact full HTML.

Done means:

- You can test handlers without starting the real server.
- Handler tests do not require a real database unless they are explicitly integration tests.

### C. Store and Integration Tests

- [ ] Read: `pgxpool`, Docker Compose, GitHub Actions PostgreSQL service containers.
- [ ] Decide on a local test DB strategy: Docker Compose is the clearest beginner path.
- [ ] Create a test database separate from development data.
- [ ] Apply migrations before running integration tests.
- [ ] Write tests for `CreateDraft`, `CreateNotebookRevision`, `ListNotebookRevisions`, and `UpdateRevisionStatus`.
- [ ] Use `t.Cleanup` to delete created rows or reset the DB.
- [ ] Keep integration tests opt-in at first if setup is heavy.

Done means:

- Store methods are tested against PostgreSQL.
- Tests do not depend on data that already exists on your machine.

### D. CI Pipeline

- [ ] Read: GitHub Actions docs, workflow syntax, setup-go action.
- [ ] Create `.github/workflows/ci.yml`.
- [ ] Add `actions/checkout`.
- [ ] Add `actions/setup-go`.
- [ ] Run `go test ./...`.
- [ ] Run `go build ./...`.
- [ ] Add a PostgreSQL service container when integration tests need DB access.
- [ ] Add `sqlc generate` only when sqlc generation is stable and intentional.
- [ ] Add golangci-lint after tests and build are green.

Done means:

- Every push and PR runs build and tests.
- A broken build fails before code is merged.

### E. Validation

- [ ] Read: OWASP Input Validation Cheat Sheet.
- [ ] Create `internal/validator/validator.go`.
- [ ] Add helper methods like `Check`, `Valid`, and `AddFieldError`.
- [ ] Validate notebook title is required.
- [ ] Validate notebook title max length.
- [ ] Validate notebook body max length.
- [ ] Validate route IDs are positive integers.
- [ ] Return validation errors through form data or `templateData.Errors`.
- [ ] Add tests for all validation rules.

Done means:

- Bad form input returns `422 Unprocessable Entity`.
- Users see field errors.
- Invalid data does not reach the database.

### F. CSRF Protection

- [ ] Read: OWASP CSRF Cheat Sheet.
- [ ] Read either `gorilla/csrf` or `nosurf` docs.
- [ ] Choose one CSRF library. Do not write your own CSRF system first.
- [ ] Add a CSRF secret to config from an environment variable.
- [ ] Add CSRF middleware to the router.
- [ ] Add CSRF token data to `templateData`.
- [ ] Add the hidden CSRF field to every POST form template.
- [ ] Test that POST without a token fails.
- [ ] Test that POST with a valid token succeeds.

Done means:

- All state-changing form submissions are protected.
- Tokens are not hardcoded.
- The app still works through normal forms.

### G. Session and Cookie Hardening

- [ ] Read: SCS docs and OWASP Session Management Cheat Sheet.
- [ ] Ensure `HttpOnly` is true.
- [ ] Ensure `SameSite` is intentionally set.
- [ ] Ensure `Secure` is true in production.
- [ ] Keep session lifetime explicit.
- [ ] Renew session token after login.
- [ ] Destroy the session on logout.
- [ ] Do not store full user structs in the session; store only `userID`.

Done means:

- Session settings are clearly configured in `main.go`.
- Development and production cookie settings are not accidentally the same.

### H. Secret Management

- [ ] Read: GitHub Actions secrets docs: https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions
- [ ] Move secrets out of source code.
- [ ] Use environment variables for `DATABASE_URL`.
- [ ] Add a CSRF secret env var.
- [ ] Document required env vars in README.
- [ ] Do not commit real `.env` secrets.
- [ ] Add `.env.example` with fake values if useful.

Done means:

- A new developer knows which env vars to set.
- Real secrets are not in git.

### I. Rate Limiting

- [ ] Read: Go `x/time/rate` docs.
- [ ] Start with login rate limiting only.
- [ ] Decide the key: IP address is simplest.
- [ ] Add a small middleware or login-specific guard.
- [ ] Return `429 Too Many Requests` when the limit is exceeded.
- [ ] Add tests for allowed and blocked requests.

Done means:

- Repeated login attempts are slowed down.
- Normal usage is not affected.

### J. Observability

- [ ] Read: `log/slog` docs.
- [ ] Add consistent structured logs for startup.
- [ ] Log database connection failures.
- [ ] Log handler server errors through `app.serverError`.
- [ ] Include request ID where possible.
- [ ] Later, read Prometheus `promhttp` docs.
- [ ] Later, add `/metrics` if you are ready to introduce Prometheus.
- [ ] Later, read OpenTelemetry Go docs if you want tracing.

Done means:

- When something fails, logs tell you which layer failed.
- Metrics/tracing are optional, not blocking early Phase 5.

### K. Documentation

- [ ] Update README with local setup.
- [ ] Document `DATABASE_URL`.
- [ ] Document test commands.
- [ ] Document how to run integration tests.
- [ ] Document CI expectations.
- [ ] Document security decisions: CSRF, cookies, sessions.
- [ ] Keep `optimized.md` and this file as learning guides, not runtime docs.

Done means:

- A future you can clone the repo, set env vars, run tests, and understand the safety checks.

## 4. Suggested Implementation Order

Do not start with the hardest thing. Use this order:

1. Add one tiny unit test.
2. Add handler tests for redirects and status codes.
3. Add validation helpers and validation tests.
4. Add CI with `go test ./...` and `go build ./...`.
5. Add CSRF middleware and update forms.
6. Add CSRF tests.
7. Add Docker/Postgres integration tests.
8. Add GitHub Actions PostgreSQL service container.
9. Harden session/cookie config.
10. Add login rate limiting.
11. Improve structured logs.
12. Update README.

## 5. Minimal First Milestone

If you want a small first win, implement only this:

- [ ] `cmd/web/helpers_test.go` with tests for route ID parsing or template data.
- [ ] One handler test for unauthenticated `/notebooks`.
- [ ] `.github/workflows/ci.yml` running `go test ./...` and `go build ./...`.
- [ ] README section: "Run tests".

This is enough to say Phase 5 has started.

## 6. Security Implementation Notes For This App

Your current app is server-rendered and session-cookie based. That means CSRF is not optional once POST forms exist.

Protect these routes first:

- `POST /login`
- `POST /logout`
- `POST /notebooks/new`
- `POST /notebooks/{id}/edit`
- `POST /approvals/approve`
- `POST /approvals/reject`

Cookie settings to aim for:

- `HttpOnly: true`
- `SameSite: Lax` or `Strict`; this app currently uses `Strict`
- `Secure: true` in production
- explicit lifetime, already present through `sessionManager.Lifetime`

Validation rules to add first:

- notebook title is required
- notebook title has a max length
- notebook body has a max length
- IDs from routes/forms must be positive integers
- approval/rejection IDs must be valid before touching the store

Logging to add first:

- server startup with port and environment
- database connection success/failure
- login success/failure without logging passwords
- approval/rejection actions
- all calls to `app.serverError`

## 7. How To Use Gemini Or Another Assistant Safely

When asking Gemini to help, give it a very small task and paste the relevant file. Avoid asking it to "implement Phase 5" in one shot.

Good prompts:

- "Write table-driven tests for this validator package. Do not change production code."
- "Review this handler test and explain why it does or does not need a real session manager."
- "Help me add gorilla/csrf to this chi router. Keep changes minimal."
- "Create a GitHub Actions workflow that runs `go test ./...` and `go build ./...` for this Go module."

Bad prompts:

- "Make the app production ready."
- "Add all security."
- "Refactor the whole project."
- "Rewrite my data layer."

After Gemini gives code:

- [ ] Read every changed file.
- [ ] Check whether it added new dependencies.
- [ ] Run `gofmt`.
- [ ] Run `go test ./...`.
- [ ] Run `go build ./...`.
- [ ] Make sure it did not change migrations or generated files by accident.

## 8. Final Phase 5 Definition Of Done

Phase 5 is done when:

- [ ] `go test ./...` passes locally.
- [ ] `go build ./...` passes locally.
- [ ] GitHub Actions runs tests and build on push/PR.
- [ ] At least core handlers have tests.
- [ ] Core store methods have integration tests.
- [ ] Forms have server-side validation.
- [ ] POST routes have CSRF protection.
- [ ] Production cookies are secure.
- [ ] Secrets are read from environment variables.
- [ ] Login has basic rate limiting.
- [ ] Logs are structured and useful.
- [ ] README explains setup, tests, CI, and required env vars.

When this checklist is done, Control Plane Notebook will not just have features. It will have guardrails.
