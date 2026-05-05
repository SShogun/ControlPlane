# Phase 0-1 Wiring Recipe

This document is the "recipe" for wiring Phase 0 and Phase 1 of Control Plane Notebook.

The goal is not to dump a lot of code. The goal is to make the data flow feel obvious.

## The Core Mental Model

There are two kinds of state in this app:

### 1. Long-lived application dependencies

These are created once at startup and shared by all requests:

- config
- PostgreSQL pool
- session manager
- template cache
- logger later

These belong on the `application` struct.

### 2. Request-scoped data

These only exist for one request:

- route params
- form values
- the current session values
- the current user
- flash messages
- CSRF token later

These do **not** belong on the `application` struct. They belong in the `*http.Request` and its `Context`.

That distinction is the whole pattern.

## The One-Sentence Architecture Rule

`main.go` builds the application once, `app.routes()` builds the router once, and each request then flows through middleware into handler methods on `app`.

If you keep that rule, the project stays understandable.

## What Is Currently Off in Your Wiring

Your current code is very close to the right shape, but a few seams are crossed:

- `main.go` creates an `application`, but the server uses `routes(conn)` instead of `app.routes()`.
- `routes.go` creates a new session manager locally, so it is not the same shared session manager that lives on `application`.
- the `scs` middleware is configured, but it is never actually attached with `r.Use(app.sessionManager.LoadAndSave)`.
- your handlers are free functions right now, so they cannot naturally reach `app.conn`, `app.sessionManager`, or `app.templateCache`.
- you are using `text/template` in `main.go`; for HTML pages you want `html/template`.

None of this is unusual. It just means the object graph is not fully connected yet.

## Phase 0 - Build the Shared Object Graph

Phase 0 is about one thing: create the shared dependencies once and put them behind `application`.

### Step 1: Build config

`main.go` should be the one place that reads environment variables and flags.

The config should answer questions like:

- what port do we listen on?
- what database URL do we connect to?
- are we in dev or production?
- should session cookies be marked `Secure`?

### Step 2: Open the PostgreSQL pool

Create the `pgxpool.Pool` once in `main.go`.

The pool is a shared, concurrency-safe dependency. It is not request-specific, and it is not something handlers should construct for themselves.

You should also `Ping()` or otherwise validate the pool during startup so the server does not boot with a dead DB connection.

### Step 3: Create the session manager

Create the `scs.SessionManager` once in `main.go`.

Important ideas:

- set the store once, using `pgxstore.New(pool)`
- configure cookie behavior once
- keep the manager on `application`

The session manager is long-lived, but the session **data** it works with is request-scoped and travels through `r.Context()`.

### Step 4: Build the template cache

Parse templates once at startup into a cache.

For a server-rendered app, this should use `html/template`, not `text/template`, because `html/template` provides contextual auto-escaping for HTML output.

The cache should be treated as read-only after startup.

### Step 5: Pack everything into `application`

Your `application` struct is the dependency backpack that every handler method carries implicitly.

For Phase 0 and 1, a good shape is:

```go
type application struct {
    config         Config
    conn           *pgxpool.Pool
    sessionManager *scs.SessionManager
    templateCache  map[string]*template.Template
}
```

This means a handler method like `app.home` can see the DB pool, the session manager, and the template cache without globals.

### Step 6: Make the router a method on `application`

This is the key DI move.

Instead of:

```go
Handler: routes(conn)
```

you want:

```go
Handler: app.routes()
```

And then:

```go
func (app *application) routes() http.Handler
```

That one change is what allows the router to bind handler methods like `app.home`, `app.login`, and `app.notebookList`.

## The Phase 0 Data Flow

Startup should feel like this:

```text
main.go
  -> load config
  -> open pgxpool
  -> create scs session manager
  -> build template cache
  -> create application
  -> call app.routes()
  -> give returned router to http.Server
```

Nothing in this flow is request-specific yet. This is just building the machine.

## Phase 1 - Wire the First Request End-to-End

Phase 1 is where you prove the structure works with one real page.

The best first page is a Home page that:

- reads a session value
- prepares simple template data
- renders a template

## How `chi` Fits the Pattern

`chi` is not a separate framework universe. It is still `net/http`.

That means:

- your handlers are still `func(w http.ResponseWriter, r *http.Request)`
- your middleware is still standard `net/http` middleware
- `scs.LoadAndSave` fits directly into `chi` because it is standard middleware

So the router method should conceptually look like this:

```go
func (app *application) routes() http.Handler {
    r := chi.NewRouter()

    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(app.sessionManager.LoadAndSave)

    r.Get("/", app.home)
    r.Get("/login", app.loginForm)
    r.Post("/login", app.login)

    return r
}
```

The important thing is not the exact middleware list. The important thing is that the router is created by `app`, and the handlers are methods on `app`.

## Where `scs` Sits in the Pipeline

`scs` belongs in the router middleware chain.

Use it like this:

```go
r.Use(app.sessionManager.LoadAndSave)
```

Why this matters:

1. on the way in, `LoadAndSave` loads the session and attaches session state to the request context
2. your handlers and later middleware can call `app.sessionManager.Get...` and `Put(...)` using `r.Context()`
3. on the way out, `LoadAndSave` commits any session changes back to the store and cookie

So the session manager itself is an app-level dependency, but the actual session contents are request-level data.

That is the subtle but important split.

## Handler Pattern for Phase 1

Your first handler should be a method on `application`:

```go
func (app *application) home(w http.ResponseWriter, r *http.Request)
```

That handler should do three things:

1. read request-scoped data
2. prepare template data
3. render a page

For example:

### Read session data

Because `LoadAndSave` already ran, the handler can do things like:

- `app.sessionManager.GetString(r.Context(), "flash")`
- `app.sessionManager.GetInt(r.Context(), "authenticatedUserID")`

### Prepare template data

Do not pass raw DB rows and random values directly into templates forever. Start a small `templateData` struct early.

For Phase 1 it can be tiny:

```go
type templateData struct {
    Flash string
}
```

Then a helper like:

```go
func (app *application) newTemplateData(r *http.Request) *templateData
```

can read common data from the session and request context and package it once.

### Render the template

Do not parse templates inside the handler. The handler should call a render helper that uses the cached templates from `app.templateCache`.

Conceptually:

```go
func (app *application) home(w http.ResponseWriter, r *http.Request) {
    data := app.newTemplateData(r)
    data.Flash = app.sessionManager.PopString(r.Context(), "flash")

    app.render(w, http.StatusOK, "home.tmpl", data)
}
```

The exact implementation can vary. The pattern should not.

## The Phase 1 Request Flow

For a request to `GET /`, the flow should feel like this:

```text
client
  -> http.Server
  -> app.routes() chi router
  -> chi middleware stack
  -> app.sessionManager.LoadAndSave
  -> app.home handler
  -> app.newTemplateData(r)
  -> app.render(...)
  -> template cache lookup
  -> HTML response
  -> scs saves any session changes
```

That is the full data path you want in your head.

## The Practical Wiring Plan

If I were wiring your current code forward, I would do it in this exact order.

### Step A: Fix the `application` ownership boundary

Make sure `main.go` does this:

- builds the pool
- builds the session manager
- builds the template cache
- stores them all on `app`
- calls `app.routes()`

Do **not** create a second session manager inside `routes.go`.

### Step B: Turn route handlers into methods

Instead of:

- `home`
- `login`
- `notebookList`

use:

- `app.home`
- `app.login`
- `app.notebookList`

This is dependency injection in practice. The method receiver is how the handler gets the shared app dependencies.

### Step C: Add a render helper

Create one render helper that:

- pulls the correct parsed template from the cache
- writes the HTTP status
- executes the template
- centralizes template error handling

This keeps handlers small.

### Step D: Add a small `templateData` type

Even before you have a current user type, start with:

- flash messages
- auth flag later
- CSRF token later

This becomes the stable contract between handlers and templates.

### Step E: Let session middleware run before any code that needs sessions

Any middleware or handler that reads session data must run **after**:

```go
r.Use(app.sessionManager.LoadAndSave)
```

That includes:

- flash message helpers
- current-user lookup middleware
- auth-required middleware

### Step F: Keep request data out of `application`

Do not put things like:

- current user
- current team
- current route param
- flash message

on the `application` struct.

Those belong in:

- `r.Context()`
- local variables inside handlers
- `templateData`

## The Pattern to Remember

When you feel lost, fall back to this:

- `application` owns shared dependencies
- `app.routes()` owns the router and middleware graph
- middleware enriches the request
- handlers are methods on `app`
- handlers read request-scoped values from `r` and `r.Context()`
- handlers call small helpers like `newTemplateData()` and `render()`

That is the recipe.

## What Phase 0 and Phase 1 Should Give You

By the end of these two phases, you should have:

- one `application` struct containing the DB pool, session manager, and template cache
- one `app.routes()` method returning the fully-wired `chi` router
- session middleware attached once at the router layer
- one simple page handler method that can read a session value and render HTML through the template cache

Once that works, the rest of the project becomes repetition of the same pattern, not a new architecture problem every time.
