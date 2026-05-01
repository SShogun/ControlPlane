# Control Plane Notebook Optimization Notes

## Section 1: Project Overview (for a beginner)

Control Plane Notebook is a server-rendered Go web application for internal operational knowledge: runbooks, incident notes, notebook drafts, reviews, approvals, and audit history. It solves the problem of keeping important team knowledge governed instead of scattered across chat, loose documents, or unaudited notes.

It is built as a server-rendered Go app because the browser mostly needs complete pages, forms, redirects, and cookies. A JSON API would force a separate frontend to own page state, form errors, authentication transitions, and rendering, which would add another moving part before the backend fundamentals are clear.

The app uses PostgreSQL for durable data, `pgxpool` for database connections, `scs` for sessions, `chi` for routing, and `html/template` for safe HTML output. The code is intentionally explicit so a beginner can see where each dependency is created, where each request enters the router, and where handlers call the data layer.

The one-sentence architecture rule is: `main.go` builds the app once, `app.routes()` builds the router once, and each request flows through middleware into handler methods on `app`.

## Section 2: Domain Map — What Each File/Folder Does

### cmd/web/main.go

`cmd/web/main.go` owns application startup. It reads configuration, creates the PostgreSQL pool, checks database connectivity, creates the one shared session manager, builds the template cache, creates the `Application` struct, and starts the `http.Server`. Per-request data does not belong here; this file should not read route params, form values, or the current user. A concrete example is `sessionManager := scs.New()`, followed by `sessionManager.Store = pgxstore.New(pool)`.

### cmd/web/routes.go

`cmd/web/routes.go` owns the URL map and middleware graph. It decides which middleware wraps requests and which handler method responds to each path. Database queries, form parsing, and template execution do not belong here. A concrete example is `r.Get("/notebooks", app.listNotebooks)`.

### cmd/web/middleware.go

`cmd/web/middleware.go` owns request pipeline helpers that run before handlers. It contains authentication and role-checking middleware that can attach a user to context or redirect a user who lacks access. Page-specific database work does not belong here. A concrete example is `app.authenticate`, which reads `"userID"` from the session and stores the loaded user in `r.Context()`.

### cmd/web/helpers.go  (or helpers.go + templates.go if split)

`cmd/web/helpers.go` owns shared handler helpers. It defines `templateData`, renders pre-parsed templates, builds common template data, reads route IDs, logs audit events, and sends server errors. Route registration does not belong here, and long business workflows should stay in handlers or the data layer. A concrete example is `app.render(w, r, http.StatusOK, "home.page.tmpl", data)`.

### cmd/web/context.go

`cmd/web/context.go` owns typed access to request context values used by the web layer. It hides the raw context key so handlers call `contextGetUser(r.Context())` instead of repeating `ctx.Value(...)`. Database pools, sessions, and template caches do not belong here. A concrete example is `contextSetUser(ctx, &user)`.

### cmd/web/auth.go

`cmd/web/auth.go` owns login, login form rendering, and logout handlers. It parses login forms, checks credentials through the store, renews the session token, stores `"userID"` in the session, and redirects after login or logout. Notebook editing and approval workflows do not belong here. A concrete example is `app.sessionManager.Put(r.Context(), "userID", user.ID)`.

### cmd/web/home.go  (or whichever handler files exist)

`cmd/web/home.go` does not exist yet. The home and dashboard handlers currently live in `cmd/web/routes.go`, where `app.home`, `app.dashboard`, and `app.adminAudit` render their pages. If this file is created later, it should own simple page handlers, not router construction or shared dependency setup. A concrete current example is `app.home`, which renders `home.page.tmpl`.

### cmd/web/notebooks.go

`cmd/web/notebooks.go` owns notebook page handlers. It renders the create, list, view, edit, and search pages; it parses notebook forms; and it calls store methods such as `CreateDraft`, `CreateNotebookRevision`, `ListNotebooks`, `ListNotebookRevisions`, and `ListNotebookTags`. Session setup and router construction do not belong here. A concrete example is `app.notebookCreateSubmit`, which validates the title, creates a draft, creates an initial revision, and redirects to the notebook page.

### cmd/web/approvals.go

`cmd/web/approvals.go` owns reviewer approval handlers. It lists submitted revisions, approves a revision through `ApproveRevisionTx`, rejects a revision through `UpdateRevisionStatus`, writes flash messages, and redirects after POST requests. Login, template cache creation, and raw route setup do not belong here. A concrete example is `app.approveRevisionSubmit`, which calls `app.store.ApproveRevisionTx(...)`.

### cmd/web/admin.go

`cmd/web/admin.go` does not exist yet. The current admin audit handler lives in `cmd/web/routes.go` as `app.adminAudit`. If this file is created later, it should own admin-only page handlers and leave role middleware in `middleware.go`. A concrete current example is the `/admin/audit` route, which renders `admin-audit.page.tmpl`.

### internal/data/store.go

`internal/data/store.go` owns the data-layer interface and plain domain structs used by the web layer. It describes what storage operations the app needs without saying exactly how PostgreSQL executes them. HTTP handlers and session logic do not belong here. A concrete example is the `UserStore` interface method `ListSubmittedRevisions(ctx context.Context) ([]NotebookRevision, error)`.

### internal/data/pgxstore.go

`internal/data/pgxstore.go` owns the concrete PostgreSQL implementation of the store. It runs SQL with `pgxpool`, scans rows into domain structs, checks bcrypt password hashes, and performs the approval transaction. Router setup, template rendering, and request redirects do not belong here. A concrete example is `ApproveRevisionTx`, which updates the revision, updates the notebook, inserts an audit event, and commits the transaction.

### internal/data/tx.go

`internal/data/tx.go` exists but is currently empty except for the package declaration. It is a reasonable place for future transaction helpers if several store methods begin sharing transaction setup. Handler code and route declarations do not belong here. There is no concrete behavior in this file yet.

### internal/data/queries/

`internal/data/queries/` owns the SQL files intended for `sqlc` input. These files describe named queries such as `GetNotebookByID`, `ListPublishedNotebooks`, `ApproveRevision`, and `InsertAuditLog`. Go handler code and runtime state do not belong here. A concrete example is `internal/data/queries/notebooks.sql`, which contains notebook read, update, and search queries.

### internal/data/sqlc/

`internal/data/sqlc/` does not exist in this repo. The generated `sqlc` package currently lives at `internal/data/generated/` instead. If `internal/data/sqlc/` is created later, it should contain generated query code only, not hand-written business logic. The concrete current equivalent is `internal/data/generated/notebooks.sql.go`.

### internal/data/generated/

`internal/data/generated/` owns generated `sqlc` Go files. These files are marked `Code generated by sqlc. DO NOT EDIT.` and include generated models, query methods, and the `Queries` type. Hand-written cleanup does not belong here unless the SQL is regenerated by `sqlc`. A concrete example is `internal/data/generated/approved.sql.go`, which provides generated approval update methods.

### internal/auth/permissions.go  (if it exists)

`internal/auth/permissions.go` does not exist yet. Permission checks currently live in `cmd/web/middleware.go` through `app.requireRole("reviewer")` and `app.requireRole("admin")`. If this file is added later, it should own reusable permission rules, not HTTP redirects or template rendering. A concrete current pattern is the role check in `requireRole`.

### internal/validator/validator.go  (if it exists)

`internal/validator/validator.go` does not exist yet. Validation is currently local and minimal, such as the notebook title check in `app.notebookCreateSubmit`. If this package is added later, it should own reusable validation helpers, not database writes or redirects. A concrete current example is `form.Errors["title"] = "Title is required"`.

### ui/html/

`ui/html/` does not exist yet. The actual template folder is `ui/templates/`, and `main.go` currently calls `newTemplateCache("./ui/templates")`. Template folders should contain HTML template files only, not Go handlers or database code. A concrete current path is `ui/templates/`, although it has no listed template files in this workspace snapshot.

### migrations/

`migrations/` owns database schema changes. It creates tables such as `users`, `teams`, `notebooks`, `notebook_revisions`, `approvals`, `audit_events`, `moderation_flags`, and `audit_logs`. Go request logic and templates do not belong here. A concrete example is `migrations/0001_initial_schema.sql`, which creates the core notebook and approval schema.

### docs/

`docs/` owns learning and architecture notes for the project. It is for explanations, recipes, and reading maps, not runtime code. It should not contain handlers, migrations, or generated query code. A concrete example is `docs/Control Plane Notebook Architecture.md`.

### go.mod

`go.mod` owns the module name, Go version, and dependency requirements. It should change only when the project genuinely adds, removes, or upgrades Go dependencies. Application config and route definitions do not belong here. A concrete example is the dependency on `github.com/go-chi/chi/v5`.

### go.sum

`go.sum` owns dependency checksum records used by Go to verify downloaded modules. Humans normally do not edit it by hand. Business logic, config, and documentation do not belong here. A concrete example is checksum data for transitive packages used by `pgx` and `scs`.

### sqlc.yaml

`sqlc.yaml` owns `sqlc` generation settings. It tells `sqlc` where SQL queries live and where generated Go code should be written. Runtime database access and handlers do not belong here. A concrete current example is the generated package under `internal/data/generated/`.

### README.md

`README.md` owns the project introduction, learning goals, roadmap, and local development notes. It is useful for humans entering the repo, not for code executed by the server. Implementation details that must compile should stay in Go files. A concrete example is the README's architecture direction: `main.go` builds long-lived dependencies and handlers are methods on `app`.

## Section 3: The Request Lifecycle — Step by Step

1. `http.Server` receives the TCP connection. The server created in `main.go` listens on `":" + strconv.Itoa(cfg.Port)` and hands matching HTTP requests to `app.routes()`. This matters because the server is the single entry point into the web app.

2. The `chi` router matches the path. For `GET /notebooks`, `chi` finds the route registered as `r.Get("/notebooks", app.listNotebooks)` inside the authenticated route group. This produces a handler chain for that exact path.

3. The global middleware chain runs in order. `middleware.RequestID` gives the request an ID, `middleware.RealIP` records the client IP, `middleware.Logger` logs the request, and `middleware.Recoverer` catches panics so the server can return an error instead of crashing the process.

4. `app.sessionManager.LoadAndSave` loads session state into context. It reads the session cookie, loads session data through the configured `pgxstore`, and makes values like `"userID"` and `"flash"` available during the request. This matters because handlers should not parse session cookies manually.

5. `app.authenticate` reads the session user ID and attaches the `User` to `r.Context()`. It calls `app.sessionManager.GetInt(r.Context(), "userID")`, loads the user through `app.store.GetUser`, and then calls `contextSetUser`. This matters because later middleware and handlers can ask `contextGetUser(r.Context())` instead of querying the user repeatedly.

6. The authenticated group middleware runs. `app.requireAuthentication` checks whether a current user exists in context and redirects to `/login` if not. This matters because `/notebooks` should be available only to signed-in users.

7. The handler `app.listNotebooks` runs. It owns the page-specific behavior for `GET /notebooks`. This matters because the router and middleware have finished deciding access, so the handler can focus on loading notebook data and rendering the response.

8. The handler calls `app.newTemplateData(r)`. This builds common template data such as the flash message, current user, and `IsAuthenticated`. This matters because every page gets consistent data without repeating the same setup in every handler.

9. The handler calls the store query. `app.listNotebooks` calls `app.store.ListNotebooks(r.Context())`, which is implemented by `PgxStore` using SQL against PostgreSQL. This matters because database access stays behind the store interface instead of being mixed into templates or routes.

10. The handler calls `app.render(...)`. It passes the status code, template name, and `templateData` to the shared render helper. This matters because template execution is centralized and errors are handled consistently.

11. `render` looks up the pre-parsed template in `app.templateCache`. The cache is created once during startup by `newTemplateCache("./ui/templates")`. This matters because template parse errors are found early and handlers do not re-parse files on every request.

12. The template executes and the HTML is written to `w`. `render` executes into a buffer first, then writes the status code and buffered HTML to the response. This matters because a template execution error can be caught before a partial response is sent.

13. `LoadAndSave` commits any session changes on the way out. If a handler popped a flash message or wrote session data, `scs` persists those changes through the session store and response cookie. This matters because login, logout, and flash messages depend on session changes surviving the request.

## Section 4: Changes Made — Phase by Phase

### Phase 0 — Server Skeleton and Shared Dependencies

**File:** `cmd/web/main.go`  
**Change:** Simplified session manager creation to `sessionManager := scs.New()`.  
**Why it works:** The session manager is still created exactly once in `main.go`, then stored on `Application`. The shorter declaration removes beginner noise without changing behavior.

**File:** `cmd/web/main.go`  
**Change:** Added a real `slog.Logger` to the `Application` struct during startup.  
**Why it works:** Helpers such as `serverError` and `logAuditEvent` already use `app.logger`. Initializing it once avoids nil logger panics and keeps logging as long-lived application state.

**File:** `cmd/web/main.go`  
**Change:** Checked the error returned by `server.ListenAndServe()`.  
**Why it works:** If the server fails to bind or crashes during startup, the program now logs a fatal error instead of silently ignoring the failure.

**File:** `cmd/web/routes.go`  
**Change:** Kept routing owned by `app.routes()` and left `main.go` calling `app.routes()`.  
**Why it works:** Shared dependencies are created once in `main.go`; routes use the already-built `app` instead of rebuilding dependencies.

**File:** `cmd/web/routes.go`  
**Change:** Added `r.Use(app.authenticate)` after `r.Use(app.sessionManager.LoadAndSave)`.  
**Why it works:** Session data must be loaded before authentication can read `"userID"`. Authentication must run before protected routes can check `contextGetUser`.

### Phase 1 — DB-Backed Pages

**File:** `cmd/web/helpers.go`  
**Change:** Checked and logged the error from `buf.WriteTo(w)`.  
**Why it works:** Response write failures are now visible in logs instead of being silently discarded.

**File:** `cmd/web/helpers.go`  
**Change:** Fixed the audit log error message typo from `"audio"` to `"audit"`.  
**Why it works:** Clear log messages help beginners connect an error to the correct subsystem.

**File:** `cmd/web/routes.go`  
**Change:** Renamed exported `Home` to unexported `home` and updated the route.  
**Why it works:** The handler is internal to the web package, so the lowercase name better matches the other handler methods and avoids suggesting it is a public API.

**File:** `cmd/web/routes.go`  
**Change:** Removed comments that merely described the route block contents.  
**Why it works:** The route names already say what they do, so the comments were visual noise.

### Phase 2 — Identity, Sessions, and Access Control

**File:** `cmd/web/auth.go`  
**Change:** Checked the error from `app.sessionManager.RenewToken`.  
**Why it works:** Token renewal is part of safe login. If it fails, continuing would hide a real session problem.

**File:** `cmd/web/middleware.go`  
**Change:** Removed comments that restated context-setting code.  
**Why it works:** The function names already explain the action, so the comments did not add useful reasoning.

**File:** `internal/data/store.go`  
**Change:** Removed an abandoned `data.Application`, `SessionManager`, and `LoginSubmit` handler from the data package.  
**Why it works:** HTTP handlers belong in `cmd/web`, while `internal/data` owns data types and storage contracts. This fixes a crossed seam and removes dead code.

### Phase 3 — Notebook Core (Forms, Validation, PRG)

**File:** `cmd/web/notebooks.go`  
**Change:** Removed an abandoned route checklist comment block.  
**Why it works:** The actual routes live in `routes.go`, so the comment could drift out of date and confuse a beginner.

**File:** `cmd/web/notebooks.go`  
**Change:** Checked the error from `app.store.ListNotebookTags`.  
**Why it works:** Failing to load tags is a real page-rendering problem. The handler now returns a server error instead of rendering incomplete data silently.

**File:** `cmd/web/notebooks.go`  
**Change:** Kept the PRG pattern after draft creation.  
**Why it works:** `notebookCreateSubmit` creates data, sets a flash message, and redirects with `http.StatusSeeOther`, which prevents accidental form resubmission on refresh.

### Phase 4 — Approvals, Audit, and Moderation

**File:** `cmd/web/approvals.go`  
**Change:** Moved `rejectRevisionSubmit` into the approvals handler file.  
**Why it works:** Rejection is an approval workflow handler, not middleware. Keeping it near `approveRevisionSubmit` and `approvalQueue` makes the code easier to scan.

**File:** `internal/data/store.go`  
**Change:** Added `UpdateRevisionStatus` and `UpdateRevisionStatusParams` to the store contract.  
**Why it works:** The rejection handler already needed this storage operation. Putting it on the store interface keeps handlers dependent on the data seam rather than a concrete database type.

**File:** `internal/data/pgxstore.go`  
**Change:** Implemented `UpdateRevisionStatus` for PostgreSQL.  
**Why it works:** The method updates `notebook_revisions.status` and `updated_at`, matching the approval workflow's need to mark a revision as rejected.

**File:** `internal/data/tx.go`  
**Change:** Removed the placeholder comment from the otherwise empty transaction helper file.  
**Why it works:** Empty placeholder comments do not teach behavior and can be recreated later when the file gains real transaction helpers.

## Section 5: The Two Kinds of State (Core Mental Model)

**Long-lived application state** lives on the `Application` struct. In this app, that means things like the DB pool, session manager, template cache, logger, and config. These are created once in `main.go`, reused by all requests, and safe to share across concurrent goroutines when the underlying library is designed for it.

**Request-scoped state** lives in `*http.Request` and `r.Context()`. In this app, that means the current user, flash message, parsed form values, route params like `{id}`, and future values such as CSRF tokens. These are created for one request and discarded when the response is sent.

Never store request-scoped state on `Application`. If the current user or form values were put on `app`, one user's request could leak into another user's request, or concurrent requests could race with each other. Never recreate long-lived state per request either; rebuilding sessions, pools, or template caches inside handlers can cause stale data, wasted work, and sessions that do not persist correctly.

## Section 6: Key Libraries — What Each One Does

`chi` is the router. It matches HTTP methods and paths like `GET /notebooks` to handler methods like `app.listNotebooks`, and it composes middleware around route groups. Without it, you would manually inspect `r.URL.Path`, check methods, and build your own route grouping behavior.

`pgxpool` is the PostgreSQL connection pool. It keeps a reusable set of database connections so each request does not have to open a fresh TCP connection to PostgreSQL. Without it, you would manually manage connection lifetimes, pooling, timeouts, and concurrency pressure.

`sqlc` generates typed Go code from SQL files. This repo has generated files under `internal/data/generated/`, although the active `PgxStore` also contains hand-written SQL methods. Without `sqlc`, you write all query methods, parameter structs, and scan code by hand.

`scs` plus `scs/pgxstore` handles session management. `scs` gives the app `LoadAndSave`, `Put`, `GetInt`, `PopString`, `Destroy`, and token renewal; `pgxstore` persists sessions in PostgreSQL. Without it, you would need to design signed cookies or server-side session tables, expiration, renewal, and cookie writes yourself.

`html/template` safely renders HTML. It escapes dynamic values by default, which helps prevent template-driven cross-site scripting bugs. Without it, you would concatenate strings or use unsafe rendering helpers and have to remember every escaping rule yourself.

`bcrypt` hashes and verifies passwords. The store uses `bcrypt.CompareHashAndPassword` so plaintext passwords are checked against stored password hashes. Without it, you might accidentally store plaintext passwords or use a fast hash that is unsafe for credentials.

`nosurf` or `gorilla/csrf` is not currently present in `go.mod`. A CSRF library would protect POST routes like login, logout, notebook creation, approval, and rejection from cross-site request forgery. Without one, the app must manually generate, store, render, and verify CSRF tokens for unsafe HTTP methods.

## Section 7: Common Mistakes and How the Code Avoids Them

**Mistake:** Creating the session manager inside `routes()` or inside handlers.  
**How avoided:** `main.go` creates `sessionManager` once, configures its `pgxstore`, and stores it on `Application`. `routes.go` uses `app.sessionManager.LoadAndSave`.

**Mistake:** Protecting routes before loading the session.  
**How avoided:** `routes.go` runs `app.sessionManager.LoadAndSave` before `app.authenticate`, and `app.authenticate` runs before `requireAuthentication`.

**Mistake:** Letting handlers live in the data package.  
**How avoided:** The abandoned `internal/data.Application` login handler was removed. Web behavior now stays in `cmd/web`, and `internal/data` exposes storage methods and structs.

**Mistake:** Ignoring session token renewal errors during login.  
**How avoided:** `loginSubmit` now checks `app.sessionManager.RenewToken` and returns a server error if renewal fails.

**Mistake:** Parsing templates inside the handler on every request.  
**How avoided:** Templates are parsed once at startup into `app.templateCache` by `newTemplateCache`. The render helper looks them up by page name.

**Mistake:** Writing partial HTML before knowing whether the template executes.  
**How avoided:** `app.render` executes the template into a `bytes.Buffer` first, then writes the status code and buffer to the response.

**Mistake:** Silently dropping database errors when rendering a page.  
**How avoided:** `notebookView` now checks the error from `ListNotebookTags` and returns `app.serverError` when tag loading fails.

**Mistake:** Putting approval workflow handlers in middleware.  
**How avoided:** `rejectRevisionSubmit` lives in `cmd/web/approvals.go` beside `approvalQueue` and `approveRevisionSubmit`, while `middleware.go` contains request pipeline code only.
