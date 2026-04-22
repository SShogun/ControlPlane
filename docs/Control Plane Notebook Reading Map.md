# Control Plane Notebook - Reading Map

## Goal

This is the right first large project in the sequence.

Read this before Ledger API. The Control Plane Notebook path should make you comfortable with server-rendered Go applications, forms, sessions, cookies, templates, DB-backed pages, and access control before you move into a headless JSON API.

Use this together with [Control Plane Notebook Architecture.md](./Control%20Plane%20Notebook%20Architecture.md).

## Why This Project Comes Before Ledger

Ledger API teaches transport design for JSON clients.

Control Plane Notebook teaches the full web backend loop:

- render HTML
- parse forms
- validate input
- protect writes with CSRF
- authenticate with sessions and cookies
- authorize by role and team
- fetch data for pages
- redirect after writes

If you skip this, Ledger can make you look more advanced than you really are.

## Read in This Order

## Phase 0 - Build the Server-Rendered Skeleton

Read these `Let's Go` sections:

- 2.3 Routing requests - PDF page 18
- 2.6 Project structure and organization - PDF page 36
- 2.7 HTML templating and inheritance - PDF page 41
- 2.8 Serving static files - PDF page 51
- 2.9 The `http.Handler` interface - PDF page 59
- 3.1 Managing configuration settings - PDF page 63
- 3.2 Leveled logging - PDF page 68
- 3.3 Dependency injection - PDF page 73
- 3.4 Centralized error handling - PDF page 79
- 3.5 Isolating the application routes - PDF page 84

What to extract:

- how to keep `main.go` small
- how to split routes, helpers, and handlers
- how template layout inheritance works
- how static assets fit into a Go web app
- how dependency injection keeps handlers testable

Official online follow-up:

- Go `html/template`: https://pkg.go.dev/html/template
- Go `net/http`: https://pkg.go.dev/net/http
- `chi`: https://pkg.go.dev/github.com/go-chi/chi/v5

## Phase 1 - Get Comfortable with DB-Backed Pages

Read these `Let's Go` sections:

- 4.4 Creating a database connection pool - PDF page 95
- 4.5 Designing a database model - PDF page 99
- 4.6 Executing SQL statements - PDF page 104
- 4.7 Single-record SQL queries - PDF page 109
- 4.8 Multiple-record SQL queries - PDF page 116
- 4.9 Transactions and other details - PDF page 121
- 5.1 Displaying dynamic data - PDF page 128
- 5.2 Template actions and functions - PDF page 136
- 5.3 Caching templates - PDF page 142
- 5.4 Catching runtime errors - PDF page 149
- 5.5 Common dynamic data - PDF page 153
- 5.6 Custom template functions - PDF page 157

What to extract:

- how handlers load data from the database for pages
- how template caches work
- how to create page-specific and shared view data
- how to keep SQL out of handlers

Official online follow-up:

- `pgx`: https://pkg.go.dev/github.com/jackc/pgx/v5
- `pgxpool`: https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool
- `sqlc`: https://docs.sqlc.dev/en/latest/
- PostgreSQL constraints: https://www.postgresql.org/docs/current/ddl-constraints.html
- PostgreSQL index introduction: https://www.postgresql.org/docs/current/indexes-intro.html

## Phase 2 - Forms, Validation, and PRG

Read these `Let's Go` sections:

- 8.1 Setting up a HTML form - PDF page 195
- 8.2 Parsing form data - PDF page 198
- 8.3 Validating form data - PDF page 204
- 8.4 Displaying errors - PDF page 208
- 8.5 Creating validation helpers - PDF page 214
- 8.6 Automatic form parsing - PDF page 219

What to extract:

- how forms differ from JSON bodies
- how to validate and repopulate form fields
- how to show field-level errors
- why Post/Redirect/Get matters

Official online follow-up:

- Go `Request.ParseForm`: https://pkg.go.dev/net/http#Request.ParseForm

## Phase 3 - Sessions, Cookies, Auth, and CSRF

Read these `Let's Go` sections:

- 9.1 Choosing a session manager - PDF page 226
- 9.2 Setting up the session manager - PDF page 227
- 9.3 Working with session data - PDF page 232
- 11.1 Routes setup - PDF page 261
- 11.2 Creating a users model - PDF page 265
- 11.3 User signup and password encryption - PDF page 269
- 11.4 User login - PDF page 284
- 11.5 User logout - PDF page 293
- 11.6 User authorization - PDF page 295
- 11.7 CSRF protection - PDF page 302
- 12.1 How request context works - PDF page 314

What to extract:

- how session-backed auth differs from bearer tokens
- how cookies and session state relate
- how auth and authorization differ
- how CSRF fits into form-based web apps
- how current-user data flows through context

Official online follow-up:

- `scs`: https://pkg.go.dev/github.com/alexedwards/scs/v2
- `scs/pgxstore`: https://pkg.go.dev/github.com/alexedwards/scs/pgxstore
- Go cookie helpers: https://pkg.go.dev/net/http#Cookie
- Go `SetCookie`: https://pkg.go.dev/net/http#SetCookie

## Phase 4 - Teams, Roles, Search, and Governance

Read these `Let's Go` sections again where they matter:

- 4.8 Multiple-record SQL queries - PDF page 116
- 6.1 How middleware works - PDF page 163
- 6.3 Request logging - PDF page 171
- 6.4 Panic recovery - PDF page 173
- 6.5 Composable middleware chains - PDF page 178
- 12.1 How request context works - PDF page 314

What to extract:

- how route groups and middleware map to access levels
- how search and filter pages are still just HTTP handlers
- how current-user and current-team context should be injected once
- how audit and moderation actions should be observable in logs

Official online follow-up:

- PostgreSQL partial indexes: https://www.postgresql.org/docs/current/indexes-partial.html
- PostgreSQL text search introduction: https://www.postgresql.org/docs/current/textsearch-intro.html
- PostgreSQL `EXPLAIN`: https://www.postgresql.org/docs/current/using-explain.html

## Phase 5 - Testing and Hardening

Read these `Let's Go` sections:

- 10.4 Connection timeouts - PDF page 256
- 14.1 Unit testing and sub-tests - PDF page 334
- 14.2 Testing HTTP handlers and middleware - PDF page 342
- 14.5 Mocking dependencies - PDF page 359
- 14.6 Testing HTML forms - PDF page 370
- 14.7 Integration testing - PDF page 377

What to extract:

- how to test handlers without a browser
- how to test form flows and redirects
- how to mock dependencies cleanly
- how to integration-test DB-backed behavior

Official online follow-up:

- Go `http.Server`: https://pkg.go.dev/net/http#Server
- Go `context`: https://pkg.go.dev/context

## Short Version if You Want the Minimum Read Set

If you want the smallest high-value reading path first, read this:

- 2.6 Project structure and organization - page 36
- 2.7 HTML templating and inheritance - page 41
- 3.1 to 3.5 - pages 63 to 84
- 4.4 to 4.9 - pages 95 to 121
- 5.1 to 5.6 - pages 128 to 157
- 6.1 to 6.5 - pages 163 to 178
- 8.1 to 8.6 - pages 195 to 219
- 9.1 to 9.3 - pages 226 to 232
- 11.3 to 11.7 - pages 269 to 302
- 12.1 - page 314
- 14.2 and 14.6 - pages 342 and 370

That is the best first pass through `Let's Go` for this project.

## Best First Week Plan

1. Day 1: read 2.6, 2.7, 3.1 to 3.5 and wire `main.go`, routes, template cache, and a home page.
2. Day 2: read 4.4 to 4.9 and 5.1 to 5.3, then swap in PostgreSQL, `pgxpool`, `sqlc`, and one DB-backed page.
3. Day 3: read 8.1 to 8.6, then build notebook create and edit forms with validation and PRG.
4. Day 4: read 9.1 to 9.3 and 11.3 to 11.7, then add sessions, login, logout, and CSRF.
5. Day 5: read 6.1 to 6.5, 12.1, and 14.2, then add middleware, current-user context, logging, and handler tests.

That will put you in a much better position to approach Ledger afterward.
