package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
)

func TestRealTemplatesRenderShowcaseSurfaces(t *testing.T) {
	templates, err := newTemplateCache("../../ui/templates")
	if err != nil {
		t.Fatalf("failed to build real template cache: %v", err)
	}

	store := newFakeStore()
	store.revisions[3] = []data.NotebookRevision{{
		ID:         30,
		NotebookID: 3,
		AuthorID:   7,
		Title:      "Database Failover Runbook",
		Body:       "Promote the replica and verify recovery.",
		Status:     "draft",
	}}
	store.tags[3] = []data.Tag{{ID: 1, Name: "database"}}
	store.submitted = []data.NotebookRevision{{
		ID:         31,
		NotebookID: 3,
		AuthorID:   7,
		Title:      "Submitted Runbook",
		Status:     "submitted",
	}}
	store.auditEvents = []data.AuditEvent{{
		ID:         1,
		ActorID:    9,
		EventType:  "revision_approved",
		EntityType: "notebook_revision",
		EntityID:   31,
	}}
	store.flags = []data.ModerationFlag{{
		ID:          2,
		NotebookID:  3,
		ModeratorID: 7,
		Reason:      "Needs owner update",
	}}

	app := newTestApplication(store, templates)
	user := &data.User{ID: 7, Email: "member@example.com", Role: "member"}
	reviewer := &data.User{ID: 9, Email: "reviewer@example.com", Role: "reviewer"}
	admin := &data.User{ID: 10, Email: "admin@example.com", Role: "admin"}

	tests := []struct {
		name     string
		handler  http.HandlerFunc
		path     string
		user     *data.User
		setup    func(*http.Request) *http.Request
		contains []string
	}{
		{
			name:    "dashboard drafts",
			handler: app.dashboard,
			path:    "/dashboard",
			user:    user,
			contains: []string{
				"Your Recent Drafts",
				"Submit for Approval",
				"Database Failover Runbook",
			},
		},
		{
			name:    "notebook view workflow",
			handler: app.notebookView,
			path:    "/notebooks/3",
			user:    user,
			setup: func(r *http.Request) *http.Request {
				return addRouteParam(r, "id", "3")
			},
			contains: []string{
				"Database Failover Runbook",
				"Submit for Approval",
				"Flag Notebook",
				"database",
			},
		},
		{
			name:    "approval queue",
			handler: app.approvalQueue,
			path:    "/approvals",
			user:    reviewer,
			contains: []string{
				"Approval Queue",
				"Submitted Runbook",
				"Approve",
				"Reject",
			},
		},
		{
			name:    "moderation queue",
			handler: app.moderationQueue,
			path:    "/moderation",
			user:    reviewer,
			contains: []string{
				"Moderation Queue",
				"Needs owner update",
				"Resolve",
			},
		},
		{
			name:    "admin audit",
			handler: app.adminAudit,
			path:    "/admin/audit",
			user:    admin,
			contains: []string{
				"System Audit Logs",
				"revision_approved",
				"notebook_revision",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.setup != nil {
				req = tt.setup(req)
			}
			req = req.WithContext(contextSetUser(req.Context(), tt.user))

			rr := serveWithSession(app, tt.handler, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected status %d; got %d body=%q", http.StatusOK, rr.Code, rr.Body.String())
			}
			for _, want := range tt.contains {
				if !strings.Contains(rr.Body.String(), want) {
					t.Fatalf("expected body to contain %q; got %q", want, rr.Body.String())
				}
			}
		})
	}
}
