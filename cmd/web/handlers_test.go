package main

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
)

//! yes, 90% of this file is AI generated because i dont like to write my own test files

/*
for my understanding purposes
we make a fake store for testing our database
and a fake sessionManager for our sessions

we then rewrite the exact same code for our handlers but with the fake store & sessionManager
to simulate operations
*/

// mockStore is a fake implementation of data.UserStore for testing.
type mockStore struct {
	users      map[int]*data.User
	notebooks  map[int]*data.Notebook
	revisions  map[int][]data.NotebookRevision
	tags       map[int][]data.Tag
	nextUserID int

	createDraftCalled            bool
	createNotebookRevisionCalled bool
	lastCreateDraftParams        data.CreateDraftParams
	lastCreateRevisionParams     data.CreateNotebookRevisionParams

	approveRevisionTxCalled bool
	lastApprovedRevisionID  int
	lastApprovedNotebookID  int
	lastApprovedReviewerID  int
}

func newMockStore() *mockStore {
	return &mockStore{
		users:      make(map[int]*data.User),
		notebooks:  make(map[int]*data.Notebook),
		revisions:  make(map[int][]data.NotebookRevision),
		tags:       make(map[int][]data.Tag),
		nextUserID: 1,
	}
}

func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (data.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return *u, nil
		}
	}
	return data.User{}, nil
}

func (m *mockStore) GetUser(ctx context.Context, id int) (data.User, error) {
	u, ok := m.users[id]
	if !ok {
		return data.User{}, nil
	}
	return *u, nil
}

func (m *mockStore) CheckPassword(user data.User, password string) bool {

	return password == "test123"
}

func (m *mockStore) CreateUser(ctx context.Context, params data.CreateUserParams) (int, error) {
	for _, user := range m.users {
		if user.Email == params.Email {
			return user.ID, nil
		}
	}
	userID := m.nextUserID
	m.nextUserID++
	m.users[userID] = &data.User{ID: userID, Email: params.Email, PasswordHash: params.PasswordHash, Role: "member"}
	return userID, nil
}

func (m *mockStore) ListNotebooks(ctx context.Context) ([]data.Notebook, error) {
	var result []data.Notebook
	for _, nb := range m.notebooks {
		result = append(result, *nb)
	}
	return result, nil
}

func (m *mockStore) SearchNotebooks(ctx context.Context, query string) ([]data.Notebook, error) {
	return m.ListNotebooks(ctx)
}

func (m *mockStore) NotebookView(ctx context.Context, id int) ([]data.Notebook, error) {
	nb, ok := m.notebooks[id]
	if !ok {
		return nil, nil
	}
	return []data.Notebook{*nb}, nil
}

func (m *mockStore) CreateDraft(ctx context.Context, params data.CreateDraftParams) (int, error) {
	m.createDraftCalled = true
	m.lastCreateDraftParams = params
	return 1, nil
}

func (m *mockStore) UpdateDraft(ctx context.Context, params data.UpdateDraftParams) (int, error) {
	m.createNotebookRevisionCalled = true
	m.lastCreateRevisionParams = data.CreateNotebookRevisionParams{
		DocumentID: params.NotebookID,
		AuthorID:   params.AuthorID,
		Title:      params.Title,
		Body:       params.Body,
		Status:     "draft",
	}
	return 2, nil
}

func (m *mockStore) ListRecentDrafts(ctx context.Context, authorID int) ([]data.NotebookRevision, error) {
	var result []data.NotebookRevision
	for _, revisions := range m.revisions {
		for _, revision := range revisions {
			if revision.AuthorID == authorID && revision.Status == "draft" {
				result = append(result, revision)
			}
		}
	}
	return result, nil
}

func (m *mockStore) ListAuditEvents(ctx context.Context) ([]data.AuditEvent, error) {
	return nil, nil
}

func (m *mockStore) ListTeams(ctx context.Context) ([]data.Team, error) {
	return []data.Team{}, nil
}

func (m *mockStore) GetTeam(ctx context.Context, id int) (data.Team, error) {
	return data.Team{}, nil
}

func (m *mockStore) AddMembership(ctx context.Context, userID, teamID int, role string) error {
	return nil
}

func (m *mockStore) DeleteNotebook(ctx context.Context, notebookID int) error {
	delete(m.notebooks, notebookID)
	delete(m.revisions, notebookID)
	return nil
}

func (m *mockStore) DeleteNotebookRevision(ctx context.Context, revisionID int) error {
	for notebookID, revisions := range m.revisions {
		filtered := make([]data.NotebookRevision, 0, len(revisions))
		for _, revision := range revisions {
			if revision.ID != revisionID {
				filtered = append(filtered, revision)
			}
		}
		m.revisions[notebookID] = filtered
	}
	return nil
}

func (m *mockStore) CreateNotebookRevision(ctx context.Context, params data.CreateNotebookRevisionParams) (int, error) {
	m.createNotebookRevisionCalled = true
	m.lastCreateRevisionParams = params
	return 1, nil
}

func (m *mockStore) ListNotebookRevisions(ctx context.Context, notebookID int) ([]data.NotebookRevision, error) {
	if items, ok := m.revisions[notebookID]; ok {
		return items, nil
	}
	return []data.NotebookRevision{}, nil
}

func (m *mockStore) CreateTag(ctx context.Context, name string) (int, error) {
	return 1, nil
}

func (m *mockStore) AttachTag(ctx context.Context, notebookID, tagID int) error {
	return nil
}

func (m *mockStore) ListNotebookTags(ctx context.Context, notebookID int) ([]data.Tag, error) {
	if items, ok := m.tags[notebookID]; ok {
		return items, nil
	}
	return []data.Tag{}, nil
}

func (m *mockStore) InsertAuditLog(ctx context.Context, params data.InsertAuditLogParams) error {
	return nil
}

func (m *mockStore) ApproveRevisionTx(ctx context.Context, revisionID, notebookID, reviewerID int, note string) error {
	m.approveRevisionTxCalled = true
	m.lastApprovedRevisionID = revisionID
	m.lastApprovedNotebookID = notebookID
	m.lastApprovedReviewerID = reviewerID
	return nil
}

func (m *mockStore) RejectRevisionTx(ctx context.Context, revisionID, reviewerID int, note string) error {
	return m.UpdateRevisionStatus(ctx, data.UpdateRevisionStatusParams{ID: revisionID, Status: "rejected"})
}

func (m *mockStore) ListSubmittedRevisions(ctx context.Context) ([]data.NotebookRevision, error) {
	return []data.NotebookRevision{}, nil
}

func (m *mockStore) UpdateRevisionStatus(ctx context.Context, params data.UpdateRevisionStatusParams) error {
	return nil
}

func (m *mockStore) CreateModerationFlag(ctx context.Context, params data.CreateModerationFlagParams) (int, error) {
	return 1, nil
}

func (m *mockStore) ListModerationFlags(ctx context.Context) ([]data.ModerationFlag, error) {
	return nil, nil
}

func (m *mockStore) ResolveModerationFlag(ctx context.Context, id int) error {
	return nil
}

// Tests

// unauthenticated requests redirect to /login.
func TestRequireAuthenticationMiddleware(t *testing.T) {
	// Create a simple test handler that would return 200 if it reached it
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	app := &Application{
		store: newMockStore(),
	}

	handler := app.requireAuthentication(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("want status %d; got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/login" {
		t.Errorf("want redirect to /login; got %s", location)
	}
}

// verifies authenticated requests proceed.
func TestRequireAuthenticationMiddlewareWithUser(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	app := &Application{
		store: newMockStore(),
	}

	handler := app.requireAuthentication(innerHandler)

	user := &data.User{ID: 1, Email: "test@example.com"}
	ctx := contextSetUser(context.Background(), user)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, rr.Code)
	}
}

func TestLoginSubmitBadCredentials(t *testing.T) {
	store := newMockStore()
	sessionManager := scs.New()

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
	}

	form := url.Values{}
	form.Set("email", "missing@example.com")
	form.Set("password", "wrong-password")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	// Use LoadAndSave so session calls (Put/RenewToken) have valid request context.
	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.loginSubmit))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("want status %d; got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/login" {
		t.Errorf("want redirect to /login; got %s", location)
	}
}

func TestLoginSubmitValidCredentials(t *testing.T) {
	store := newMockStore()
	store.users[1] = &data.User{ID: 1, Email: "test@example.com"}
	sessionManager := scs.New()

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
	}

	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("password", "test123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.loginSubmit))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("want status %d; got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/dashboard" {
		t.Errorf("want redirect to /dashboard; got %s", location)
	}
}

func TestLoginFormGET(t *testing.T) {
	store := newMockStore()
	sessionManager := scs.New()

	cache := map[string]*template.Template{
		"login.page.tmpl": parseTemplate("login.page.tmpl", "<form action=\"/login\" method=\"post\"></form>"),
	}

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
		templateCache:  cache,
	}

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.loginForm))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "<form action=\"/login\"") {
		t.Errorf("want login form in body")
	}
}

func TestLogoutSubmit(t *testing.T) {
	store := newMockStore()
	sessionManager := scs.New()

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
	}

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rr := httptest.NewRecorder()

	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.logout))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("want status %d; got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("want redirect to /; got %s", location)
	}
}

func TestNotebookCreateSubmitInvalidForm(t *testing.T) {
	store := newMockStore()
	sessionManager := scs.New()

	cache := map[string]*template.Template{
		"notebook-create.page.tmpl": parseTemplate("notebook-create.page.tmpl", "invalid form"),
	}

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
		templateCache:  cache,
	}

	form := url.Values{}
	form.Set("title", "")
	form.Set("body", "some body")

	req := httptest.NewRequest(http.MethodPost, "/notebooks/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.notebookCreateSubmit))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("want status %d; got %d", http.StatusUnprocessableEntity, rr.Code)
	}

	if store.createDraftCalled {
		t.Errorf("CreateDraft should not be called for invalid form")
	}
}

func TestNotebookCreateSubmitValidForm(t *testing.T) {
	store := newMockStore()
	sessionManager := scs.New()

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
	}

	user := &data.User{ID: 7, Email: "author@example.com"}

	form := url.Values{}
	form.Set("title", "My first notebook")
	form.Set("slug", "my-first-notebook")
	form.Set("visibility", "private")
	form.Set("body", "Hello from tests")

	baseReq := httptest.NewRequest(http.MethodPost, "/notebooks/new", strings.NewReader(form.Encode()))
	baseReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := baseReq.WithContext(contextSetUser(baseReq.Context(), user))
	rr := httptest.NewRecorder()

	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.notebookCreateSubmit))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("want status %d; got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/notebooks/1" {
		t.Errorf("want redirect to /notebooks/1; got %s", location)
	}

	if !store.createDraftCalled {
		t.Errorf("CreateDraft should be called")
	}

	if !store.createNotebookRevisionCalled {
		t.Errorf("CreateNotebookRevision should be called")
	}

	if store.lastCreateDraftParams.AuthorID != user.ID {
		t.Errorf("want draft AuthorID %d; got %d", user.ID, store.lastCreateDraftParams.AuthorID)
	}

	if store.lastCreateRevisionParams.AuthorID != user.ID {
		t.Errorf("want revision AuthorID %d; got %d", user.ID, store.lastCreateRevisionParams.AuthorID)
	}
}

func TestNotebookViewInvalidID(t *testing.T) {
	store := newMockStore()
	sessionManager := scs.New()

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
	}

	req := httptest.NewRequest(http.MethodGet, "/notebooks/abc", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.notebookView))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want status %d; got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestNotebookViewValidID(t *testing.T) {
	store := newMockStore()
	store.revisions[1] = []data.NotebookRevision{{
		ID:         10,
		NotebookID: 1,
		AuthorID:   7,
		Title:      "Latest Title",
		Body:       "Latest Body",
		Status:     "draft",
	}}
	store.tags[1] = []data.Tag{{ID: 1, Name: "go"}, {ID: 2, Name: "backend"}}

	sessionManager := scs.New()
	cache := map[string]*template.Template{
		"notebook-view.page.tmpl": parseTemplate("notebook-view.page.tmpl", "{{if .CurrentRevision}}{{.CurrentRevision.Title}}{{end}}|{{len .Tags}}"),
	}

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
		templateCache:  cache,
	}

	req := httptest.NewRequest(http.MethodGet, "/notebooks/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.notebookView))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Latest Title") {
		t.Errorf("want body to contain latest revision title; got %q", body)
	}
	if !strings.Contains(body, "|2") {
		t.Errorf("want body to contain tag count 2; got %q", body)
	}
}

func TestApproveRevisionSubmit(t *testing.T) {
	store := newMockStore()
	sessionManager := scs.New()

	app := &Application{
		store:          store,
		sessionManager: sessionManager,
	}

	reviewer := &data.User{ID: 42, Email: "reviewer@example.com"}

	req := httptest.NewRequest(http.MethodPost, "/approvals/9/revisions/3/approve", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("docID", "9")
	rctx.URLParams.Add("revID", "3")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(contextSetUser(req.Context(), reviewer))
	rr := httptest.NewRecorder()

	handler := sessionManager.LoadAndSave(http.HandlerFunc(app.approveRevisionSubmit))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("want status %d; got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/notebooks/9" {
		t.Errorf("want redirect to /notebooks/9; got %s", location)
	}

	if !store.approveRevisionTxCalled {
		t.Errorf("ApproveRevisionTx should be called")
	}

	if store.lastApprovedRevisionID != 3 || store.lastApprovedNotebookID != 9 || store.lastApprovedReviewerID != 42 {
		t.Errorf("unexpected approval params got rev=%d notebook=%d reviewer=%d", store.lastApprovedRevisionID, store.lastApprovedNotebookID, store.lastApprovedReviewerID)
	}
}
