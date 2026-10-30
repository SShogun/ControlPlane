package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
)

type approvalErrorStore struct {
	*fakeStore
	approveErr   error
	rejectErr    error
	state        string
	approveCalls int
	rejectCalls  int
}

func (s *approvalErrorStore) ApproveRevisionTx(_ context.Context, _, _, _ int, _ string) error {
	s.approveCalls++
	if s.approveErr != nil {
		return s.approveErr
	}
	if s.state != "submitted" {
		return data.ErrInvalidTransition
	}
	s.state = "approved"
	return nil
}

func (s *approvalErrorStore) RejectRevisionTx(_ context.Context, _, _ int, _ string) error {
	s.rejectCalls++
	if s.rejectErr != nil {
		return s.rejectErr
	}
	if s.state != "submitted" {
		return data.ErrInvalidTransition
	}
	s.state = "rejected"
	return nil
}

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

func TestApprovalHandlersMapTransitionErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   int
		reject bool
	}{
		{name: "approve missing revision", err: data.ErrNotFound, want: http.StatusNotFound},
		{name: "approve already decided", err: data.ErrInvalidTransition, want: http.StatusConflict},
		{name: "reject missing revision", err: data.ErrNotFound, want: http.StatusNotFound, reject: true},
		{name: "reject already decided", err: data.ErrInvalidTransition, want: http.StatusConflict, reject: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &approvalErrorStore{fakeStore: newFakeStore(), approveErr: tt.err, rejectErr: tt.err}
			app := newTestApplication(store.fakeStore, nil)
			app.store = store
			reviewer := &data.User{ID: 42, Email: "reviewer@example.com"}
			path := "/approvals/approve"
			handler := app.approveRevisionSubmit
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req = addRouteParam(req, "docID", "9")
			req = addRouteParam(req, "revID", "3")
			if tt.reject {
				handler = app.rejectRevisionSubmit
				req = httptest.NewRequest(http.MethodPost, "/approvals/reject", nil)
				req = addRouteParam(req, "revID", "3")
			}
			req = req.WithContext(contextSetUser(req.Context(), reviewer))
			rr := serveWithSession(app, handler, req)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%q", rr.Code, tt.want, rr.Body.String())
			}
		})
	}
}

func TestApprovalHandlersRejectInvalidIDs(t *testing.T) {
	for _, reject := range []bool{false, true} {
		name := "approve"
		if reject {
			name = "reject"
		}
		t.Run(name, func(t *testing.T) {
			store := &approvalErrorStore{fakeStore: newFakeStore(), state: "submitted"}
			app := newTestApplication(store.fakeStore, nil)
			app.store = store
			path, handler := "/approvals/approve", app.approveRevisionSubmit
			if reject {
				path, handler = "/approvals/reject", app.rejectRevisionSubmit
			}
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req = addRouteParam(req, "revID", "not-an-id")
			req = addRouteParam(req, "docID", "9")
			req = req.WithContext(contextSetUser(req.Context(), &data.User{ID: 42}))
			rr := serveWithSession(app, handler, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
			if store.approveCalls != 0 || store.rejectCalls != 0 {
				t.Fatal("invalid ID reached the store")
			}
		})
	}
}

func TestApprovalHandlerRepeatedDecisionReturnsConflict(t *testing.T) {
	store := &approvalErrorStore{fakeStore: newFakeStore(), state: "submitted"}
	app := newTestApplication(store.fakeStore, nil)
	app.store = store
	reviewer := &data.User{ID: 42, Email: "reviewer@example.com"}
	makeRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/approvals/approve", nil)
		req = addRouteParam(req, "docID", "9")
		req = addRouteParam(req, "revID", "3")
		return req.WithContext(contextSetUser(req.Context(), reviewer))
	}
	first := serveWithSession(app, app.approveRevisionSubmit, makeRequest())
	if first.Code != http.StatusSeeOther {
		t.Fatalf("first approval status = %d, want %d", first.Code, http.StatusSeeOther)
	}
	second := serveWithSession(app, app.approveRevisionSubmit, makeRequest())
	if second.Code != http.StatusConflict {
		t.Fatalf("repeat approval status = %d, want %d", second.Code, http.StatusConflict)
	}
	if store.approveCalls != 2 || store.state != "approved" {
		t.Fatalf("unexpected repeated approval state: calls=%d state=%q", store.approveCalls, store.state)
	}
}
