# Control Plane Notebook - Architecture Guide

## Why This Version Exists

This project should come before Ledger API.

Control Plane Notebook forces you through the part of backend work that JSON APIs let you postpone: templates, forms, redirects, sessions, cookies, CSRF, access control, and page-oriented handler design. If Ledger API makes you think like an API engineer, this project makes you think like a full backend engineer.

This guide is written in the same build-first style as the ConcurFlow and Ledger guides. It is not a full implementation. It is the mental scaffolding for one.

## What This Project Actually Is

Control Plane Notebook is an internal documentation and incident runbook system for teams.

Think:

- users sign in with session-backed auth
- teams own and view documents
- authors create drafts
- reviewers approve or reject changes
- documents can be tagged, searched, moderated, and audited
- pages are server-rendered HTML, not a JSON frontend talking to an API

Do not think:

- personal note-taking app
- wiki with no permissions
- JSON-first SPA
- simple CRUD over one table

This project is closer to "internal knowledge system for operations teams" than to "blog".

## How This Relates to Your Current `ministore`

Your current `ledger/ministore` app already has the tiniest useful skeleton:

- `main.go`
- `routes.go`
- `handlers.go`
- `helpers.go`
- `internal/data`

That skeleton is good for learning HTTP structure, but it is still a JSON toy app with in-memory data. Control Plane Notebook keeps the same general separation of concerns and grows it into:

- server-rendered pages
- PostgreSQL-backed storage
- generated typed queries with `sqlc`
- sessions and cookies
- access-control aware handlers
- workflow and audit logic

## One Shared Example for the Whole Guide

Use one running example while reading the rest of this document:

- Alice belongs to the `platform` team.
- Alice signs in and creates a draft runbook called `Rotate API keys`.
- The draft is tagged `security` and marked `team` visibility.
- Alice submits the draft for approval.
- Bob, a reviewer, approves it.
- The published page becomes visible to the `platform` team.
- Later, a moderator hides the page temporarily because the content contains stale production steps.
- Every action is recorded in the audit log.

If the architecture is good, that one story should move cleanly through the whole system.

## Proposed Project Shape

This is a strong target layout for a first serious version:

```text
ministore/
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
  internal/auth/
    permissions.go
  internal/data/
    store.go
    tx.go
    sqlc/
    queries/
      users.sql
      teams.sql
      notebooks.sql
      approvals.sql
      audit.sql
      moderation.sql
  internal/validator/
    validator.go
  ui/html/
    base.tmpl
    partials/
    pages/
  ui/static/
    css/
    js/
  migrations/
    000001_create_users_and_teams.sql
    000002_create_sessions.sql
    000003_create_notebooks.sql
    000004_create_search_and_tags.sql
    000005_create_approvals_audit_and_moderation.sql
  sqlc.yaml
```

If you keep the current `cmd/ministore` name for now, that is fine. But this project will read much more clearly if the server-rendered app lives under `cmd/web` or `cmd/notebook`.

## Suggested Database Backbone

The easiest way to keep this project coherent is to model stable identities separately from editable revisions.

### Core tables

- `users`
  - identity, email, password hash, active flag
- `teams`
  - human grouping and ownership boundary
- `team_memberships`
  - join table plus role information
- `documents`
  - stable identity for a notebook page
- `document_revisions`
  - draft and published content versions
- `tags`
  - normalized tag list
- `document_tags`
  - many-to-many join
- `approvals`
  - reviewer decisions on submitted revisions
- `audit_events`
  - append-only record of security and content changes
- `moderation_flags`
  - moderation queue entries
- `sessions`
  - server-side session store if you use `scs/pgxstore`

### Why split `documents` and `document_revisions`

Because drafts, approvals, audit history, and published content all become easier when:

- `documents` represents the stable page identity, ownership, and visibility
- `document_revisions` represents editable content snapshots and workflow state

Without this split, drafts and approvals turn into fragile "overwrite one row and hope nothing breaks" logic.

## Phase 0 - Foundation and Wiring

Build the system shell first.

### `cmd/web/main.go`

**Purpose**

`main.go` is the entry point and shared dependency setup.

**Role in the system**

It builds the logger, config, `pgxpool`, session manager, template cache, and application struct.

**Mental model**

Think of it as the control room. It turns the building on and wires the shared services together.

**What belongs here**

- config flags
- logger setup
- PostgreSQL pool creation
- `sqlc` store wiring
- session manager setup
- template cache setup
- `http.Server` construction
- graceful shutdown and timeouts

**What does not belong here**

- page-specific logic
- SQL queries
- per-route auth decisions

### `cmd/web/routes.go`

**Purpose**

`routes.go` declares the HTTP surface area.

**Role in the system**

It groups routes by access level and feature area.

**Mental model**

Think of it as the site map plus security gates.

**What belongs here**

- public routes
  - home
  - login
- authenticated routes
  - notebook list
  - notebook create/edit
  - approvals queue
  - search
- elevated routes
  - admin audit pages
  - moderation pages
- middleware grouping

**What does not belong here**

- form parsing
- permission checks for a specific document
- SQL

### `cmd/web/middleware.go`

**Purpose**

`middleware.go` owns cross-cutting request behavior.

**Role in the system**

Every request passes through it before page handlers run.

**Mental model**

Think of it as the corridor every visitor walks through before entering a room.

**What belongs here**

- panic recovery
- structured request logging
- secure headers
- session loading
- authentication guards
- CSRF middleware
- current-user lookup

**What does not belong here**

- notebook business rules
- SQL for page data

### `cmd/web/templates.go`

**Purpose**

`templates.go` owns template parsing and caching.

**Role in the system**

It gives handlers a fast, consistent way to render HTML pages with shared layout and partials.

**Mental model**

Think of it as the print shop for every HTML response.

**What belongs here**

- template cache creation
- custom template functions
- a helper like `render(w, status, page, data)`

**What does not belong here**

- business data loading
- session mutation

### `internal/data/store.go`

**Purpose**

`store.go` is the entry point to the PostgreSQL data layer.

**Role in the system**

It exposes one coherent dependency around the `pgxpool` and generated `sqlc` queries.

**Mental model**

Think of it as the database counter. Handlers place requests here; raw SQL does not leak back upward.

**What belongs here**

- wrapper around `*pgxpool.Pool`
- wrapper around generated `sqlc.Queries`
- shared data-layer errors

**What does not belong here**

- HTML rendering
- cookie management

### `sqlc.yaml`

**Purpose**

This file defines how SQL becomes generated Go code.

**Role in the system**

It is the contract between handwritten SQL and typed query code.

**Mental model**

Think of it as the compiler settings for your query layer.

**Important rule**

Treat `internal/data/queries/*.sql` as source code and `internal/data/sqlc/*` as generated artifacts. Do not hand-edit generated files.

### `migrations/`

**Purpose**

The migrations directory is the durable history of the database.

**Role in the system**

It guarantees that a new developer can rebuild the schema from scratch and land in the same state.

**Mental model**

Think of it as the database timeline.

**What belongs here**

- table creation
- indexes
- constraints
- seed data only if it is truly baseline data

**What does not belong here**

- ad hoc manual fixes
- queries used by handlers

## Phase 1 - Server-Rendered Transport Layer

This phase makes the project feel like a website instead of a JSON exercise.

### `cmd/web/helpers.go`

**Purpose**

`helpers.go` holds transport and rendering helpers.

**Role in the system**

It standardizes redirects, template rendering, flash messages, form errors, and common response behavior.

**Mental model**

Think of it as the adapter layer between raw HTTP and the UI.

**What belongs here**

- render helpers
- form helpers
- redirect helpers
- flash message helpers
- not-found and server-error helpers

**What does not belong here**

- template definitions
- SQL

### `ui/html/`

**Purpose**

This directory contains the actual HTML view layer.

**Role in the system**

It defines how notebook data, search results, audit entries, and auth pages are presented.

**Mental model**

Think of it as the frame and panels of the site. The handlers decide what data to send; the templates decide how it appears.

**Suggested layout**

- `base.tmpl`
- `partials/nav.tmpl`
- `partials/flash.tmpl`
- `partials/form_errors.tmpl`
- `pages/home.tmpl`
- `pages/login.tmpl`
- `pages/notebook_list.tmpl`
- `pages/notebook_view.tmpl`
- `pages/notebook_edit.tmpl`
- `pages/approvals_queue.tmpl`
- `pages/audit_log.tmpl`
- `pages/moderation_queue.tmpl`

**Important rule**

Keep authorization decisions out of templates when possible. Templates can branch on already-prepared view data, but they should not become mini permission engines.

## Phase 2 - Identity, Sessions, and Access Control

This phase turns the app from "pages over a database" into a real internal system.

### `cmd/web/auth.go`

**Purpose**

`auth.go` owns login, logout, and session-backed identity flows.

**Role in the system**

It translates forms and cookies into authenticated sessions.

**Mental model**

Think of it as the front desk: sign in, sign out, and remember who is in the building.

**What belongs here**

- login form page
- login submission handler
- logout handler
- password verification
- session renewal

**What does not belong here**

- global permission logic
- notebook queries

### `cmd/web/context.go`

**Purpose**

`context.go` centralizes request-scoped values like current user, current team, and CSRF-aware view data.

**Role in the system**

It gives every handler a consistent way to access identity and request metadata.

**Mental model**

Think of it as the typed pocket attached to each request.

### `internal/auth/permissions.go`

**Purpose**

`permissions.go` expresses authorization rules in one place.

**Role in the system**

Handlers ask it questions like:

- can this user view this document?
- can this user edit this draft?
- can this user approve this revision?
- can this user moderate this page?

**Mental model**

Think of it as the policy book.

**What belongs here**

- role checks
- team-membership checks
- visibility rules
- reviewer and moderator policy helpers

**What does not belong here**

- template rendering
- raw SQL

### `internal/data/queries/users.sql` and `teams.sql`

**Purpose**

These query files own the identity and membership model.

**Suggested data shape**

- `users`
- `teams`
- `team_memberships`

**Important constraints**

- unique user email
- unique team slug
- unique `(user_id, team_id)` membership
- foreign keys from memberships to users and teams

## Phase 3 - Notebook Core

This phase creates the heart of the product.

### `internal/data/queries/notebooks.sql`

**Purpose**

This query file owns notebook identities, revisions, tags, search metadata, and listing pages.

**Suggested document model**

`documents`

- stable identity
- owner
- team
- visibility scope
- current status
- current published revision pointer

`document_revisions`

- title
- body
- summary
- author
- workflow state
- created timestamp

`tags` and `document_tags`

- normalized tags
- many-to-many attachment

**Important indexes**

- unique slug for published documents
- index on `(team_id, visibility_scope, status)`
- GIN index for text search
- partial index for "currently reviewable" revisions if the workload justifies it

### `cmd/web/notebooks.go`

**Purpose**

`notebooks.go` owns page handlers for viewing, editing, listing, and searching notebook content.

**Role in the system**

It is the main feature surface for authors and readers.

**Mental model**

Think of it as the controller layer for the knowledge system.

**What belongs here**

- list page
- detail page
- create draft page
- edit draft page
- search page
- tag-filtered page
- ownership and visibility aware loading

**What does not belong here**

- login logic
- review queue logic
- raw SQL strings

**Critical UI pattern**

Use Post/Redirect/Get for every mutating form:

1. validate form input
2. perform the write
3. store a flash message
4. redirect to the resulting GET page

Without this, refreshes will accidentally repeat writes.

## Phase 4 - Approvals, Audit, and Moderation

This phase is what makes the project feel serious.

### `internal/data/queries/approvals.sql`

**Purpose**

This query file owns approval workflow transitions.

**Mental model**

Think of it as the state machine behind "submit", "approve", and "reject".

**Important rule**

Approval transitions should generally happen inside a transaction together with audit writes.

### `internal/data/queries/audit.sql`

**Purpose**

This file owns append-only audit history.

**Mental model**

Think of it as the black box recorder for the application.

**Events worth storing**

- login and logout
- document create
- revision submit
- approval and rejection
- visibility change
- moderation hide and restore
- role changes

### `internal/data/queries/moderation.sql`

**Purpose**

This file owns moderation queue behavior.

**Role in the system**

It lets admins or moderators flag, review, and resolve problematic content without destroying history.

### `cmd/web/approvals.go`

**Purpose**

`approvals.go` owns reviewer-facing pages and handlers.

**Role in the system**

It handles review queues, approval decisions, and rejection flows.

### `cmd/web/admin.go`

**Purpose**

`admin.go` owns audit views, moderation pages, and admin-only user or role management.

**Mental model**

Think of it as the operator console for governance.

## Phase 5 - Operational Readiness and Verification

This phase keeps the app maintainable.

### Logging

Use structured request logging from the start. You should be able to answer:

- who made the request
- what route was hit
- how long it took
- whether it succeeded
- which document or revision was affected

### SQL maturity

This project should force you to get comfortable with:

- named constraints
- foreign keys
- unique constraints
- partial indexes
- simple transactions
- reading `EXPLAIN`

### Tests

Good early tests include:

- handler tests for login, logout, and access guards
- handler tests for notebook create and edit flows
- template-rendering tests for expected page content
- data-layer tests for query behavior
- integration tests for approval plus audit transaction behavior

## Request Lifecycle in Plain English

When the project is healthy, a serious write flow should look like this:

1. Alice loads the edit page for a draft.
2. The request passes through session middleware and current-user middleware.
3. The handler loads the draft and checks whether Alice may edit it.
4. Alice submits the form.
5. The handler parses form data, validates it, and calls a transactional store method.
6. The store writes a new revision or updates the draft state.
7. An audit event is inserted in the same transaction.
8. The handler sets a flash message and redirects.
9. The GET page renders the new state from the database.

That is the basic loop of the whole application.

## Common Mistakes to Avoid

- Treating this like a JSON API with HTML pasted on top.
- Keeping only one mutable `documents` row and trying to bolt drafts and approvals onto it later.
- Hand-editing generated `sqlc` files.
- Rendering templates directly from many handlers without a central cache and render helper.
- Using sessions without rotating or renewing them after login.
- Skipping CSRF because the app is "internal".
- Mixing authorization rules into templates.
- Writing approval and audit records in separate non-transactional steps.
- Building search as permanent `%LIKE%` scans without learning how indexes and plans behave.

## Recommended Build Order

Build in this order:

1. app wiring, `pgxpool`, `sqlc`, migrations, template cache
2. home page and base layout
3. login and logout with sessions
4. users, teams, and membership-aware middleware
5. notebook list and detail pages
6. create and edit draft flow with forms and PRG
7. search and tags
8. approval queue and review actions
9. audit and moderation pages
10. SQL review, `EXPLAIN`, testing, and hardening

By the end, you should be comfortable building a real server-rendered backend, not just a JSON service.
