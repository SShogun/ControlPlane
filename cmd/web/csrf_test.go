package web

import (
	"html"
	"html/template"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
)

func newCSRFTestApp(store *fakeStore) (*Application, http.Handler) {
	if store == nil {
		store = newFakeStore()
	}

	templates := map[string]*template.Template{
		"home.page.tmpl":             parseTemplate("home.page.tmpl", "home"),
		"login.tmpl":                 parseTemplate("login.tmpl", `<form>{{.CSRFToken}}</form>`),
		"dashboard.page.tmpl":        parseTemplate("dashboard.page.tmpl", "dashboard"),
		"notebook-create.page.tmpl":  parseTemplate("notebook-create.page.tmpl", "create"),
		"notebooks.page.tmpl":        parseTemplate("notebooks.page.tmpl", "list"),
		"notebook-view.page.tmpl":    parseTemplate("notebook-view.page.tmpl", "view"),
		"notebook-edit.page.tmpl":    parseTemplate("notebook-edit.page.tmpl", "edit"),
		"notebooks-search.page.tmpl": parseTemplate("notebooks-search.page.tmpl", "search"),
		"approval-queue.page.tmpl":   parseTemplate("approval-queue.page.tmpl", "approvals"),
		"admin-audit.page.tmpl":      parseTemplate("admin-audit.page.tmpl", "audit"),
	}

	app := newTestApplication(store, templates)
	app.config.CSRFSecret = []byte("test-csrf-secret-exactly-32bytes")
	app.config.SecureCookies = false

	return app, app.routes()
}

func closeResponseBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp == nil || resp.Body == nil {
		return
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("failed to close response body: %v", err)
	}
}

func TestCSRFRejectsPostWithoutToken(t *testing.T) {
	_, handler := newCSRFTestApp(nil)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		// Don't follow redirects — we want to inspect the response.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// GET /login to establish CSRF cookie.
	loginResp, err := client.Get(ts.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	if err := loginResp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	// POST without the CSRF token field.
	resp, err := client.PostForm(ts.URL+"/login", url.Values{
		"email":    {"test@example.com"},
		"password": {"test123"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d for POST without CSRF token; got %d", http.StatusForbidden, resp.StatusCode)
	}
}

func TestCSRFAcceptsPostWithValidToken(t *testing.T) {
	store := newFakeStore()
	store.addUser(data.User{ID: 1, Email: "test@example.com"})

	_, handler := newCSRFTestApp(store)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// GET /login to obtain CSRF token from the rendered body.
	getResp, err := client.Get(ts.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	defer closeResponseBody(t, getResp)

	bodyBytes, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)

	token := strings.TrimPrefix(body, "<form>")
	token = strings.TrimSuffix(token, "</form>")
	token = html.UnescapeString(token)

	if token == "" {
		t.Fatal("CSRF token was empty from GET /login")
	}

	// POST with the valid CSRF token.
	form := url.Values{
		"email":      {"test@example.com"},
		"password":   {"test123"},
		"csrf_token": {token},
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", ts.URL+"/login")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer closeResponseBody(t, resp)

	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	respBody := string(bodyBytes)

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected status %d for POST with valid CSRF token; got %d (body: %s)", http.StatusSeeOther, resp.StatusCode, respBody)
	}
	if got := resp.Header.Get("Location"); got != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard; got %q", got)
	}
}

func TestCSRFTokenRenderedInLoginForm(t *testing.T) {
	_, handler := newCSRFTestApp(nil)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	defer closeResponseBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)

	if body == "<form></form>" {
		t.Fatal("CSRF token was empty in rendered template")
	}
	if !strings.Contains(body, "<form>") {
		t.Fatalf("unexpected body: %q", body)
	}
}
