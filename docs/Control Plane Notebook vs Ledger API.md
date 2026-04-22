# Control Plane Notebook vs Ledger API

## The Correct Order

The corrected sequence is:

1. Control Plane Notebook
2. Ledger API

That order is better because Control Plane Notebook teaches the full browser-oriented backend loop first, and Ledger then strips the UI away and forces you to think in terms of JSON contracts and API behavior.

## What Control Plane Notebook Teaches That Ledger Does Not Force Early

### 1. Server-rendered HTML

Control Plane Notebook requires you to learn:

- layout templates
- partials
- page-specific view models
- static assets
- rendering helpers

Ledger API does not force any of that because it returns JSON instead of HTML.

### 2. Form handling and PRG

Control Plane Notebook requires:

- parsing form submissions
- repopulating fields
- field-level error display
- Post/Redirect/Get after writes

Ledger API mostly replaces this with JSON decoding and error envelopes.

### 3. Sessions, cookies, and CSRF

Control Plane Notebook requires:

- session-backed auth
- cookie handling
- flash messages
- CSRF protection

Ledger API replaces this with bearer-token style authentication and therefore skips a large part of browser security and session state.

### 4. UI-shaped authorization

Control Plane Notebook forces you to think about:

- who can see this page
- who can see this button
- who can submit this draft
- who can approve this revision
- who can moderate this content

Ledger API still has auth and authorization, but it is more endpoint-shaped than page-shaped.

### 5. Workflow and governance

Control Plane Notebook introduces:

- drafts
- approvals
- audit trails
- moderation queues
- role and team based visibility

Ledger API is more operational CRUD plus API concerns like version checks, pagination, rate limits, and bearer auth.

### 6. `pgx`, `pgxpool`, and `sqlc`

Control Plane Notebook is a good place to learn:

- `pgx` as the PostgreSQL-native driver
- `pgxpool` as your shared connection pool
- handwritten SQL plus generated typed code with `sqlc`

Ledger API can absolutely reuse those ideas, but the notebook project is the better place to absorb them because the domain is more relational and page-query-heavy.

### 7. SQL maturity

Control Plane Notebook should push you into:

- schema design
- foreign keys
- uniqueness
- partial indexes
- full-text search basics
- transactions around approval plus audit writes
- `EXPLAIN`

Ledger API still needs SQL discipline, but Notebook is where the relational model is richer and more educational.

## What Carries Forward from Control Plane Notebook into Ledger API

These habits transfer directly:

- keeping `main.go` small
- dependency injection through an application struct
- clear route organization
- middleware chains
- validation helpers
- centralized error handling
- request-scoped context
- structured request logging
- testable handlers
- migration discipline
- thinking about indexes before performance problems hurt you

## What Changes When You Move from Notebook to Ledger

### What drops away

- templates
- static assets
- forms
- redirects
- sessions
- cookies
- CSRF
- flash messages

### What becomes more important

- JSON request and response contracts
- malformed JSON handling
- field error envelopes
- bearer token auth
- optimistic concurrency
- pagination metadata
- rate limiting
- operational endpoints like `/debug/vars`

## The Practical Difference in One Sentence

Control Plane Notebook teaches you how to build a serious web application.

Ledger API then teaches you how to expose the same backend discipline through a clean machine-facing interface.
