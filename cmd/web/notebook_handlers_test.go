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

func TestNotebookCreateSubmitRejectsMissingTitle(t *testing.T) {
	store := newFakeStore()
	app := newTestApplication(store, map[string]*template.Template{
		"notebook-create.page.tmpl": parseTemplate("notebook-create.page.tmpl", "invalid form"),
	})

	form := url.Values{
		"title": {""},
		"body":  {"some body"},
	}
	req := httptest.NewRequest(http.MethodPost, "/notebooks/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := serveWithSession(app, app.notebookCreateSubmit, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d; got %d", http.StatusUnprocessableEntity, rr.Code)
	}
	if store.createDraftCalled {
		t.Fatal("CreateDraft should not be called for an invalid form")
	}
}

func TestNotebookCreateSubmitCreatesDraftAndRevision(t *testing.T) {
	store := newFakeStore()
	app := newTestApplication(store, nil)
	user := &data.User{ID: 7, Email: "author@example.com"}

	form := url.Values{
		"title": {"My first notebook"},
		"body":  {"Hello from tests"},
	}
	req := httptest.NewRequest(http.MethodPost, "/notebooks/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(contextSetUser(req.Context(), user))

	rr := serveWithSession(app, app.notebookCreateSubmit, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d; got %d", http.StatusSeeOther, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/notebooks/1" {
		t.Fatalf("expected redirect to /notebooks/1; got %q", got)
	}
	if !store.createDraftCalled {
		t.Fatal("CreateDraft should be called")
	}
	if !store.createRevisionCalled {
		t.Fatal("CreateNotebookRevision should be called")
	}
	if store.createDraftParams.AuthorID != user.ID {
		t.Fatalf("expected draft author %d; got %d", user.ID, store.createDraftParams.AuthorID)
	}
	if store.createRevisionParams.AuthorID != user.ID {
		t.Fatalf("expected revision author %d; got %d", user.ID, store.createRevisionParams.AuthorID)
	}
}

func TestNotebookViewRejectsInvalidID(t *testing.T) {
	app := newTestApplication(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/notebooks/abc", nil)
	req = addRouteParam(req, "id", "abc")

	rr := serveWithSession(app, app.notebookView, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d; got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestNotebookViewRendersRevisionAndTags(t *testing.T) {
	store := newFakeStore()
	store.revisions[1] = []data.NotebookRevision{{
		ID:         10,
		NotebookID: 1,
		AuthorID:   7,
		Title:      "Latest Title",
		Body:       "Latest Body",
		Status:     "draft",
	}}
	store.tags[1] = []data.Tag{
		{ID: 1, Name: "go"},
		{ID: 2, Name: "backend"},
	}

	app := newTestApplication(store, map[string]*template.Template{
		"notebook-view.page.tmpl": parseTemplate("notebook-view.page.tmpl", "{{.CurrentRevision.Title}}|{{len .Tags}}"),
	})
	req := httptest.NewRequest(http.MethodGet, "/notebooks/1", nil)
	req = addRouteParam(req, "id", "1")

	rr := serveWithSession(app, app.notebookView, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Body.String(); got != "Latest Title|2" {
		t.Fatalf("expected latest revision and tag count; got %q", got)
	}
}

func TestNotebookCreateSubmitRejectsTitleTooLong(t *testing.T) {
	store := newFakeStore()
	app := newTestApplication(store, map[string]*template.Template{
		"notebook-create.page.tmpl": parseTemplate("notebook-create.page.tmpl", "error"),
	})
	user := &data.User{ID: 7, Email: "author@example.com"}

	longTitle := strings.Repeat("a", 201)
	form := url.Values{
		"title": {longTitle},
		"body":  {"valid body"},
	}
	req := httptest.NewRequest(http.MethodPost, "/notebooks/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(contextSetUser(req.Context(), user))

	rr := serveWithSession(app, app.notebookCreateSubmit, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d; got %d", http.StatusUnprocessableEntity, rr.Code)
	}
	if store.createDraftCalled {
		t.Fatal("CreateDraft should not be called when title exceeds max length")
	}
}

func TestNotebookCreateSubmitRejectsBodyTooLong(t *testing.T) {
	store := newFakeStore()
	app := newTestApplication(store, map[string]*template.Template{
		"notebook-create.page.tmpl": parseTemplate("notebook-create.page.tmpl", "error"),
	})
	user := &data.User{ID: 7, Email: "author@example.com"}

	longBody := strings.Repeat("x", 50001)
	form := url.Values{
		"title": {"Valid Title"},
		"body":  {longBody},
	}
	req := httptest.NewRequest(http.MethodPost, "/notebooks/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(contextSetUser(req.Context(), user))

	rr := serveWithSession(app, app.notebookCreateSubmit, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d; got %d", http.StatusUnprocessableEntity, rr.Code)
	}
	if store.createDraftCalled {
		t.Fatal("CreateDraft should not be called when body exceeds max length")
	}
}

func TestNotebookViewRejectsZeroID(t *testing.T) {
	app := newTestApplication(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/notebooks/0", nil)
	req = addRouteParam(req, "id", "0")

	rr := serveWithSession(app, app.notebookView, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d; got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestNotebookViewRejectsNegativeID(t *testing.T) {
	app := newTestApplication(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/notebooks/-1", nil)
	req = addRouteParam(req, "id", "-1")

	rr := serveWithSession(app, app.notebookView, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d; got %d", http.StatusBadRequest, rr.Code)
	}
}
