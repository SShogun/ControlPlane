package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
)

const (
	authNotebookID = 12
	authAuthorID   = 7
)

func authorizationDraftStore(teamID int, status string) *fakeStore {
	store := newFakeStore()
	store.addUser(data.User{ID: authAuthorID, Email: "author@example.com", Role: "member"})
	store.notebooks[authNotebookID] = data.Notebook{ID: authNotebookID, TeamID: teamID}
	store.revisions[authNotebookID] = []data.NotebookRevision{{
		ID:         34,
		NotebookID: authNotebookID,
		AuthorID:   authAuthorID,
		Title:      "Original title",
		Body:       "Original body",
		Status:     status,
	}}
	return store
}

func draftRequest(app *Application, handler http.HandlerFunc, user *data.User, form url.Values) *httptest.ResponseRecorder {
	var body *strings.Reader
	if form == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(http.MethodPost, "/notebooks/12/edit", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = addRouteParam(req, "id", "12")
	if user != nil {
		req = req.WithContext(contextSetUser(req.Context(), user))
	}
	return serveWithSession(app, handler, req)
}

func TestNotebookDraftOwnerCanEditAndSubmit(t *testing.T) {
	for _, role := range []string{"member", "reviewer", "admin"} {
		t.Run(role, func(t *testing.T) {
			store := authorizationDraftStore(0, "draft")
			user := &data.User{ID: authAuthorID, Email: "author@example.com", Role: role}
			app := newTestApplication(store, nil)
			editForm := url.Values{"title": {"Updated title"}, "body": {"Updated body"}, "tags": {"go"}}

			editResponse := draftRequest(app, app.notebookEditSubmit, user, editForm)
			if editResponse.Code != http.StatusSeeOther {
				t.Fatalf("edit returned %d: %q", editResponse.Code, editResponse.Body.String())
			}
			if store.updateDraftCalls != 1 || len(store.revisions[authNotebookID]) != 2 {
				t.Fatalf("edit did not create exactly one revision: calls=%d revisions=%d", store.updateDraftCalls, len(store.revisions[authNotebookID]))
			}
			if got := store.revisions[authNotebookID][0]; got.Title != "Updated title" || got.Body != "Updated body" || got.Status != "draft" {
				t.Fatalf("unexpected edited revision: %+v", got)
			}

			submitForm := url.Values{"message": {"Ready for review"}}
			submitRequest := httptest.NewRequest(http.MethodPost, "/notebooks/12/submit", strings.NewReader(submitForm.Encode()))
			submitRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			submitRequest = addRouteParam(submitRequest, "id", "12")
			submitRequest = submitRequest.WithContext(contextSetUser(submitRequest.Context(), user))
			submitResponse := serveWithSession(app, app.notebookSubmitForApproval, submitRequest)
			if submitResponse.Code != http.StatusSeeOther {
				t.Fatalf("submit returned %d: %q", submitResponse.Code, submitResponse.Body.String())
			}
			if store.submitDraftCalls != 1 || store.revisions[authNotebookID][0].Status != "submitted" {
				t.Fatalf("submit did not change the current draft: calls=%d revision=%+v", store.submitDraftCalls, store.revisions[authNotebookID][0])
			}
		})
	}
}

func TestNotebookDraftRejectsNonOwnerWithoutMutation(t *testing.T) {
	store := authorizationDraftStore(0, "draft")
	app := newTestApplication(store, nil)
	other := &data.User{ID: 8, Email: "other@example.com", Role: "admin"}
	form := url.Values{"title": {"Attempted edit"}, "body": {"Attempted body"}}

	for _, handler := range []struct {
		name string
		fn   http.HandlerFunc
	}{
		{name: "edit", fn: app.notebookEditSubmit},
		{name: "submit", fn: app.notebookSubmitForApproval},
	} {
		t.Run(handler.name, func(t *testing.T) {
			response := draftRequest(app, handler.fn, other, form)
			if response.Code != http.StatusForbidden {
				t.Fatalf("returned %d, want 403: %q", response.Code, response.Body.String())
			}
			if store.updateDraftCalls != 0 || store.submitDraftCalls != 0 || store.auditInsertCalls != 0 || store.tagCreateCalls != 0 || store.tagAttachCalls != 0 {
				t.Fatalf("unauthorized request mutated store: updates=%d submits=%d audits=%d tag creates=%d tag attaches=%d", store.updateDraftCalls, store.submitDraftCalls, store.auditInsertCalls, store.tagCreateCalls, store.tagAttachCalls)
			}
			if got := store.revisions[authNotebookID]; len(got) != 1 || got[0].Title != "Original title" || got[0].Status != "draft" {
				t.Fatalf("unauthorized request changed revisions: %+v", got)
			}
		})
	}
}

func TestNotebookDraftTeamMembershipControlsOwnerEdit(t *testing.T) {
	for _, member := range []bool{false, true} {
		name := "nonmember"
		if member {
			name = "member"
		}
		t.Run(name, func(t *testing.T) {
			store := authorizationDraftStore(3, "draft")
			if member {
				store.memberships[3] = map[int]bool{authAuthorID: true}
			}
			app := newTestApplication(store, nil)
			user := &data.User{ID: authAuthorID, Email: "author@example.com", Role: "member"}
			form := url.Values{"title": {"Updated title"}, "body": {"Updated body"}}

			response := draftRequest(app, app.notebookEditSubmit, user, form)
			if member {
				if response.Code != http.StatusSeeOther || store.updateDraftCalls != 1 {
					t.Fatalf("team member edit returned %d and made %d updates", response.Code, store.updateDraftCalls)
				}
			} else if response.Code != http.StatusForbidden || store.updateDraftCalls != 0 {
				t.Fatalf("nonmember edit returned %d and made %d updates", response.Code, store.updateDraftCalls)
			}
		})
	}
}

func TestNotebookDraftAnonymousAndMissingResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		store  *fakeStore
		user   *data.User
		status int
	}{
		{name: "anonymous", store: authorizationDraftStore(0, "draft"), status: http.StatusUnauthorized},
		{name: "missing notebook", store: newFakeStore(), user: &data.User{ID: authAuthorID, Role: "member"}, status: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestApplication(tc.store, nil)
			response := draftRequest(app, app.notebookEditSubmit, tc.user, url.Values{"title": {"Updated"}, "body": {"Body"}})
			if response.Code != tc.status {
				t.Fatalf("returned %d, want %d: %q", response.Code, tc.status, response.Body.String())
			}
			if tc.store.updateDraftCalls != 0 || tc.store.submitDraftCalls != 0 {
				t.Fatalf("request mutated store: updates=%d submits=%d", tc.store.updateDraftCalls, tc.store.submitDraftCalls)
			}
		})
	}
}

func TestReviewerAuthorizationStillAllowsReviewerAndAdmin(t *testing.T) {
	app := newTestApplication(nil, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	for _, tc := range []struct {
		role string
		want int
	}{{"member", http.StatusSeeOther}, {"reviewer", http.StatusOK}, {"admin", http.StatusOK}} {
		t.Run(tc.role, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/approvals", nil)
			req = req.WithContext(contextSetUser(req.Context(), &data.User{ID: 20, Role: tc.role}))
			response := serveWithSession(app, app.requireRole("reviewer")(next).ServeHTTP, req)
			if response.Code != tc.want {
				t.Fatalf("role %q returned %d, want %d", tc.role, response.Code, tc.want)
			}
		})
	}
}

func TestDeleteNotebookAsRequiresStoredAdminForApprovedNotebook(t *testing.T) {
	t.Run("fakeStore", func(t *testing.T) {
		store := authorizationDraftStore(0, "approved")
		result, err := store.DeleteNotebookAs(context.Background(), authNotebookID, authAuthorID, true)
		if err != data.ErrForbidden || result.DeletedNotebook || store.notebooks[authNotebookID].ID == 0 {
			t.Fatalf("member deletion = (%+v, %v), want forbidden without deletion", result, err)
		}
		store.addUser(data.User{ID: 21, Email: "admin@example.com", Role: "admin"})
		result, err = store.DeleteNotebookAs(context.Background(), authNotebookID, 21, true)
		if err != nil || !result.DeletedNotebook {
			t.Fatalf("admin deletion = (%+v, %v), want successful deletion", result, err)
		}
	})

	t.Run("mockStore", func(t *testing.T) {
		store := newMockStore()
		store.notebooks[authNotebookID] = &data.Notebook{ID: authNotebookID}
		store.revisions[authNotebookID] = []data.NotebookRevision{{ID: 34, NotebookID: authNotebookID, AuthorID: authAuthorID, Status: "approved"}}
		store.users[authAuthorID] = &data.User{ID: authAuthorID, Role: "reviewer"}
		result, err := store.DeleteNotebookAs(context.Background(), authNotebookID, authAuthorID, true)
		if err != data.ErrForbidden || result.DeletedNotebook || store.notebooks[authNotebookID] == nil {
			t.Fatalf("reviewer deletion = (%+v, %v), want forbidden without deletion", result, err)
		}
		store.users[21] = &data.User{ID: 21, Role: "admin"}
		result, err = store.DeleteNotebookAs(context.Background(), authNotebookID, 21, true)
		if err != nil || !result.DeletedNotebook {
			t.Fatalf("admin deletion = (%+v, %v), want successful deletion", result, err)
		}
	})
}

func TestDeleteNotebookAsRejectsUnstoredAdminClaim(t *testing.T) {
	store := authorizationDraftStore(0, "approved")
	result, err := store.DeleteNotebookAs(context.Background(), authNotebookID, 99, true)
	if err != data.ErrForbidden || result.DeletedNotebook || store.notebooks[authNotebookID].ID == 0 {
		t.Fatalf("unstored admin deletion = (%+v, %v), want forbidden without deletion", result, err)
	}

	mock := newMockStore()
	mock.notebooks[authNotebookID] = &data.Notebook{ID: authNotebookID}
	mock.revisions[authNotebookID] = []data.NotebookRevision{{ID: 34, NotebookID: authNotebookID, AuthorID: authAuthorID, Status: "approved"}}
	result, err = mock.DeleteNotebookAs(context.Background(), authNotebookID, 99, true)
	if err != data.ErrForbidden || result.DeletedNotebook || mock.notebooks[authNotebookID] == nil {
		t.Fatalf("unstored admin mock deletion = (%+v, %v), want forbidden without deletion", result, err)
	}
}

func TestNotebookDraftTeamMembershipControlsOwnerSubmit(t *testing.T) {
	for _, member := range []bool{false, true} {
		name := "nonmember"
		if member {
			name = "member"
		}
		t.Run(name, func(t *testing.T) {
			store := authorizationDraftStore(3, "draft")
			if member {
				store.memberships[3] = map[int]bool{authAuthorID: true}
			}
			app := newTestApplication(store, nil)
			user := &data.User{ID: authAuthorID, Role: "member"}
			response := draftRequest(app, app.notebookSubmitForApproval, user, url.Values{"message": {"Ready"}})
			if member {
				if response.Code != http.StatusSeeOther || store.submitDraftCalls != 1 || store.revisions[authNotebookID][0].Status != "submitted" {
					t.Fatalf("member submission returned %d, calls %d, status %s", response.Code, store.submitDraftCalls, store.revisions[authNotebookID][0].Status)
				}
			} else if response.Code != http.StatusForbidden || store.submitDraftCalls != 0 || store.auditInsertCalls != 0 || store.revisions[authNotebookID][0].Status != "draft" {
				t.Fatalf("nonmember submission returned %d, calls %d, status %s", response.Code, store.submitDraftCalls, store.revisions[authNotebookID][0].Status)
			}
		})
	}
}
