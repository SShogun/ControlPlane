<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Status-Phase%205-blueviolet?style=for-the-badge" alt="Phase 5" />
  <img src="https://img.shields.io/badge/CI-GitHub%20Actions-2088FF?style=for-the-badge&logo=githubactions&logoColor=white" alt="CI" />
</p>

# ControlPlane

A server-rendered Go web application for internal runbooks, incident documentation, and operational knowledge sharing. Built as a serious backend engineering project: HTML templates, forms, sessions, PostgreSQL, access control, approvals, audit history, and moderation — all without hiding behind a frontend SPA or a JSON-only API.

The project currently supports:

- session-backed authentication with login/logout flows
- server-rendered pages with a startup template cache
- notebook drafts, revisions, and content versioning
- tag-based categorization and search
- approval workflow with submit → approve/reject state transitions
- role and team-based access control (member, reviewer, admin)
- transactional audit logging for all governance-critical actions
- moderation flags for content governance
- handler-level unit tests with a fake store abstraction
- GitHub Actions CI pipeline with automated test and build verification

To make the implementation easier to follow, the sections below include code excerpts from the actual codebase so the architecture description maps directly to real files and logic.

---

## Table of Contents

- [Why This Project Exists](#why-this-project-exists)
- [Product Vision](#product-vision)
- [Current Status](#current-status)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Key Components](#key-components)
- [Database Schema](#database-schema)
- [API Routes](#api-routes)
- [Testing](#testing)
- [CI/CD Pipeline](#cicd-pipeline)
- [Local Development](#local-development)
- [Development Roadmap](#development-roadmap)
- [Learning Goals](#learning-goals)
- [Docs](#docs)

---

## Why This Project Exists

This project is meant to teach the part of backend development that JSON APIs let you postpone:

- server-rendered page architecture
- form handling and validation
- session-backed authentication
- cookies and CSRF-aware flows
- database-backed page rendering
- role and team-based access control
- auditability and operational governance

The goal is to end up with something that feels closer to an internal production tool than to a tutorial CRUD app.

---

## Product Vision

ControlPlane is an internal notebook and runbook system for teams.

The intended workflow looks like this:

1. A user signs in
2. Creates or edits a draft runbook
3. Assigns tags, ownership, and visibility scope
4. Submits it for review
5. A reviewer approves or rejects it
6. The published version becomes visible to the right team
7. Moderation and audit history capture the important actions along the way

This is not a blog, not a todo app, and not a public wiki. It is an operational knowledge system with governance.

---

## Current Status

> **Phase 5 — Production Hardening** (in progress)

Phase 5 test files are complete. CI pipeline is implemented and currently being refined.

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 0 | Foundation, wiring, shared dependencies | ✅ Complete |
| Phase 1 | First server-rendered page, template cache | ✅ Complete |
| Phase 2 | Forms, validation, sessions, auth, context | ✅ Complete |
| Phase 3 | Notebooks, revisions, tags, search, DB-backed pages | ✅ Complete |
| Phase 4 | Approvals, audit, moderation, governance | ✅ Complete |
| Phase 5 | Tests, CI, production hardening | 🔧 In Progress |

---

## Architecture

The architecture is intentionally conservative and explicit:

- `main.go` builds the long-lived dependencies once
- an `Application` struct carries shared dependencies (config, DB pool, session manager, template cache)
- `app.routes()` builds the router and middleware graph
- handlers are methods on `app`
- request-scoped data lives in `*http.Request` and `Context`, not on global variables

```mermaid
flowchart TD
    main["main.go"] --> deps["Build Dependencies\nConfig | Logger | DB Pool\nSession Manager | Template Cache"]
    deps --> app["Application struct"]
    app --> router["app.routes()\nchi Router"]
    
    router --> mw["Middleware Pipeline\nRequestID → RealIP → Logger\nRecoverer → LoadAndSave → authenticate"]
    
    mw --> public["Public Routes\n/ | /login"]
    mw --> protected["Protected Routes\n/dashboard | /notebooks/*\n/approvals/*"]
    mw --> admin["Admin Routes\n/admin/audit"]
    
    protected --> handlers["Handlers\nparse input → call store\nprepare data → render/redirect"]
    handlers --> store["internal/data\nUserStore Interface\nPgxStore Implementation"]
    store --> pg[("PostgreSQL\nusers | teams | notebooks\nrevisions | tags | audit")]
    
    handlers --> tmpl["Template Cache\nhtml/template\nServer-side rendering"]
```

### Request Lifecycle

```
Browser
  → GET /notebooks/1
  → chi router
  → RequestID → RealIP → Logger → Recoverer
  → scs LoadAndSave (session hydration)
  → authenticate (load user from session → context)
  → requireAuthentication (guard)
  → app.notebookView handler
  → store.ListNotebookRevisions + store.ListNotebookTags
  → app.render (template execution → buffered write)
  → HTML response
```

### Two Kinds of State

| Concept | Scope | Storage | Example |
|---------|-------|---------|---------|
| Application | server lifetime | process memory | DB pool, session manager, template cache |
| Session | many requests | DB/cookie-backed | `userID`, flash message |
| Context | one request | memory | current loaded user pointer |

---

## Tech Stack

| Category | Technology |
|----------|------------|
| Language | Go 1.25 |
| HTTP Router | [`chi/v5`](https://github.com/go-chi/chi) |
| Database | PostgreSQL via [`pgx/v5`](https://github.com/jackc/pgx) with `pgxpool` |
| Sessions | [`scs/v2`](https://github.com/alexedwards/scs) with `pgxstore` backend |
| Templates | `html/template` (server-rendered) |
| Query Gen | [`sqlc`](https://sqlc.dev) for typed query generation |
| Auth | bcrypt password hashing via `golang.org/x/crypto` |
| CI | GitHub Actions |

---

## Project Structure

```text
ControlPlane/
├── cmd/web/
│   ├── main.go                    # Entry point, dependency wiring, server startup
│   ├── routes.go                  # Route declarations, middleware graph, page handlers
│   ├── middleware.go              # authenticate, requireAuthentication, requireRole
│   ├── helpers.go                 # render, newTemplateData, logAuditEvent, serverError
│   ├── context.go                 # Typed request-context helpers (user get/set)
│   ├── auth.go                    # Login/logout handlers
│   ├── notebooks.go               # Notebook CRUD, search, view handlers
│   ├── approvals.go               # Approval queue, approve/reject handlers
│   ├── test_support_test.go       # Shared test infrastructure (fakeStore, helpers)
│   ├── handlers_test.go           # Handler unit tests
│   ├── auth_handlers_test.go      # Auth handler tests
│   ├── notebook_handlers_test.go  # Notebook handler tests
│   ├── approval_handlers_test.go  # Approval handler tests
│   ├── middleware_test.go         # Middleware unit tests
│   ├── helpers_test.go            # Helper function tests
│   └── context_test.go            # Context helper tests
├── internal/data/
│   ├── store.go                   # Domain types, UserStore interface
│   ├── pgxstore.go                # PostgreSQL implementation of UserStore
│   ├── queries/                   # Raw SQL for sqlc
│   │   ├── notebooks.sql
│   │   ├── approvals.sql
│   │   ├── approved.sql
│   │   ├── audit.sql
│   │   ├── teams.sql
│   │   └── moderation.sql
│   └── generated/                 # sqlc-generated Go code
│       ├── db.go
│       ├── models.go
│       ├── notebooks.sql.go
│       ├── approved.sql.go
│       └── audit.sql.go
├── migrations/
│   ├── 0001_initial_schema.sql    # Core tables, indexes, constraints
│   └── 0002_create_audit_logs.sql # Audit log table
├── ui/templates/                  # Server-rendered HTML templates
├── docs/                          # Architecture and study documentation
│   ├── Control Plane Notebook Architecture.md
│   ├── Control Plane Notebook Reading Map.md
│   ├── Control Plane Notebook vs Ledger API.md
│   ├── GO_BACKEND_WIRING_RECIPE_PHASE_0_TO_5.md
│   ├── Phase 0-1 Wiring Recipe.md
│   ├── Phase 2-5 Wiring Recipe.md
│   └── how_to_phase_5.md
├── .github/workflows/
│   └── ci.yml                     # GitHub Actions CI pipeline
├── sqlc.yaml                      # sqlc configuration
├── go.mod
├── go.sum
├── .env
└── .gitignore
```

---

## Key Components

### `Application` Struct — Dependency Injection

The `Application` struct is dependency injection in plain Go. It avoids globals, keeps handlers testable, and makes shared dependencies explicit.

```go
type Application struct {
    config         Config
    conn           *pgxpool.Pool
    logger         *slog.Logger
    store          data.UserStore
    sessionManager *scs.SessionManager
    templateCache  map[string]*template.Template
}
```

### `UserStore` Interface — Data Access Layer

All database operations are abstracted behind an interface, enabling fake implementations for testing:

```go
type UserStore interface {
    GetUserByEmail(ctx context.Context, email string) (User, error)
    GetUser(ctx context.Context, id int) (User, error)
    CheckPassword(user User, password string) bool
    ListNotebooks(ctx context.Context) ([]Notebook, error)
    CreateDraft(ctx context.Context, params CreateDraftParams) (int, error)
    CreateNotebookRevision(ctx context.Context, params CreateNotebookRevisionParams) (int, error)
    ListNotebookRevisions(ctx context.Context, notebookID int) ([]NotebookRevision, error)
    ListNotebookTags(ctx context.Context, notebookID int) ([]Tag, error)
    InsertAuditLog(ctx context.Context, params InsertAuditLogParams) error
    ApproveRevisionTx(ctx context.Context, revisionID, notebookID, reviewerID int) error
    ListSubmittedRevisions(ctx context.Context) ([]NotebookRevision, error)
    UpdateRevisionStatus(ctx context.Context, params UpdateRevisionStatusParams) error
    // ... and more
}
```

### Middleware Pipeline

Middleware order is intentional and matters:

```go
r.Use(middleware.RequestID)      // assign unique request ID
r.Use(middleware.RealIP)         // resolve client IP behind proxies
r.Use(middleware.Logger)         // structured request logging
r.Use(middleware.Recoverer)      // panic recovery
r.Use(app.sessionManager.LoadAndSave) // session hydration/commit
r.Use(app.authenticate)         // load current user into context
```

Then route-level guards:

```go
r.Use(app.requireAuthentication)  // block anonymous access
r.Use(app.requireRole("reviewer")) // enforce role-based access
r.Use(app.requireRole("admin"))    // admin-only routes
```

### Transactional Approval Workflow

Approval and audit happen atomically in a single database transaction to prevent partial writes:

```go
func (s *PgxStore) ApproveRevisionTx(ctx context.Context, revisionID, notebookID, reviewerID int) error {
    tx, err := s.DB.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // 1. Update revision status to 'approved'
    _, err = tx.Exec(ctx, updateRevisionQuery, revisionID)
    // 2. Set notebook's current published revision
    _, err = tx.Exec(ctx, updateNotebookQuery, revisionID, notebookID)
    // 3. Insert audit event
    _, err = tx.Exec(ctx, auditQuery, reviewerID, "revision_approved", "notebook_revision", revisionID)

    return tx.Commit(ctx)
}
```

---

## Database Schema

The relational model is designed around identity, content versioning, and governance:

```mermaid
erDiagram
    users ||--o{ memberships : "belongs to"
    teams ||--o{ memberships : "has"
    users ||--o{ notebook_revisions : "authors"
    notebooks ||--o{ notebook_revisions : "has versions"
    notebooks ||--o{ notebook_tags : "tagged with"
    tags ||--o{ notebook_tags : "applied to"
    notebook_revisions ||--o{ approvals : "reviewed in"
    users ||--o{ approvals : "reviews"
    notebooks ||--o{ moderation_flags : "flagged in"
    users ||--o{ audit_events : "performed by"

    users {
        bigserial id PK
        text email UK
        bytea password_hash
        timestamptz created_at
    }
    teams {
        bigserial id PK
        text name UK
    }
    notebooks {
        bigserial id PK
        text title
        text content
        text slug UK
        text visibility
        boolean is_published
        bigint current_published_revision_id FK
        bigint team_id FK
    }
    notebook_revisions {
        bigserial id PK
        bigint notebook_id FK
        bigint author_id FK
        text title
        text body
        text status
    }
    tags {
        bigserial id PK
        text name UK
    }
    approvals {
        bigserial id PK
        bigint notebook_revision_id FK
        bigint reviewer_id FK
        text decision
    }
    audit_events {
        bigserial id PK
        bigint actor_id FK
        text event_type
        text entity_type
        bigint entity_id
        jsonb details
    }
    moderation_flags {
        bigserial id PK
        bigint notebook_id FK
        bigint moderator_id FK
        text reason
        timestamptz resolved_at
    }
```

### Why Split `notebooks` and `notebook_revisions`?

`notebooks` is the stable identity — who owns it, what's the slug, what's published.
`notebook_revisions` is the editable content history — what title/body did this version have, who authored it, what's the workflow status.

Without this split, drafts and approvals become fragile because you keep overwriting the same row.

---

## API Routes

### Public Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/` | `home` | Landing page |
| `GET` | `/login` | `loginForm` | Login form |
| `POST` | `/login` | `loginSubmit` | Authenticate user |

### Authenticated Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/dashboard` | `dashboard` | User dashboard |
| `POST` | `/logout` | `logout` | Destroy session |
| `GET` | `/notebooks` | `listNotebooks` | List published notebooks |
| `GET` | `/notebooks/new` | `notebookCreateForm` | Create draft form |
| `POST` | `/notebooks/new` | `notebookCreateSubmit` | Submit new draft |
| `GET` | `/notebooks/{id}` | `notebookView` | View notebook with revision & tags |
| `GET` | `/notebooks/{id}/edit` | `notebookEditForm` | Edit draft form |
| `POST` | `/notebooks/{id}/edit` | `notebookEditSubmit` | Submit draft edit |
| `GET` | `/notebooks/search` | `notebooksSearch` | Search notebooks |

### Reviewer Routes (role: `reviewer`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/approvals` | `approvalQueue` | Submitted revisions queue |
| `POST` | `/approvals/approve` | `approveRevisionSubmit` | Approve and publish revision |
| `POST` | `/approvals/reject` | `rejectRevisionSubmit` | Reject revision |

### Admin Routes (role: `admin`)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/admin/audit` | `adminAudit` | Audit event log |

---

## Testing

Tests use a **fake store pattern** — a lightweight in-memory implementation of the `UserStore` interface — to verify handler behavior without touching the database.

### Test Architecture

```
fakeStore (implements UserStore)
  └── in-memory maps for users, notebooks, revisions, tags
  └── call tracking flags (e.g. createDraftCalled, approveCalled)

newTestApplication(store, templates)
  └── builds Application with fake store, real scs session manager, discard logger

serveWithSession(app, handler, request)
  └── wraps handler with LoadAndSave for valid session context
```

### Test Coverage

| File | Coverage Area | Tests |
|------|--------------|-------|
| `context_test.go` | Context user get/set helpers | 2 |
| `helpers_test.go` | `readIDParam`, template render | 2 |
| `middleware_test.go` | Auth redirect, role enforcement | 3 |
| `auth_handlers_test.go` | Login form, credentials, logout | 4 |
| `notebook_handlers_test.go` | Create draft, view, validation | 4 |
| `approval_handlers_test.go` | Approve/reject transactions | 2 |
| `handlers_test.go` | Full handler integration tests | 10+ |

### Running Tests

```bash
go test -v ./...
```

---

## CI/CD Pipeline

GitHub Actions runs on every push to `main` and on pull requests:

```yaml
name: CI Pipeline

on:
  push:
    branches: ["main"]
  pull_request:
    branches: ["main"]

jobs:
  test-and-build:
    runs-on: ubuntu-latest
    steps:
      - name: Check out repository code
        uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25.0'
      - name: Run Go tests
        run: go test -v ./...
      - name: Build application
        run: go build -v ./...
```

---

## Local Development

### Prerequisites

- Go `1.25+`
- PostgreSQL

### Environment Variables

Create a `.env` file at the project root:

```env
ENVIRONMENT=development
DATABASE_URL=postgres://user:password@localhost:5432/controlplane
SESSION_SECRET=change-me
```

### Setup

```bash
# Clone the repository
git clone https://github.com/SShogun/ControlPlane.git
cd ControlPlane

# Apply migrations
psql -d controlplane -f migrations/0001_initial_schema.sql
psql -d controlplane -f migrations/0002_create_audit_logs.sql

# Generate sqlc code (if modifying queries)
sqlc generate

# Run the application
go run ./cmd/web

# Run tests
go test -v ./...
```

The app listens on `:6767` by default.

---

## Development Roadmap

```mermaid
gantt
    title ControlPlane Build Phases
    dateFormat  YYYY-MM-DD
    axisFormat  %b

    section Phase 0
    Foundation & wiring                :done, p0, 2026-01-01, 14d

    section Phase 1
    Template cache & first page        :done, p1, after p0, 14d

    section Phase 2
    Auth, sessions, forms, CSRF        :done, p2, after p1, 21d

    section Phase 3
    Notebooks, revisions, tags, search :done, p3, after p2, 21d

    section Phase 4
    Approvals, audit, moderation       :done, p4, after p3, 21d

    section Phase 5
    Tests & CI                         :active, p5, after p4, 14d
    Production hardening               :        p5b, after p5, 14d
```

### Phase 0 — Foundation ✅

- Application struct and dependency injection
- PostgreSQL connection pool with `pgxpool`
- Session manager setup with `scs` and `pgxstore`
- Router wiring with `chi`

### Phase 1 — First Page ✅

- Template cache built at startup
- Buffered render helper
- Home page with session-backed flash messages
- End-to-end request flow proof

### Phase 2 — Auth & Forms ✅

- Login/logout with bcrypt password verification
- Session token renewal after authentication
- Request-context helpers for current user
- `authenticate` and `requireAuthentication` middleware
- `requireRole` middleware for role-based access
- Post/Redirect/Get pattern for form submissions

### Phase 3 — Notebook Domain ✅

- Notebook CRUD with draft creation
- Document/revision split for content versioning
- Tag system with normalized join table
- Notebook view with latest revision and tags
- Search endpoint

### Phase 4 — Governance ✅

- Approval queue for submitted revisions
- Approve/reject workflow with state transitions
- Transactional approval + audit in single DB transaction
- Audit event logging for all governance actions
- Moderation flags table
- Admin audit page

### Phase 5 — Production Hardening 🔧

- [x] Test infrastructure: `fakeStore`, `newTestApplication`, `serveWithSession`
- [x] Context helper tests
- [x] Helper function tests
- [x] Middleware tests (auth redirect, role enforcement)
- [x] Auth handler tests (login, logout, credentials)
- [x] Notebook handler tests (create, view, validation)
- [x] Approval handler tests (approve/reject transactions)
- [x] GitHub Actions CI pipeline
- [ ] CI pipeline refinement and edge cases
- [ ] Structured error handling improvements
- [ ] Request logging and observability
- [ ] Graceful shutdown

---

## Learning Goals

This repository is deliberately being used to practice:

- dependency injection in Go without frameworks
- request lifecycle design (middleware → handler → render)
- middleware composition and ordering
- relational schema design with proper normalization
- production-shaped SQL habits (transactions, indexes, constraints)
- testable handler design with interface-based fakes
- server-rendered backend architecture
- CI/CD pipeline design

---

## Docs

Project documentation lives in [`docs/`](./docs):

| Document | Description |
|----------|-------------|
| [Architecture Guide](./docs/Control%20Plane%20Notebook%20Architecture.md) | System design and component relationships |
| [Reading Map](./docs/Control%20Plane%20Notebook%20Reading%20Map.md) | Recommended study order |
| [Notebook vs Ledger API](./docs/Control%20Plane%20Notebook%20vs%20Ledger%20API.md) | Design comparison |
| [Phase 0-1 Wiring Recipe](./docs/Phase%200-1%20Wiring%20Recipe.md) | Foundation build guide |
| [Phase 2-5 Wiring Recipe](./docs/Phase%202-5%20Wiring%20Recipe.md) | Full feature build guide |
| [Full Wiring Recipe (Phase 0-5)](./docs/GO_BACKEND_WIRING_RECIPE_PHASE_0_TO_5.md) | Complete systems walkthrough |
| [How to Phase 5](./docs/how_to_phase_5.md) | Testing and hardening guide |

---

<p align="center">
  <sub>Built with intent, not with scaffolding.</sub>
</p>
