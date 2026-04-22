# ControlPlane

ControlPlane is a server-rendered Go web application for internal runbooks, incident documentation, and operational knowledge sharing. It is being built as a serious backend engineering project: HTML templates, forms, sessions, PostgreSQL, access control, approvals, audit history, and moderation, all without hiding behind a frontend SPA or a JSON-only API.

## Why This Project Exists

This project is meant to teach the part of backend development that JSON APIs let you postpone:

- server-rendered page architecture
- form handling and validation
- session-backed authentication
- cookies and CSRF-aware flows
- database-backed page rendering
- role and team based access control
- auditability and operational governance

The goal is to end up with something that feels closer to an internal production tool than to a tutorial CRUD app.

## Product Vision

ControlPlane is an internal notebook and runbook system for teams.

The intended workflow looks like this:

- a user signs in
- creates or edits a draft runbook
- assigns tags, ownership, and visibility scope
- submits it for review
- a reviewer approves or rejects it
- the published version becomes visible to the right team
- moderation and audit history capture the important actions along the way

This is not a blog, not a todo app, and not a public wiki. It is an operational knowledge system with governance.

## Current Status

The project is in early foundation work.

Right now the repo contains the initial app shell:

- Go module setup
- `chi` router wiring
- PostgreSQL connection pool setup with `pgxpool`
- session management setup direction with `scs`
- architecture and study docs for the build plan

The next implementation milestone is Phase 0 and Phase 1 wiring:

- move all shared dependencies behind the `application` struct
- attach `scs.LoadAndSave` correctly in the router pipeline
- add a template cache and render helper
- render a first server-side page cleanly

## Tech Stack

- Go
- `net/http`
- `github.com/go-chi/chi/v5`
- `github.com/jackc/pgx/v5/pgxpool`
- `github.com/alexedwards/scs/v2`
- `github.com/alexedwards/scs/pgxstore`
- PostgreSQL

Planned additions:

- `html/template` for server rendering
- `sqlc` for typed query generation
- SQL migrations
- structured request logging
- role and permission middleware

## Architecture Direction

The architecture is intentionally conservative and explicit:

- `main.go` builds the long-lived dependencies once
- an `application` struct carries shared dependencies like config, DB pool, session manager, and template cache
- `app.routes()` builds the router and middleware graph
- handlers are methods on `app`
- request-scoped data lives in `*http.Request` and `Context`, not on global variables

That separation is the core design rule of the app.

## Planned Capabilities

- user accounts and team membership
- session-backed authentication
- role and permission checks
- server-rendered notebook pages
- drafts and published revisions
- approval workflow
- audit history
- moderation queue
- tags and ownership metadata
- search and filtering

## Project Structure

Current shape:

```text
ControlPlane/
  cmd/web/
    main.go
    routes.go
  docs/
    Control Plane Notebook Architecture.md
    Control Plane Notebook Reading Map.md
    Control Plane Notebook vs Ledger API.md
    Phase 0-1 Wiring Recipe.md
  .env
  .gitignore
  go.mod
```

Target shape:

```text
ControlPlane/
  cmd/web/
    main.go
    routes.go
    middleware.go
    helpers.go
    templates.go
    context.go
    home.go
    auth.go
    notebooks.go
    approvals.go
    admin.go
  internal/
    auth/
    data/
    validator/
  ui/
    html/
    static/
  migrations/
  sqlc.yaml
  docs/
```

## Local Development

### Prerequisites

- Go `1.25+`
- PostgreSQL

### Environment variables

The current app expects:

```env
ENVIRONMENT=development
DATABASE_URL=postgres://user:password@localhost:5432/controlplane
SESSION_SECRET=change-me
```

### Run

```bash
go run ./cmd/web
```

The app currently listens on `:8080`.

## Development Roadmap

### Phase 0

- clean application wiring
- shared dependency injection
- DB connectivity checks
- session manager initialization
- router and middleware ownership cleanup

### Phase 1

- template cache
- render helper
- home page
- session-backed flash/example state

### Phase 2

- users, teams, memberships
- auth flows
- form handling
- CSRF-aware write flows

### Phase 3

- notebooks, revisions, tags, visibility scopes
- approvals and publishing workflow

### Phase 4

- audit history
- moderation
- search
- operational hardening and tests

## Learning Goals

This repository is deliberately being used to practice:

- dependency injection in Go
- request lifecycle design
- middleware composition
- relational schema design
- production-shaped SQL habits
- testable handler design
- server-rendered backend architecture

## Docs

Project docs live in [`docs/`](./docs):

- [Architecture Guide](./docs/Control%20Plane%20Notebook%20Architecture.md)
- [Reading Map](./docs/Control%20Plane%20Notebook%20Reading%20Map.md)
- [Notebook vs Ledger API](./docs/Control%20Plane%20Notebook%20vs%20Ledger%20API.md)
- [Phase 0-1 Wiring Recipe](./docs/Phase%200-1%20Wiring%20Recipe.md)

## Notes

This repository is still in active build-out, so the architecture is ahead of the implementation right now. That is intentional. The point is to build the app with a clear mental model rather than growing it by accident.
