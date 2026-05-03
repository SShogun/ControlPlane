package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
)

func TestRequireAuthenticationRedirectsAnonymousUser(t *testing.T) {
	app := newTestApplication(nil, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)

	app.requireAuthentication(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d; got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("expected redirect to /login; got %q", got)
	}
}

func TestRequireAuthenticationAllowsCurrentUser(t *testing.T) {
	app := newTestApplication(nil, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	user := &data.User{ID: 1, Email: "test@example.com"}
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req = req.WithContext(contextSetUser(context.Background(), user))
	rr := httptest.NewRecorder()

	app.requireAuthentication(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, rr.Code)
	}
}

func TestRequireRoleRedirectsWrongRole(t *testing.T) {
	app := newTestApplication(nil, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	user := &data.User{ID: 1, Email: "member@example.com", Role: "member"}
	req := httptest.NewRequest(http.MethodGet, "/approvals", nil)
	req = req.WithContext(contextSetUser(req.Context(), user))

	rr := serveWithSession(app, app.requireRole("reviewer")(next).ServeHTTP, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d; got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard; got %q", got)
	}
}
