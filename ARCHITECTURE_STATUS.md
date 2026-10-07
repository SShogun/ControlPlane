# ControlPlane Architecture Status

This document describes the repository as it exists today. It is intentionally narrower than the project name: ControlPlane is currently a **server-rendered Go application for internal knowledge and governance workflows**, not a Kubernetes-style controller or a separate distributed control-plane/data-plane system.

## Current Shape

The application is one Go process built around `net/http` and `chi`.

Request flow:

```text
HTTP request
  -> chi middleware
  -> session load/save
  -> CSRF protection
  -> authentication / role checks
  -> handler
  -> UserStore interface
  -> PostgreSQL
  -> server-rendered HTML response
```

PostgreSQL is the primary durable store. The same database also backs server-side sessions through `scs/pgxstore`.

## Implemented

- Email/password authentication with bcrypt verification.
- PostgreSQL-backed sessions and secure-cookie behavior in production mode.
- CSRF protection on the web application.
- Role checks for authenticated, reviewer, and admin routes.
- Notebook creation, revision history, tags, search, submission, and deletion flows.
- Reviewer approval/rejection workflows.
- Transactional approval/rejection persistence with approval records and audit events.
- Moderation flags and admin audit/team workflows.
- Graceful HTTP shutdown on `SIGINT`/`SIGTERM`, followed by PostgreSQL pool cleanup.
- Handler tests using fake stores and `httptest`.
- PostgreSQL integration tests using `TEST_DATABASE_URL`.
- GitHub Actions verification with migrations, formatting, vet, race-enabled tests, build, and golangci-lint.

## Important Runtime Semantics

Approval and rejection use explicit PostgreSQL transactions. Their revision state change, approval record, and audit event are committed together.

Normal handler-side audit calls outside those transaction methods are synchronous database writes. They are not an external event bus and do not provide cross-service delivery guarantees.

The application currently runs as a single process. Session state is centralized in PostgreSQL, but there is no replica coordination layer for rate limiting or background work.

## Not Implemented Yet

The following are future work and should not be inferred from the repository name or architecture diagrams:

- Prometheus metrics endpoint.
- OpenTelemetry tracing.
- Production Docker image and Kubernetes manifests.
- `/healthz` and `/readyz` probe endpoints.
- Redis-backed distributed rate limiting.
- gRPC or external service-to-service APIs.
- A separate control-plane service and data-plane service.
- Kubernetes reconciliation/controller behavior.
- Distributed consensus, leader election, or multi-region coordination.

## Verification Boundary

CI proves that the checked-in migrations apply to PostgreSQL and that the Go code passes formatting, vet, race-enabled tests, build, and lint checks.

The repository does **not** currently claim production deployment validation, load-test SLOs, multi-node failover testing, or Kubernetes operational readiness.
