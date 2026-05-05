# Go Backend Wiring Recipe: Phase 0 to Phase 5

This is the final stitched wiring recipe for Control Plane Notebook.

Use it as a systems walkthrough, not only as a checklist. The goal is to understand how each layer connects to the next: startup wiring, routes, handlers, forms, sessions, context, database models, workflow, audit, and production hardening.

Related project docs:

- [Control Plane Notebook Architecture.md](./Control%20Plane%20Notebook%20Architecture.md)
- [Control Plane Notebook Reading Map.md](./Control%20Plane%20Notebook%20Reading%20Map.md)
- [Control Plane Notebook vs Ledger API.md](./Control%20Plane%20Notebook%20vs%20Ledger%20API.md)
- [Phase 0-1 Wiring Recipe.md](./Phase%200-1%20Wiring%20Recipe.md)
- [Phase 2-5 Wiring Recipe.md](./Phase%202-5%20Wiring%20Recipe.md)

## Overall System Goal

You are building a progressively layered Go backend:

```text
Phase 0 -> Server skeleton and shared dependencies
Phase 1 -> First server-rendered page
Phase 2 -> Forms, validation, sessions, context, and auth
Phase 3 -> Notebook domain, tags, search, and DB-backed pages
Phase 4 -> Approvals, audit, moderation, and governance
Phase 5 -> Production hardening, tests, and operational readiness
```

Each phase composes over the previous phase. You are not replacing the earlier work. You are adding one clearer layer at a time.

## Global Architecture Mental Model

```text
main.go
  -> builds long-lived dependencies
  -> creates Application
  -> calls app.routes()

routes.go
  -> declares route groups
  -> attaches middleware
  -> points routes at handler methods

middleware.go
  -> enriches or protects each request
  -> loads session data
  -> attaches current user to context

handlers
  -> parse input
  -> call store/domain logic
  -> prepare template data
  -> render or redirect

internal/data
  -> owns SQL, sqlc queries, and transactions

templates
  -> render prepared view data only
```

The core architecture rule:

```text
main.go builds the application once, app.routes() builds the router once, and each request then flows through middleware into handler methods on app.
```

If you keep that rule, the project stays understandable.

## The Two Kinds of State

Backend confusion often starts when long-lived state and request-scoped state get mixed together.

### Long-Lived Application Dependencies

These are created once at startup and shared by all requests:

- config
- logger
- PostgreSQL connection pool
- sqlc/store layer
- session manager
- template cache
- mailer or background worker later

These belong on the `Application` struct.

```go
type Application struct {
    config         Config
    logger         *slog.Logger
    conn           *pgxpool.Pool
    store          data.UserStore
    sessionManager *scs.SessionManager
    templateCache  map[string]*template.Template
}
```

### Request-Scoped Data

These exist only for one request:

- route params
- form values
- flash messages
- CSRF token
- current user
- current team
- request deadline or cancellation signal

These belong in `*http.Request`, its form fields, its session values, or its `Context`.

Mental checkpoint:

```text
Application struct = shared backpack
Request context = this one request's timeline and metadata
Session = cross-request user state stored behind a cookie
```

## Book References

Use the reading map as the main sequence.

For this recipe, the most important `Let's Go` ideas are:

- project structure and organization
- HTML templating and inheritance
- dependency injection
- centralized error handling
- isolated routes
- database connection pools
- models and SQL queries
- template caches and common dynamic data
- middleware chains
- forms and validation
- sessions, cookies, login, logout, authorization, CSRF
- request context
- handler and form tests

The practical learning order is:

```text
Project structure -> routes -> templates -> DB pool -> models -> forms -> sessions -> context -> auth -> workflow -> tests
```

## Phase 0 - Foundation and Wiring

### Goal

Build the server shell. No serious product behavior yet.

Phase 0 answers:

- How does the app start?
- Where are shared dependencies created?
- How does the router receive those dependencies?
- Where do templates, sessions, and the DB pool live?

### Files

```text
cmd/web/main.go
cmd/web/routes.go
cmd/web/helpers.go
ui/templates/
internal/data/
```

### `cmd/web/main.go`

Purpose:

`main.go` is the entry point and dependency setup file.

What belongs here:

- load config
- create logger
- open `pgxpool.Pool`
- ping database
- create `scs.SessionManager`
- configure session cookies
- build template cache
- create `Application`
- create `http.Server`
- call `app.routes()`
- handle graceful shutdown

What does not belong here:

- page-specific logic
- raw SQL
- form validation
- auth decisions
- template rendering logic

Mental model:

```text
main.go is the control room.
It turns on the building, wires the shared services, then hands traffic to the router.
```

Startup flow:

```text
main.go
  -> load config
  -> open PostgreSQL pool
  -> ping database
  -> create session manager
  -> build template cache
  -> create Application
  -> call app.routes()
  -> start http.Server
```

Example:

```go
app := &Application{
    config:         cfg,
    conn:           pool,
    store:          &data.PgxStore{DB: pool},
    sessionManager: sessionManager,
    templateCache:  templateCache,
}

server := http.Server{
    Addr:         ":" + strconv.Itoa(cfg.Port),
    Handler:      app.routes(),
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
}
```

Inputs:

- environment variables
- config values
- filesystem templates
- database URL

Outputs:

- a running HTTP server
- one shared `Application` object
- one shared router

### `Application` Struct

Purpose:

`Application` is dependency injection in plain Go.

Why it exists:

- avoids globals
- keeps handlers testable
- makes shared dependencies explicit
- gives handler methods access to the same DB pool, session manager, template cache, and logger

Handler methods then look like this:

```go
func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
    data := app.newTemplateData(r)
    app.render(w, r, http.StatusOK, "home.page.tmpl", data)
}
```

Mental checkpoint:

```text
If a dependency is created once and reused by many requests, it probably belongs on Application.
If data belongs to one visitor's current request, it does not.
```

### `newTemplateCache`

Purpose:

Parse templates once at startup, then reuse them.

Why:

- avoids parsing templates on every request
- catches template errors early
- keeps rendering consistent
- uses `html/template` for contextual HTML escaping

Example shape:

```go
func newTemplateCache(dir string) (map[string]*template.Template, error) {
    cache := map[string]*template.Template{}
    pages, err := filepath.Glob(filepath.Join(dir, "*.page.tmpl"))
    if err != nil {
        return nil, err
    }

    for _, page := range pages {
        name := filepath.Base(page)
        ts, err := template.ParseFiles(page)
        if err != nil {
            return nil, err
        }
        cache[name] = ts
    }

    return cache, nil
}
```

Input:

- template directory path

Output:

- `map[string]*template.Template`
- `error`

### `cmd/web/routes.go`

Purpose:

Declare the HTTP surface area.

Why:

- keeps `main.go` small
- centralizes route grouping
- makes middleware order visible
- makes auth boundaries easier to reason about

Example:

```go
func (app *Application) routes() http.Handler {
    r := chi.NewRouter()

    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(app.sessionManager.LoadAndSave)

    r.Get("/", app.Home)

    return r
}
```

Key idea:

```text
Handler: app.routes()
```

not:

```text
Handler: routes(conn)
```

That one shift lets routes bind to handler methods that can use `app`.

### `cmd/web/helpers.go`

Purpose:

Reusable HTTP helpers.

Early helpers:

- `app.render`
- server error response helper
- not found helper
- bad request helper
- form parsing helpers later

Example:

```go
func (app *Application) render(w http.ResponseWriter, r *http.Request, status int, page string, data *templateData) {
    t, ok := app.templateCache[page]
    if !ok {
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }

    var buf bytes.Buffer
    if err := t.Execute(&buf, data); err != nil {
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(status)
    buf.WriteTo(w)
}
```

Why buffer first:

If template execution fails, you can still send a clean `500`. If you write directly to `w` and fail halfway through, part of the broken page may already be sent.

### Phase 0 Clarity Check

You understand Phase 0 when you can answer:

- What does `main.go` create once?
- Why does `app.routes()` need to be a method?
- Why does `Application` hold the session manager but not the current user?
- Why are templates parsed at startup?
- Why is the DB pool shared?

## Phase 1 - First Request End-to-End

### Goal

Prove the wiring works with one real page.

The first page should:

- receive an HTTP request
- pass through middleware
- prepare template data
- render HTML
- optionally read a flash message from the session

### Request Flow

```text
Browser
  -> GET /
  -> chi router
  -> common middleware
  -> scs LoadAndSave
  -> app.Home
  -> app.newTemplateData
  -> app.render
  -> HTML response
```

### `templateData`

Purpose:

`templateData` is the page view model.

It lets handlers prepare exactly what templates need.

Early version:

```go
type templateData struct {
    Flash string
}
```

Later version:

```go
type templateData struct {
    Flash           string
    User            *data.User
    IsAuthenticated bool
    Form            any
    Errors          map[string]string
    CSRFToken       string
}
```

Rule:

```text
Templates should render prepared data.
They should not contain business logic, SQL logic, or authorization decisions.
```

### `app.Home`

Purpose:

The first real handler.

Example:

```go
func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
    data := app.newTemplateData(r)
    data.Flash = app.sessionManager.PopString(r.Context(), "flash")
    app.render(w, r, http.StatusOK, "home.page.tmpl", data)
}
```

Inputs:

- `http.ResponseWriter`
- `*http.Request`
- session data via `r.Context()`

Outputs:

- HTML response

Why this exists:

- proves routing works
- proves middleware runs
- proves session loading works
- proves template rendering works

### Sessions in Phase 1

The session manager is shared:

```go
app.sessionManager
```

The session data is request-scoped:

```go
app.sessionManager.GetString(r.Context(), "flash")
app.sessionManager.Put(r.Context(), "flash", "Saved successfully")
```

`scs.LoadAndSave` is what connects those ideas.

```text
Incoming request:
  LoadAndSave reads cookie/session state and attaches it to request context.

Handler:
  reads or writes session values using r.Context().

Outgoing response:
  LoadAndSave commits changes back to the store/cookie.
```

Mental checkpoint:

```text
Session manager = the machine
Session values = this visitor's cross-request state
Context = the access path during this one request
```

### Phase 1 Clarity Check

You understand Phase 1 when you can answer:

- Why must `LoadAndSave` be in the middleware chain?
- Why does the handler receive `w` and `r`?
- Why does `newTemplateData(r)` accept the request?
- Why is `Flash` usually popped instead of just read?
- What happens if the template name is missing from the cache?

## Phase 2 - Forms, Sessions, Context, and Auth

### Goal

Make the app know who is using it and what they are allowed to do.

Phase 2 adds:

- forms
- validation
- Post/Redirect/Get
- login
- logout
- sessions
- current-user middleware
- request context helpers
- protected route groups
- CSRF protection

### Concept: Context

`r.Context()` is request-scoped.

It carries:

- cancellation signal
- deadline
- request-scoped values

Mental model:

```text
Every request has its own timeline.
Context carries that timeline plus request-only metadata.
```

Use context for current request data, such as the loaded user:

```go
ctx := contextSetUser(r.Context(), user)
next.ServeHTTP(w, r.WithContext(ctx))
```

Then later:

```go
user := contextGetUser(r.Context())
```

Important:

```text
Context is not a database.
Context is not a session.
Context is not a place for long-lived dependencies.
```

### `cmd/web/context.go`

Purpose:

Typed helpers for request-scoped values.

Recommended shape:

```go
type contextKey string

const userContextKey contextKey = "user"

func contextSetUser(ctx context.Context, user *data.User) context.Context {
    return context.WithValue(ctx, userContextKey, user)
}

func contextGetUser(ctx context.Context) *data.User {
    user, ok := ctx.Value(userContextKey).(*data.User)
    if !ok {
        return nil
    }
    return user
}
```

Inputs:

- `context.Context`
- `*data.User`

Outputs:

- new `context.Context`
- or loaded `*data.User`

Why:

- avoids changing every handler signature
- lets middleware load the user once
- lets handlers and template-data helpers read the current user consistently

### Sessions vs Context

| Concept | Scope | Storage | Example |
| --- | --- | --- | --- |
| Context | one request | memory | current loaded user pointer |
| Session | many requests | DB/cookie-backed | `userID`, flash message |
| Application | whole server lifetime | process memory | DB pool, session manager |

Example login flow:

```text
1. User submits email/password.
2. Handler validates credentials.
3. Handler renews session token.
4. Handler stores userID in session.
5. Browser receives session cookie.
6. Next request sends cookie.
7. LoadAndSave loads session.
8. authenticate middleware reads userID, loads user, attaches user to context.
```

### `cmd/web/middleware.go`

Purpose:

Middleware enriches or protects requests.

`authenticate` should not block anonymous users. It only loads the current user if possible.

```go
func (app *Application) authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := app.sessionManager.GetInt(r.Context(), "userID")
        if id == 0 {
            next.ServeHTTP(w, r)
            return
        }

        user, err := app.store.GetUser(r.Context(), id)
        if err != nil {
            next.ServeHTTP(w, r)
            return
        }

        ctx := contextSetUser(r.Context(), &user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

`requireAuthentication` should protect private routes.

```go
func (app *Application) requireAuthentication(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if contextGetUser(r.Context()) == nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

Middleware order matters:

```text
RequestID
  -> RealIP
  -> Logger
  -> Recoverer
  -> LoadAndSave
  -> authenticate
  -> route-specific guards
```

Why `LoadAndSave` comes before `authenticate`:

`authenticate` needs to read `userID` from the session. The session must be loaded first.

### `cmd/web/auth.go`

Purpose:

Own login and logout page handlers.

Handlers:

- `loginForm`
- `loginSubmit`
- `logout`

Example:

```go
func (app *Application) loginSubmit(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    email := r.PostForm.Get("email")
    password := r.PostForm.Get("password")

    user, err := app.store.GetUserByEmail(r.Context(), email)
    if err != nil {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    if !app.store.CheckPassword(user, password) {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    _ = app.sessionManager.RenewToken(r.Context())
    app.sessionManager.Put(r.Context(), "userID", user.ID)

    http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
```

Why renew session token after login:

It reduces session fixation risk. The visitor gets a fresh session identity after authentication.

### Forms and Post/Redirect/Get

Every mutating form should use PRG:

```text
GET form page
  -> user submits POST
  -> handler parses form
  -> handler validates
  -> handler writes to DB
  -> handler sets flash
  -> handler redirects to GET page
```

Why:

- refresh does not repeat writes
- back button behaves better
- success pages have clean URLs
- flash messages are easy to show once

Validation belongs near the input boundary:

```go
type notebookForm struct {
    Title  string
    Body   string
    Errors map[string]string
}
```

Then:

```go
if form.Title == "" {
    form.Errors["title"] = "Title is required"
}
```

### Route Groups

Purpose:

Keep public, protected, reviewer, and admin routes separate.

Example:

```go
r.Get("/", app.Home)
r.Get("/login", app.loginForm)
r.Post("/login", app.loginSubmit)

r.Group(func(r chi.Router) {
    r.Use(app.requireAuthentication)
    r.Get("/dashboard", app.dashboard)
    r.Post("/logout", app.logout)
    r.Get("/notebooks/new", app.notebookCreateForm)
    r.Post("/notebooks/new", app.notebookCreateSubmit)
})

r.Route("/admin", func(r chi.Router) {
    r.Use(app.requireAuthentication)
    r.Use(app.requireRole("admin"))
    r.Get("/audit", app.adminAudit)
})
```

Mental checkpoint:

```text
authenticate asks: who is this request from, if anyone?
requireAuthentication asks: is anonymous access forbidden here?
requireRole asks: does this user have the needed authority?
```

### Phase 2 Clarity Check

You understand Phase 2 when you can answer:

- Why is `userID` stored in the session instead of the full user?
- Why is the full user loaded into context on each request?
- Why does `authenticate` allow anonymous users through?
- Why does `requireAuthentication` redirect?
- Why should login call `RenewToken`?
- Why do form POST handlers redirect after successful writes?
- Why is CSRF required for an internal browser app?

## Phase 3 - Notebook Core, DB Models, Tags, and Search

### Goal

Build the main product: a team-owned internal notebook/runbook system.

Phase 3 adds:

- users and teams
- team memberships
- documents
- document revisions
- tags
- search
- notebook list/detail/create/edit pages
- visibility-aware loading

### Domain Model

Use one running story:

```text
Alice belongs to the platform team.
Alice creates a draft runbook called "Rotate API keys".
She tags it security and marks it team-visible.
Bob reviews and approves it later.
The published document becomes visible to the platform team.
Every action is auditable.
```

### `internal/data`

Purpose:

Own persistence.

What belongs here:

- SQL queries
- `sqlc` generated query methods
- store interfaces
- transaction helpers
- data models

What does not belong here:

- template rendering
- redirects
- HTTP status codes
- route params

Recommended shape:

```text
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
```

### DB Backbone

Core tables:

- `users`
- `teams`
- `team_memberships`
- `documents`
- `document_revisions`
- `tags`
- `document_tags`
- `approvals`
- `audit_events`
- `moderation_flags`
- `sessions`

### Why Split `documents` and `document_revisions`

`documents` is the stable identity.

It answers:

- which team owns this?
- what is the slug?
- what visibility does it have?
- what is the current published revision?

`document_revisions` is editable content history.

It answers:

- what title/body did this version have?
- who authored it?
- is it draft, submitted, approved, or rejected?
- when was it created?

Without this split, drafts and approvals become fragile because you keep overwriting the same row.

Mental checkpoint:

```text
documents = stable page identity
document_revisions = content snapshots and workflow state
```

### `cmd/web/notebooks.go`

Purpose:

Own notebook page handlers.

Handlers:

- list notebooks
- view notebook
- show create draft form
- submit create draft form
- show edit draft form
- submit edit draft form
- search notebooks
- filter by tag

What belongs here:

- parse route params
- parse forms
- call store methods
- check visibility at the handler/service boundary
- prepare view models
- render or redirect

What does not belong here:

- raw SQL
- password checks
- approval queue logic
- admin audit UI

Example flow:

```go
func (app *Application) notebookCreateSubmit(w http.ResponseWriter, r *http.Request) {
    user := contextGetUser(r.Context())

    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    form := notebookForm{
        Title:  r.PostForm.Get("title"),
        Body:   r.PostForm.Get("body"),
        Errors: map[string]string{},
    }

    if form.Title == "" {
        form.Errors["title"] = "Title is required"
    }

    if len(form.Errors) > 0 {
        data := app.newTemplateData(r)
        data.Form = form
        app.render(w, r, http.StatusUnprocessableEntity, "notebook-create.page.tmpl", data)
        return
    }

    id, err := app.store.CreateDraft(r.Context(), data.CreateDraftParams{
        AuthorID: user.ID,
        Title:    form.Title,
        Body:     form.Body,
    })
    if err != nil {
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }

    app.sessionManager.Put(r.Context(), "flash", "Draft created")
    http.Redirect(w, r, fmt.Sprintf("/notebooks/%d", id), http.StatusSeeOther)
}
```

### Tags and Search

Tags:

- normalize tag names
- use a join table
- avoid comma-separated tags in one column

Search:

- start simple
- learn PostgreSQL full-text search
- add indexes when query plans justify them

Useful indexes:

- unique published slug
- `(team_id, visibility_scope, status)`
- GIN index for full-text search
- partial index for reviewable revisions

### Phase 3 Clarity Check

You understand Phase 3 when you can answer:

- Why does the handler call the store instead of writing SQL?
- Why do drafts belong in `document_revisions`?
- Why does a page need a view model instead of raw DB rows?
- Why are tags normalized?
- Why does search need indexes eventually?
- Where should visibility checks happen?

## Phase 4 - Approvals, Audit, Moderation, and Governance

### Goal

Make the app feel like a serious internal control plane instead of a CRUD notebook.

Phase 4 adds:

- submit for review
- approval queue
- approve/reject transitions
- audit events
- moderation flags
- admin pages
- role/team authorization
- transactional workflow writes

### Workflow Model

Example states:

```text
draft
  -> submitted
  -> approved
  -> published

submitted
  -> rejected
```

The exact states can change, but the rule is stable:

```text
Workflow transitions should be explicit and auditable.
```

### `internal/data/queries/approvals.sql`

Purpose:

Own approval queue and workflow transition queries.

Examples:

- list submitted revisions for reviewer
- get submitted revision by ID
- approve revision
- reject revision
- mark document's published revision

### `internal/data/queries/audit.sql`

Purpose:

Own append-only audit history.

Events worth storing:

- login
- logout
- document create
- draft edit
- revision submit
- approval
- rejection
- publish
- visibility change
- moderation hide
- moderation restore
- role change

Mental model:

```text
Audit is the black box recorder.
If a user changes authority, visibility, or published content, the system should remember it.
```

### `internal/data/queries/moderation.sql`

Purpose:

Own moderation queue behavior.

Moderation should hide or flag content without destroying history.

Example actions:

- flag document
- hide document
- restore document
- resolve moderation flag

### Transactions

Approval and audit should usually happen in one transaction.

Bad shape:

```text
approve revision
then insert audit event separately
```

If the second write fails, the system approved something without audit history.

Better shape:

```text
begin transaction
  approve revision
  update document published pointer
  insert audit event
commit
```

Example:

```go
func (s *Store) ApproveRevision(ctx context.Context, params ApproveRevisionParams) error {
    return s.WithTx(ctx, func(q *Queries) error {
        if err := q.ApproveRevision(ctx, params.RevisionID); err != nil {
            return err
        }

        if err := q.PublishRevision(ctx, params.RevisionID); err != nil {
            return err
        }

        return q.InsertAuditEvent(ctx, InsertAuditEventParams{
            ActorID: params.ActorID,
            Action:  "revision.approved",
        })
    })
}
```

### `cmd/web/approvals.go`

Purpose:

Own reviewer-facing pages.

Handlers:

- approval queue
- submitted revision detail
- approve POST
- reject POST

What belongs here:

- reviewer route protection
- form parsing for rejection notes
- calling transactional store methods
- flash messages and redirects

### `cmd/web/admin.go`

Purpose:

Own operator/governance pages.

Handlers:

- audit log list
- moderation queue
- user role management
- team membership management

What belongs here:

- admin-only route guards
- audit filtering
- moderation actions
- role/team pages

What does not belong here:

- notebook create/edit flow
- login/logout
- raw SQL

### Authorization

Authentication asks:

```text
Who is this?
```

Authorization asks:

```text
What is this person allowed to do?
```

Examples:

- author can edit own draft
- team member can read team-visible published document
- reviewer can approve submitted revisions
- moderator can hide problematic documents
- admin can change roles

Authorization helpers belong in a focused package such as:

```text
internal/auth/permissions.go
```

Templates may hide buttons for usability, but handlers must enforce the rule.

Mental checkpoint:

```text
Hiding a button is not security.
The POST handler must still check permission.
```

### Phase 4 Clarity Check

You understand Phase 4 when you can answer:

- Why are workflow transitions explicit?
- Why should approval and audit writes share a transaction?
- Why is audit append-only?
- Why should moderation hide content instead of deleting it?
- Why do templates not own authorization rules?
- What is the difference between reviewer, moderator, and admin?

## Phase 5 - Production Patterns, Concurrency, Testing, and Hardening

### Goal

Make the app maintainable, observable, and safe to evolve.

Phase 5 adds:

- structured logging
- timeouts
- graceful shutdown
- background tasks
- handler tests
- form tests
- data-layer tests
- integration tests
- SQL review
- security checks

### HTTP Server Hardening

Use explicit server settings:

```go
server := http.Server{
    Addr:           ":" + strconv.Itoa(cfg.Port),
    Handler:        app.routes(),
    ReadTimeout:    10 * time.Second,
    WriteTimeout:   10 * time.Second,
    MaxHeaderBytes: 1 << 20,
}
```

Why:

- avoids slow-client resource exhaustion
- makes production behavior explicit
- gives shutdown a controlled path

### Graceful Shutdown

Purpose:

Stop accepting new requests, but let in-flight requests finish within a timeout.

Flow:

```text
start server in goroutine
wait for interrupt signal
create shutdown context with timeout
call server.Shutdown(ctx)
close DB pool
log shutdown complete
```

Mental checkpoint:

```text
Startup and shutdown are both part of production behavior.
```

### Background Tasks

Purpose:

Run non-response-critical work without blocking the HTTP response.

Examples:

- send email
- write slow notification
- process import
- rebuild search index

Basic helper:

```go
func (app *Application) background(fn func()) {
    go func() {
        defer func() {
            if err := recover(); err != nil {
                app.logger.Error("background panic", "error", err)
            }
        }()

        fn()
    }()
}
```

Use:

```go
app.background(func() {
    app.mailer.Send(...)
})
```

Important warning:

```text
Starting goroutines without knowing when they stop can leak work.
```

Better production patterns:

- pass `context.Context`
- use worker pools for repeated jobs
- track background tasks with `sync.WaitGroup`
- shut workers down during server shutdown
- log panics from background goroutines

### Context and Concurrency

Context matters in concurrent code because it carries cancellation.

Example:

```go
ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
defer cancel()

result, err := app.store.SearchDocuments(ctx, query)
```

Why:

- request canceled means DB work can stop
- timeout prevents runaway work
- shutdown can cancel pending tasks

### Structured Logging

Good logs should answer:

- who made the request?
- what route was hit?
- how long did it take?
- did it succeed?
- which document/revision was affected?
- what actor performed an approval or moderation action?

Avoid logs that only say:

```text
error happened
```

Prefer logs with fields:

```go
app.logger.Info("revision approved",
    "actor_id", user.ID,
    "revision_id", revisionID,
    "document_id", documentID,
)
```

### Tests

Good early test targets:

- login success
- login invalid credentials
- logout destroys session
- anonymous user redirected from protected page
- authenticated user can access protected page
- notebook create form validation
- notebook create success uses PRG
- approval inserts audit event in same transaction
- unauthorized user cannot approve

Handler test mental model:

```text
Build request
  -> run through app routes or handler
  -> inspect response code, headers, body, session behavior
```

Data-layer test mental model:

```text
Prepare database state
  -> call store method
  -> verify rows and constraints
```

Integration test mental model:

```text
Exercise the real stack where the behavior depends on DB + sessions + middleware.
```

### SQL Maturity

Phase 5 is where SQL stops being "just queries" and becomes part of system design.

Practice:

- named constraints
- foreign keys
- unique constraints
- partial indexes
- GIN indexes for search
- transactions
- reading `EXPLAIN`
- checking slow queries

Common mistake:

```text
Building search as permanent %LIKE% scans and never learning query plans.
```

### Phase 5 Clarity Check

You understand Phase 5 when you can answer:

- What happens to in-flight requests during shutdown?
- Which work can safely run in the background?
- How does a background goroutine stop?
- Which behaviors need handler tests?
- Which behaviors need data-layer tests?
- What should be transactional?
- What would you look for in `EXPLAIN`?

## Full Request Lifecycle

When the system is healthy, a serious write flow looks like this:

```text
1. Alice requests the edit page.
2. Router receives the request.
3. Session middleware loads session state.
4. authenticate middleware reads userID from session.
5. authenticate loads Alice from the DB.
6. authenticate attaches Alice to r.Context().
7. requireAuthentication allows the request through.
8. Handler loads the draft.
9. Handler checks Alice may edit the draft.
10. Handler renders the edit form.
11. Alice submits the form.
12. POST handler parses form data.
13. POST handler validates input.
14. Handler calls a store method.
15. Store writes revision data in the DB.
16. Store inserts an audit event if required.
17. Handler stores a flash message.
18. Handler redirects to the resulting GET page.
19. GET page renders fresh state from the DB.
```

Short version:

```text
HTTP request
  -> middleware
  -> context/session
  -> handler
  -> store/domain
  -> database
  -> template data
  -> render or redirect
```

## File Purpose Map

```text
cmd/web/main.go
  Startup, config, DB pool, sessions, templates, server.

cmd/web/routes.go
  Route table, middleware order, route groups.

cmd/web/middleware.go
  Request enrichment and access gates.

cmd/web/context.go
  Typed request-context helpers.

cmd/web/helpers.go
  Shared HTTP helpers such as render and error helpers.

cmd/web/auth.go
  Login, logout, signup if needed.

cmd/web/notebooks.go
  Notebook list, detail, create, edit, search, tags.

cmd/web/approvals.go
  Reviewer queue, approve, reject.

cmd/web/admin.go
  Audit, moderation, roles, team management.

internal/data/
  SQL, sqlc, store methods, transactions.

internal/auth/
  Permission and role helpers.

internal/validator/
  Form validation helpers.

ui/templates/
  Layouts, partials, pages.

migrations/
  Schema history.
```

## Decision Table: Where Should Logic Go?

| Logic type | Location |
| --- | --- |
| server startup | `cmd/web/main.go` |
| route declaration | `cmd/web/routes.go` |
| middleware order | `cmd/web/routes.go` |
| current-user loading | `cmd/web/middleware.go` |
| context helpers | `cmd/web/context.go` |
| page rendering | `cmd/web/helpers.go` |
| form parsing | handler |
| form validation | handler plus `internal/validator` |
| SQL | `internal/data/queries/*.sql` |
| transactions | `internal/data/tx.go` or store methods |
| permission rules | `internal/auth/permissions.go` |
| templates | `ui/templates` |
| workflow transitions | store/domain layer |
| audit writes | data/store transaction methods |
| background jobs | app-level helper or worker package |

## Common Mistakes to Avoid

- Treating Control Plane Notebook like a JSON API with HTML pasted on top.
- Creating a session manager in `routes.go` instead of once in `main.go`.
- Forgetting `r.Use(app.sessionManager.LoadAndSave)`.
- Loading the current user in every handler instead of middleware.
- Putting current user on `Application`.
- Using plain string context keys everywhere.
- Skipping `RenewToken` after login.
- Skipping CSRF because the app is internal.
- Rendering templates directly from many handlers without a central helper.
- Passing raw database rows into templates forever.
- Mixing authorization rules into templates only.
- Using one mutable `documents` row for drafts, approvals, and published content.
- Writing approval and audit records outside a transaction.
- Hand-editing generated `sqlc` files.
- Starting goroutines with no shutdown path.
- Adding indexes blindly without reading query plans.

## Final Build Order

Build in this order:

```text
1. Phase 0: app wiring, config, pgxpool, sessions, template cache, app.routes()
2. Phase 1: home page, render helper, templateData, flash message
3. Phase 2: forms, validation, login, logout, context, current-user middleware, protected routes, CSRF
4. Phase 3: users, teams, memberships, notebooks, revisions, tags, search
5. Phase 4: approvals, audit events, moderation, admin pages, permission helpers
6. Phase 5: logging, timeouts, graceful shutdown, background tasks, tests, SQL review
```

By the end of Phase 5, you should have a real server-rendered Go backend foundation:

- layered architecture
- dependency injection through `Application`
- database-backed pages
- session-backed authentication
- request-context current user
- protected route groups
- forms and PRG
- draft/revision workflow
- approval and audit transactions
- moderation and admin surfaces
- production-ready server behavior
- tests around the flows that matter

## Final Mental Model

When confused, return to this:

```text
Application is the shared machine.
Routes are the map.
Middleware prepares the request.
Context carries request-local facts.
Session carries cross-request user state.
Handlers coordinate one HTTP interaction.
Store owns persistence.
Templates display prepared data.
Transactions protect multi-step state changes.
Tests lock in the behavior.
```

