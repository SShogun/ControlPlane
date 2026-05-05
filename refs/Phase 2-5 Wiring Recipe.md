# Phase 2-5 Wiring Recipe

This document continues from [Phase 0-1 Wiring Recipe.md](./Phase%200-1%20Wiring%20Recipe.md).

It assumes you already have the following foundation in place:

- `main.go` builds the shared dependencies
- `app.routes()` returns the `chi` router
- `scs.LoadAndSave` is in the middleware chain
- handlers are methods on `app`
- templates are parsed into a cache
# Phase 2-5 Wiring Recipe

This document continues from [Phase 0-1 Wiring Recipe.md](./Phase%200-1%20Wiring%20Recipe.md).

It assumes you already have the foundation in place (a working `main.go`, `app.routes()`, `scs.LoadAndSave` in the middleware chain, handlers as methods on `app`, and a template cache). If that foundation is stable, the rest of the system should feel like adding new seams to the same machine.

**Goal:** make Phase 2 actionable for learning — not just a checklist to copy-paste. The sections below explain the why, the how, and give compact examples + small exercises so you can practice thinking through each piece.

**Core rule for every phase**

- Application: owns long-lived dependencies (db pool, session manager, template cache, logger)
- Middleware: enriches the request (load session, set current user, measure timings)
- Handlers: parse input, call the store/domain layer, prepare view models
- Store/data layer: owns SQL and transactions
- Templates: only render prepared view data

If any feature breaks that separation, stop and fix the seam.

**How to use this recipe (short)**

Follow these steps when adding identity or any feature that touches auth/data:

1. Update the `Application` shape in `main.go` (session, store, templates)
2. Add or adjust middleware/context helpers (these run per request)
3. Add store/sqlc queries and transactional methods
4. Add routes and route groups (public vs protected vs admin)
5. Connect handlers to templates (pass prepared view models)
6. Grow `templateData` for templates' needs
7. Manually test happy and denied/error paths

Phase 2 is the place to build a clean identity seam so later phases remain simple.

**Phase quick map**

Phase 2 — Identity and sessions (this file focuses on this).
Phase 3 — Product features (notebooks, tags, search).
Phase 4 — Workflow and audits (transactions, approvals).
Phase 5 — Hardening and tests.

-----------------------------------------------------------------
**PHASE 2 — Identity, Sessions, Context, and Access Control (practical)**
Purpose: the app must know who is using it and what they can do. The result: anonymous pages work, users can log in/out, session stores user ID, `r.Context()` provides the loaded user, and protected routes are grouped.

Below you will find (A) short conceptual explanations, (B) tiny code examples you can type and study, and (C) short exercises to build muscle.

1) r.Context() — what and why

- What: `r.Context()` is a request-scoped context object. It carries values, deadlines, and cancellation signals tied to this request.
- Why: use it to attach request-only data (like the loaded `User`) so downstream handlers/middleware can access it without changing signatures.

Tiny example (conceptual):

```go
// attach a value
ctx2 := context.WithValue(r.Context(), "user", user)
// pass into next handler
next.ServeHTTP(w, r.WithContext(ctx2))

// retrieve later
u := r.Context().Value("user").(*data.User)
```

Exercise A: open `cmd/web/context.go` and create two helpers: `contextSetUser(ctx, user) context.Context` and `contextGetUser(ctx) *data.User`. Implement with `context.WithValue` and a typed key. (See example below.)

2) The DB pool — simple intuition

- A connection pool (`pgxpool.Pool`) keeps open DB connections and hands them to concurrent requests. This avoids the cost of opening/closing connections per request.
- In `main.go` you build the pool once and store it on `app` so handlers or the store can reuse it.

Exercise B: find where `pgxpool.New` is called (in `main.go`) and locate where `pool` is passed into your `store` or `session` manager.

3) Middleware — purpose and order

- Middleware runs in order for every request. Key middleware for auth:
  - `LoadAndSave` (session load/save) — must run before any middleware that reads session data
  - `authenticate` — reads session userID, loads full user from store, attaches user to `r.Context()`
  - `requireAuthentication` / `requireRole` — enforce protections on route groups

Middleware order (must): RequestID → RealIP → Logger → Recoverer → LoadAndSave → authenticate → route-specific middleware

Exercise C: add `app.authenticate` (non-enforcing) and `app.requireAuthentication` (enforcing) and wire `app.authenticate` into the global middleware stack.

4) Minimal code examples (copy by hand, study each line)

context.go (compact, use a typed key in real code):

```go
package main

import (
  "context"
  "github.com/SShogun/ControlPlane/internal/data"
)

type contextKey string

const userKey contextKey = "user"

func contextSetUser(ctx context.Context, u *data.User) context.Context {
  return context.WithValue(ctx, userKey, u)
}

func contextGetUser(ctx context.Context) *data.User {
  v, ok := ctx.Value(userKey).(*data.User)
  if !ok {
    return nil
  }
  return v
}
```

middleware.go (conceptual):

```go
package main

import (
  "context"
  "net/http"
)

func (app *Application) authenticate(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    id := app.sessionManager.GetInt(r.Context(), "userID")
    if id == 0 {
      next.ServeHTTP(w, r)
      return
    }
    // load user from store (errors handled gracefully)
    u, err := app.store.GetUser(r.Context(), id)
    if err != nil || u == nil {
      next.ServeHTTP(w, r)
      return
    }
    // attach to context and continue
    ctx := contextSetUser(r.Context(), u)
    next.ServeHTTP(w, r.WithContext(ctx))
  })
}

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

auth.go (compact):

```go
package main

import (
  "net/http"
)

func (app *Application) loginForm(w http.ResponseWriter, r *http.Request) {
  data := app.newTemplateData(r)
  app.render(w, r, http.StatusOK, "login.page.tmpl", data)
}

func (app *Application) loginSubmit(w http.ResponseWriter, r *http.Request) {
  if err := r.ParseForm(); err != nil { http.Error(w, "bad request", 400); return }
  email := r.PostForm.Get("email")
  pass := r.PostForm.Get("password")
  u, err := app.store.GetUserByEmail(r.Context(), email)
  if err != nil || u == nil { http.Error(w, "invalid credentials", 401); return }
  if !app.store.CheckPassword(u, pass) { http.Error(w, "invalid credentials", 401); return }
  app.sessionManager.RenewToken(r.Context())
  app.sessionManager.Put(r.Context(), "userID", u.ID)
  http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Application) logout(w http.ResponseWriter, r *http.Request) {
  _ = app.sessionManager.Destroy(r.Context())
  http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

Important: these snippets are learning examples. Type them in, read each line, and run small edits to see how behavior changes. Avoid blind copy-paste; instead, transcribe and ask yourself what each line does.

5) templateData — what to add and why

Expand the struct so templates can make decisions without hitting the DB:

```go
type templateData struct {
  Flash string
  User  *data.User
  IsAuthenticated bool
  Form interface{}
  Errors map[string]string
}

func (app *Application) newTemplateData(r *http.Request) *templateData {
  u := contextGetUser(r.Context())
  return &templateData{User: u, IsAuthenticated: u != nil}
}
```

Exercise D: update `newTemplateData` and a template (e.g., `home.page.tmpl`) to show "Log in" when unauthenticated and "Welcome, {{.User.Name}}" when authenticated.

6) routes — group and protect

Public routes (no auth required): `/`, `/login` (GET/POST)
Protected group (requires auth): `/dashboard`, `/notebooks/new`, `/logout`
Admin group (requires auth + role): `/admin/*`

Example grouping:

```go
r := chi.NewRouter()
r.Use(...common middleware...)
r.Use(app.authenticate) // important: only loads user, does not enforce

r.Get("/", app.Home)
r.Get("/login", app.loginForm)
r.Post("/login", app.loginSubmit)

r.Group(func(r chi.Router) {
  r.Use(app.requireAuthentication)
  r.Get("/dashboard", app.dashboard)
  r.Get("/logout", app.logout)
})

r.Route("/admin", func(r chi.Router) {
  r.Use(app.requireAuthentication)
  r.Use(app.requireRole("admin"))
  r.Get("/users", app.adminUsers)
})
```

Exercise E: Add a `/dashboard` handler that requires authentication. Verify that an anonymous visitor gets redirected to `/login` and that a logged-in user sees the page.

7) Small learning checkpoints (do these sequentially)

- Checkpoint 1: Implement `context.go` helpers and write a tiny test handler that prints whether `contextGetUser` is nil.
- Checkpoint 2: Implement `authenticate` middleware and wire it globally. Add a handler that prints `{{.IsAuthenticated}}` from `newTemplateData`.
- Checkpoint 3: Implement `loginForm` + `loginSubmit` that store `userID` in session; manually create a user row if needed to test.
- Checkpoint 4: Implement `requireAuthentication` and protect `/dashboard`.

8) Common pitfalls and how to reason through them

- "My middleware doesn't see the session" — check ordering: `LoadAndSave` must run before `authenticate`.
- "Context value missing in handler" — verify you're using `r.WithContext(ctx)` when passing to the next handler.
- "Session contains user ID but user load fails" — handle missing/soft-deleted users gracefully (clear session or continue as anonymous).

9) Exercises with hints (practice thinking)

- Exercise 1: Type `context.go` helpers. Then add a temporary handler `/whoami` that writes `"anonymous"` or the user email. Hint: use `contextGetUser` inside the handler.
- Exercise 2: Implement `authenticate` to load user and attach to context. Hint: read `userID` from `app.sessionManager.GetInt(r.Context(), "userID")`.
- Exercise 3: Implement `loginSubmit` that verifies password (you can skip bcrypt at first by testing with a known plaintext password stored in DB) and stores `userID` in session.

If you want, I can convert any of these exercises into a short interactive step (I will implement, run, and test), but the learning value is highest if you type and think through each line yourself first.

-----------------------------------------------------------------
Phase 3, 4 and 5 notes are intentionally shorter here — keep using the wiring pattern above. The rest of the document (previously present) described these phases and remains valid: the critical difference now is that Phase 2 has concrete, typed exercises and clear examples you can transcribe and test.

If you'd like, I can:
- generate a single minimal `cmd/web/context.go`, `cmd/web/middleware.go`, and `cmd/web/auth.go` that you type in by hand (so you learn as you transcribe), or
- open a guided interactive session where I add one file at a time and explain each line as I write it.

Which would you prefer: typed examples for practice, or an interactive build where I produce code you can run and then modify?

