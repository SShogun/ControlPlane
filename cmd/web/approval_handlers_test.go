package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
)

func TestApproveRevisionSubmitRunsApprovalTransaction(t *testing.T) {
	store := newFakeStore()
	app := newTestApplication(store, nil)
	reviewer := &data.User{ID: 42, Email: "reviewer@example.com"}

	req := httptest.NewRequest(http.MethodPost, "/approvals/approve", nil)
	req = addRouteParam(req, "docID", "9")
	req = addRouteParam(req, "revID", "3")
	req = req.WithContext(contextSetUser(req.Context(), reviewer))

	rr := serveWithSession(app, app.approveRevisionSubmit, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d; got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/notebooks/9" {
		t.Fatalf("expected redirect to /notebooks/9; got %q", got)
	}
	if !store.approveCalled {
		t.Fatal("ApproveRevisionTx should be called")
	}
	if store.approvedRevID != 3 || store.approvedDocID != 9 || store.approvedUser != reviewer.ID {
		t.Fatalf("unexpected approval params: rev=%d doc=%d user=%d", store.approvedRevID, store.approvedDocID, store.approvedUser)
	}
}

func TestRejectRevisionSubmitUpdatesStatus(t *testing.T) {
	store := newFakeStore()
	app := newTestApplication(store, nil)
	reviewer := &data.User{ID: 42, Email: "reviewer@example.com"}

	req := httptest.NewRequest(http.MethodPost, "/approvals/reject", nil)
	req = addRouteParam(req, "revID", "3")
	req = req.WithContext(contextSetUser(req.Context(), reviewer))

	rr := serveWithSession(app, app.rejectRevisionSubmit, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d; got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/approvals" {
		t.Fatalf("expected redirect to /approvals; got %q", got)
	}
	if !store.updateStatusCalled {
		t.Fatal("UpdateRevisionStatus should be called")
	}
	if store.updateStatusParams.ID != 3 || store.updateStatusParams.Status != "rejected" {
		t.Fatalf("unexpected status update params: %+v", store.updateStatusParams)
	}
}
