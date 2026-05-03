package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
)

func TestLoginForm(t *testing.T) {
	app := newTestApplication(nil, map[string]*template.Template{
		"login.tmpl": parseTemplate("login.tmpl", `<form action="/login" method="post"></form>`),
	})

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := serveWithSession(app, app.loginForm, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `action="/login"`) {
		t.Fatalf("expected login form; got %q", rr.Body.String())
	}
}

func TestLoginSubmitRejectsBadCredentials(t *testing.T) {
	app := newTestApplication(newFakeStore(), nil)
	form := url.Values{
		"email":    {"missing@example.com"},
		"password": {"wrong-password"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := serveWithSession(app, app.loginSubmit, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d; got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("expected redirect to /login; got %q", got)
	}
}

func TestLoginSubmitAcceptsValidCredentials(t *testing.T) {
	store := newFakeStore()
	store.addUser(data.User{ID: 1, Email: "test@example.com"})
	app := newTestApplication(store, nil)

	form := url.Values{
		"email":    {"test@example.com"},
		"password": {"test123"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := serveWithSession(app, app.loginSubmit, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d; got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard; got %q", got)
	}
}

func TestLogoutRedirectsHome(t *testing.T) {
	app := newTestApplication(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)

	rr := serveWithSession(app, app.logout, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d; got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/" {
		t.Fatalf("expected redirect to /; got %q", got)
	}
}
