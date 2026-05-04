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

func TestRateLimitLogin(t *testing.T) {
	app := newTestApplication(nil, nil)

	// A dummy handler that represents our loginSubmit route
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap our dummy handler in the rate limiter
	handler := app.rateLimitLogin(next)

	// We configured the limiter to allow a burst of 3 requests.
	// So, the first 3 rapid-fire POST requests from the same IP should succeed.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "192.168.1.1:12345" // Simulate the same user IP
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected request %d to be allowed (200 OK), got %d", i+1, rr.Code)
		}
	}

	// The 4th request in the same second should be blocked!
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 4th request to be blocked (429 Too Many Requests), got %d", rr.Code)
	}
}
