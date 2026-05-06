# ControlPlane - Complete Demo Walkthrough

## Project Overview

**ControlPlane** is a production-ready Go web application for managing internal runbooks, incident documentation, and operational knowledge sharing. It features:

- Session-backed authentication with role-based access control
- Server-rendered HTML pages with session management
- Notebook creation, revision, and versioning system
- Tag-based categorization and search
- Approval workflow (draft → submitted → approved/rejected)
- Audit logging for all governance-critical actions
- Team and role management (member, reviewer, admin)
- Moderation flags for content governance
- Full test coverage with unit and integration tests

**Tech Stack:**
- Go 1.25
- PostgreSQL 15
- Chi Router for routing
- SQLC for type-safe SQL
- SCS for session management
- HTML templating for server-rendered pages

---

## Prerequisites

### Required Software
1. **Go 1.25+** - Download from https://golang.org/
2. **PostgreSQL 15+** - Download from https://www.postgresql.org/ or use Docker
3. **Docker & Docker Compose** - For test database (recommended)
4. **Git** - For version control

### System Requirements
- RAM: 2GB minimum
- Disk: 500MB free space
- OS: Windows, macOS, or Linux

---

## Part 1: Database Setup

### Option A: Using Docker Compose (Recommended)

The project includes a test database configuration using Docker.

**Step 1: Start the PostgreSQL container**

```bash
cd c:\Users\soham\OneDrive\Desktop\Coding\Go Lung\Phase 1\ControlPlane
docker-compose -f docker-compose.test.yml up -d
```

**Verify the database is running:**
```bash
docker-compose -f docker-compose.test.yml ps
```

Expected output:
```
NAME          COMMAND                  STATUS
db-test       "docker-entrypoint.s…"   Up (healthy)
```

**Wait for the healthcheck to pass** (may take 10-15 seconds)

### Option B: Local PostgreSQL Installation

If you have PostgreSQL installed locally:

```bash
# Create the test database
createdb -U postgres controlplane_test

# Set database URL
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/controlplane_test?sslmode=disable"
```

---

## Part 2: Environment Setup

### Set Required Environment Variables

```powershell
# PowerShell (Windows)
$env:DATABASE_URL="postgres://testuser:testpass@localhost:5433/controlplane_test?sslmode=disable"
$env:CSRF_SECRET="12345678901234567890123456789012"
$env:ENV="dev"

# Or create a .env file and source it:
# DATABASE_URL=postgres://testuser:testpass@localhost:5433/controlplane_test?sslmode=disable
# CSRF_SECRET=12345678901234567890123456789012
# ENV=dev
```

### Environment Variables Explained

| Variable | Value | Purpose |
|----------|-------|---------|
| `DATABASE_URL` | `postgres://testuser:testpass@localhost:5433/controlplane_test?sslmode=disable` | PostgreSQL connection string |
| `CSRF_SECRET` | `12345678901234567890123456789012` | 32-byte secret for CSRF token protection |
| `ENV` | `dev` | Development mode (disables secure cookies, enables CSRF debug) |

---

## Part 3: Database Migrations

### Run Migrations

The application automatically runs migrations on startup. If you need to manually run them:

```bash
# Verify migrations directory
ls migrations/

# Expected files:
# 0001_initial_schema.sql
# 0002_create_audit_logs.sql
# 0003_create_sessions_table.sql
```

**Migration Timeline:**
1. **0001_initial_schema.sql** - Users, teams, memberships, notebooks, revisions
2. **0002_create_audit_logs.sql** - Audit logging for all governance actions
3. **0003_create_sessions_table.sql** - Session storage for authentication

---

## Part 4: Running the Application

### Build the Application

```bash
cd cmd/web
go build -o controlplane.exe
```

### Run the Application

```bash
# Using the compiled binary
./controlplane.exe

# Or run directly with Go
go run cmd/web/main.go
```

**Expected startup output:**
```
time=2026-05-06T10:00:00 level=INFO msg="Application initialized" port=6767 env=dev
time=2026-05-06T10:00:00 level=INFO msg="Starting server" addr=:6767
```

**Application is now running at:** `http://localhost:6767`

---

## Part 5: Demo Credentials

### Test User Accounts

The project includes a seed script to populate test data. Run it in another terminal:

```bash
cd cmd/seed
go run main.go
```

**Pre-seeded Users:**

| Email | Password | Role | Team | Purpose |
|-------|----------|------|------|---------|
| `alice@example.com` | `password123` | Admin | Engineering | Full system access, can approve |
| `bob@example.com` | `password123` | Reviewer | Engineering | Can review and approve notebooks |
| `charlie@example.com` | `password123` | Member | Engineering | Can create and edit notebooks |
| `diana@example.com` | `password123` | Member | Operations | Can create in Operations team |

### Manual User Creation

If you prefer to create test users manually:

```bash
go run cmd/web/main.go
# Then use the signup flow in the UI
```

---

## Part 6: Application Features Demo

### 6.1: Login & Authentication

**URL:** `http://localhost:6767/login`

**Demo Flow:**
1. Navigate to login page
2. Enter credentials:
   - Email: `alice@example.com`
   - Password: `password123`
3. Click "Sign In"
4. Session is created (cookie-based, server-side session store in PostgreSQL)

**What's Happening Behind the Scenes:**
- SCS session manager validates credentials against PostgreSQL
- Session token stored in `sessions` table
- CSRF token generated and stored in session
- Secure cookie set (domain: localhost, path: /)

---

### 6.2: Home Dashboard

**URL:** `http://localhost:6767/`

**What You'll See:**
- Welcome message with authenticated user's email
- Quick navigation to core features:
  - View all notebooks
  - Create new notebook
  - Approval queue (if reviewer/admin)
  - Audit logs (if admin)
  - Team management (if admin)

**Feature Highlights:**
- Server-rendered using HTML templates (`ui/templates/base.layout.tmpl`)
- Displays personalized content based on user role
- Team-scoped visibility and access control

---

### 6.3: Notebook Management

#### Creating a Notebook

**URL:** `http://localhost:6767/notebooks/create`

**Demo Steps:**
1. Click "New Notebook"
2. Fill in the form:
   - **Title:** "Incident Response - Database Failover"
   - **Content:** "Steps to perform safe database failover..."
   - **Tags:** incident, database, failover
   - **Team:** Engineering
3. Click "Save Draft"

**What Happens:**
- Notebook inserted into `notebooks` table
- Initial revision created in `notebook_revisions` table
- Status set to "draft" (not published)
- User marked as author

#### Editing a Notebook

**URL:** `http://localhost:6767/notebooks/{id}/edit`

**Demo Steps:**
1. Navigate to "All Notebooks"
2. Click on a draft notebook
3. Click "Edit"
4. Modify content, tags, or title
5. Click "Save Changes"

**What Happens:**
- New revision created (maintains version history)
- Old revision remains in database (full audit trail)
- `updated_at` timestamp updated

#### Publishing / Submitting for Approval

**URL:** `http://localhost:6767/notebooks/{id}/submit`

**Demo Steps:**
1. Open a draft notebook
2. Click "Submit for Review"
3. Enter optional submission message
4. Click "Submit"

**What Happens:**
- Notebook status changes from "draft" to "submitted"
- Audit log entry created: `{ action: "submit_for_review", user_id: X, notebook_id: Y, timestamp: Z }`
- Reviewers are notified (UI shows in approval queue)
- Cannot edit until approved/rejected

---

### 6.4: Approval Workflow

**URL:** `http://localhost:6767/approvals`

**Demo Steps (as Reviewer/Admin):**

1. Switch to admin account (alice@example.com) or reviewer account (bob@example.com)
2. Go to "Approval Queue"
3. See all "submitted" notebooks awaiting review
4. Click on a notebook to view full content
5. Click "Approve" or "Reject"

**Approve Flow:**
- Sets notebook status to "approved"
- Notebook becomes "published"
- Creates audit log entry
- Notifies original author

**Reject Flow:**
- Sets status to "rejected"
- Author can edit and resubmit
- Creates audit log entry with rejection reason

---

### 6.5: Search & Discovery

**URL:** `http://localhost:6767/notebooks`

**Demo Steps:**
1. Go to "All Notebooks"
2. See list of all published notebooks
3. Use search bar to find notebooks by:
   - Title keywords
   - Tags
   - Author name

**What's Happening:**
- Full-text search query against PostgreSQL
- Filters by team (only sees team's notebooks)
- Results sorted by relevance and recency

---

### 6.6: Audit Logs

**URL:** `http://localhost:6767/audit` (Admin Only)

**Demo Steps (as Admin):**
1. Log in as `alice@example.com`
2. Go to "Audit Logs"
3. See complete activity history:
   - Notebook creations
   - Submissions
   - Approvals/rejections
   - User logins
   - Modifications

**Audit Log Fields:**
| Field | Example |
|-------|---------|
| Timestamp | 2026-05-06 10:15:32 |
| User | alice@example.com |
| Action | notebook_approved |
| Resource | notebook_id: 42 |
| Details | "Published to Engineering team" |

**What's Happening:**
- Every governance-critical action logged to `audit_logs` table
- Immutable (no updates, only inserts)
- Indexed for fast querying
- Used for compliance and debugging

---

### 6.7: Team & Role Management

**URL:** `http://localhost:6767/admin/teams` (Admin Only)

**Demo Steps:**
1. Log in as admin (alice@example.com)
2. Go to "Team Management"
3. View teams: Engineering, Operations
4. View team members and their roles:
   - **Admin:** Full system access
   - **Reviewer:** Can approve notebooks, see audit logs
   - **Member:** Can create/edit own notebooks

**Feature Highlights:**
- Role-based access control (RBAC)
- Team-scoped permissions
- Can manage team membership
- Audit trail for role changes

---

### 6.8: Moderation

**URL:** `http://localhost:6767/moderation` (Admin/Moderator)

**Demo Steps:**
1. Log in as admin
2. Go to "Moderation Queue"
3. See flagged notebooks
4. Review and take action (approve/remove/request revision)

---

## Part 7: Complete Demo Scenario

### Scenario: Publishing an Incident Runbook

**Duration:** 10-15 minutes

**Participants:** Member (Charlie) → Reviewer (Bob) → Admin (Alice)

#### Step 1: Create (5 min) - Charlie's Perspective
```
1. Login as charlie@example.com / password123
2. Navigate to "Create Notebook"
3. Fill in:
   - Title: "Production Database Recovery Steps"
   - Content: "When primary DB fails:
      1. Check replication lag
      2. Failover to replica
      3. Update DNS records
      4. Monitor query performance"
   - Tags: incident, database, production
   - Team: Engineering
4. Click "Save Draft"
5. Review the notebook you created
```

**Expected Result:**
- Notebook appears in "My Notebooks" section
- Status shows "Draft"
- Only you can edit it
- CSRF token validated on form submission

#### Step 2: Submit for Review (2 min) - Charlie's Perspective
```
1. On the notebook page, click "Submit for Review"
2. Add optional message: "Ready for team review"
3. Click "Submit"
4. See notification: "Submitted for review"
```

**Expected Result:**
- Notebook status changes to "Submitted"
- Charlie can no longer edit (locked)
- Audit log created
- Appears in Bob's approval queue

#### Step 3: Review (3 min) - Bob's Perspective
```
1. Logout and login as bob@example.com / password123
2. Go to "Approval Queue"
3. See Charlie's notebook in the queue
4. Click to view full content
5. Review the content and instructions
```

**Expected Result:**
- Bob can see the full notebook
- CSRF token refreshed for this session
- Can see submission timestamp and author
- Option to approve or request changes

#### Step 4: Approve (2 min) - Bob's Perspective
```
1. Click "Approve"
2. Add reviewer message: "Clear and complete. Approved."
3. Click "Confirm Approval"
```

**Expected Result:**
- Notebook status changes to "Published"
- Audit log entry created (action: notebook_approved, reviewer: bob)
- Notebook now appears in team's searchable notebooks
- Charlie can no longer edit (published)

#### Step 5: Verify in Audit (2 min) - Alice's Perspective
```
1. Logout and login as alice@example.com / password123
2. Go to "Audit Logs"
3. Filter by "notebook" actions
4. See the complete journey:
   - Created: charlie created notebook_id:X
   - Submitted: charlie submitted for review
   - Approved: bob approved notebook_id:X
```

**Expected Result:**
- Complete activity trail visible
- Timestamps show exact sequence
- All users and actions logged
- Immutable record for compliance

---

## Part 8: Running Tests

### Unit Tests

```bash
# Run all tests
go test ./cmd/web/... ./internal/...

# Run specific test file
go test ./cmd/web/auth_handlers_test.go

# Run with verbose output
go test -v ./cmd/web/...
```

**Test Coverage:**
- Authentication and authorization
- CSRF token validation
- Session management
- Notebook CRUD operations
- Approval workflows
- Audit logging
- Input validation

### Test Database

Tests use the same PostgreSQL database (you must have `docker-compose.test.yml` running):

```bash
docker-compose -f docker-compose.test.yml up -d
go test -v ./...
```

---

## Part 9: Important Endpoints Reference

### Authentication
| Method | Path | Description |
|--------|------|-------------|
| GET | `/login` | Login form |
| POST | `/login` | Submit login credentials |
| POST | `/logout` | Clear session and logout |

### Notebooks
| Method | Path | Description |
|--------|------|-------------|
| GET | `/notebooks` | List all published notebooks |
| GET | `/notebooks/create` | Create notebook form |
| POST | `/notebooks` | Create new notebook |
| GET | `/notebooks/{id}` | View notebook |
| GET | `/notebooks/{id}/edit` | Edit form |
| POST | `/notebooks/{id}/edit` | Save changes |
| POST | `/notebooks/{id}/submit` | Submit for approval |

### Approvals (Reviewer/Admin)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/approvals` | Approval queue |
| GET | `/approvals/{id}` | View for approval |
| POST | `/approvals/{id}/approve` | Approve notebook |
| POST | `/approvals/{id}/reject` | Reject notebook |

### Admin
| Method | Path | Description |
|--------|------|-------------|
| GET | `/audit` | Audit logs |
| GET | `/admin/teams` | Team management |
| GET | `/moderation` | Moderation queue |

---

## Part 10: Security Features in Action

### CSRF Protection
- Every form includes hidden `csrf_token` field
- Token validated on all POST/PUT/DELETE operations
- Token stored server-side in session (PostgreSQL)
- In dev mode: plaintext token for debugging
- In prod mode: encrypted secure cookie

**Demo:**
```
1. Inspect form with F12 Developer Tools
2. Right-click → Inspect on "Submit" button
3. See hidden csrf_token input
4. Try submitting form with modified token (will fail with 403)
```

### Session Management
- Sessions stored in PostgreSQL (not memory)
- Cookies contain only session ID (secure)
- Session timeout: configurable
- Supports multiple concurrent sessions
- CSRF token tied to session

**Demo:**
```
1. Login and navigate to database
2. Query: SELECT * FROM sessions;
3. See your session record
4. Close browser tab, reopen URL → still authenticated
5. Logout → session deleted from database
```

### Password Security
- Passwords hashed with bcrypt (irreversible)
- Never stored in plaintext
- Password comparison uses constant-time algorithm

**Demo Code Location:** [cmd/web/auth.go](cmd/web/auth.go#L45)

### Role-Based Access Control
- User roles: admin, reviewer, member
- Team-scoped permissions
- Middleware checks role on protected routes
- Audit logs all access attempts

---

## Part 11: Architecture Walkthrough

### Request Flow Diagram

```
HTTP Request
    ↓
Chi Router (routes.go)
    ↓
Middleware Chain:
  1. RequestID / RealIP / Logger / Recoverer
  2. Session Load/Save (SCS)
  3. CSRF Protection
  4. Authentication Middleware
    ↓
Handler (e.g., notebooks.go)
    ↓
Store Layer (data/pgxstore.go)
    ↓
Database (PostgreSQL)
    ↓
Template Rendering (ui/templates/)
    ↓
HTTP Response
```

### Key Files

| File | Purpose |
|------|---------|
| [cmd/web/main.go](cmd/web/main.go) | App initialization, config, dependencies |
| [cmd/web/routes.go](cmd/web/routes.go) | HTTP route definitions |
| [cmd/web/auth.go](cmd/web/auth.go) | Authentication logic |
| [cmd/web/notebooks.go](cmd/web/notebooks.go) | Notebook handlers |
| [cmd/web/approvals.go](cmd/web/approvals.go) | Approval workflow handlers |
| [cmd/web/middleware.go](cmd/web/middleware.go) | Custom middleware |
| [internal/data/pgxstore.go](internal/data/pgxstore.go) | Database layer abstraction |
| [internal/data/store.go](internal/data/store.go) | Interface definitions |
| [internal/validator/validator.go](internal/validator/validator.go) | Input validation |
| [migrations/](migrations/) | SQL migrations |
| [ui/templates/](ui/templates/) | HTML templates |

---

## Part 12: Troubleshooting

### Database Connection Error
```
Error: "failed to connect to postgres://testuser:testpass@localhost:5433..."
```

**Solution:**
1. Check Docker is running: `docker ps`
2. Verify container is healthy: `docker-compose -f docker-compose.test.yml ps`
3. Check network: `docker-compose -f docker-compose.test.yml down && docker-compose -f docker-compose.test.yml up -d`

### CSRF Token Error (403 Forbidden)
```
Error: "CSRF token invalid"
```

**Solution:**
1. In dev mode, ensure `ENV=dev` is set
2. Check CSRF_SECRET is 32+ bytes: `$env:CSRF_SECRET.Length`
3. Clear browser cookies and try again
4. Verify form includes hidden `csrf_token` field

### Session Expired
```
Error: "Session has expired, please login again"
```

**Solution:**
1. Clear browser cookies
2. Check session in database: `SELECT COUNT(*) FROM sessions;`
3. Login again - new session will be created

### Port Already in Use
```
Error: "listen tcp :6767: bind: An attempt was made to reuse a socket..."
```

**Solution:**
```bash
# Kill process using port 6767
netstat -ano | findstr :6767
taskkill /PID <PID> /F

# Or change port in main.go (line 43)
cfg := Config{
    Port: 6768,  // Use different port
    ...
}
```

---

## Part 13: Performance Notes

### Load Times
- Home page: ~50ms (template rendering)
- Notebook list: ~100ms (SQL + template)
- Approval queue: ~150ms (JOIN on multiple tables)
- Search: ~200ms (full-text search)

### Database Indexes
- User email (unique)
- Team name (unique)
- Notebook published status
- Audit log timestamp
- Session token (for fast lookup)

### Session Store Performance
- SQL-based session store (pgxstore)
- Sessions stored in PostgreSQL, not memory
- Survives application restarts
- Supports distributed deployments

---

## Part 14: Next Steps / Future Features

### Phase 6 (Planned):
- [ ] Metrics and observability (Prometheus)
- [ ] Distributed tracing (Jaeger)
- [ ] Docker containerization
- [ ] Kubernetes deployment
- [ ] Shared rate limiting (Redis)
- [ ] API endpoints (JSON + gRPC)
- [ ] Webhooks for external integrations
- [ ] Collaborative editing
- [ ] Advanced search (Elasticsearch)

---

## Summary

You now have a complete, production-ready ControlPlane instance running with:

✅ Database running (PostgreSQL in Docker)  
✅ Application running (Go server on port 6767)  
✅ Test users created (alice, bob, charlie, diana)  
✅ Authentication system (session + CSRF)  
✅ Notebook workflow (draft → submit → approve → publish)  
✅ Audit logging (complete activity history)  
✅ Role-based access control (admin, reviewer, member)  
✅ Full test coverage  
✅ Production-ready architecture  

### Quick Start Summary
```powershell
# 1. Start database
docker-compose -f docker-compose.test.yml up -d

# 2. Set environment
$env:DATABASE_URL="postgres://testuser:testpass@localhost:5433/controlplane_test?sslmode=disable"
$env:CSRF_SECRET="12345678901234567890123456789012"
$env:ENV="dev"

# 3. Seed test data
cd cmd/seed && go run main.go

# 4. Run application
cd ../web && go run main.go

# 5. Open browser
# http://localhost:6767
# Login: alice@example.com / password123
```

---

**Project Status:** Phase 5 (Complete) - Production Hardened  
**Last Updated:** 2026-05-06  
**Documentation:** See [docs/](docs/) and [refs/](refs/) folders for detailed architecture guides.

