package main

import (
	"context"
	"errors"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/SShogun/ControlPlane/internal/data"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
)

var errTestNotFound = errors.New("test record not found")

type fakeStore struct {
	usersByID    map[int]data.User
	usersByEmail map[string]data.User
	notebooks    map[int]data.Notebook
	revisions    map[int][]data.NotebookRevision
	tags         map[int][]data.Tag
	submitted    []data.NotebookRevision
	auditEvents  []data.AuditEvent
	flags        []data.ModerationFlag

	createDraftCalled    bool
	createRevisionCalled bool
	createDraftParams    data.CreateDraftParams
	createRevisionParams data.CreateNotebookRevisionParams

	approveCalled bool
	approvedRevID int
	approvedDocID int
	approvedUser  int

	updateStatusCalled bool
	updateStatusParams data.UpdateRevisionStatusParams

	rejectCalled bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		usersByID:    map[int]data.User{},
		usersByEmail: map[string]data.User{},
		notebooks:    map[int]data.Notebook{},
		revisions:    map[int][]data.NotebookRevision{},
		tags:         map[int][]data.Tag{},
	}
}

func (s *fakeStore) addUser(user data.User) {
	s.usersByID[user.ID] = user
	s.usersByEmail[user.Email] = user
}

func (s *fakeStore) GetUserByEmail(ctx context.Context, email string) (data.User, error) {
	user, ok := s.usersByEmail[email]
	if !ok {
		return data.User{}, errTestNotFound
	}
	return user, nil
}

func (s *fakeStore) GetUser(ctx context.Context, id int) (data.User, error) {
	user, ok := s.usersByID[id]
	if !ok {
		return data.User{}, errTestNotFound
	}
	return user, nil
}

func (s *fakeStore) CheckPassword(user data.User, password string) bool {
	return user.ID != 0 && password == "test123"
}

func (s *fakeStore) ListNotebooks(ctx context.Context) ([]data.Notebook, error) {
	notebooks := make([]data.Notebook, 0, len(s.notebooks))
	for _, notebook := range s.notebooks {
		notebooks = append(notebooks, notebook)
	}
	return notebooks, nil
}

func (s *fakeStore) SearchNotebooks(ctx context.Context, query string) ([]data.Notebook, error) {
	return s.ListNotebooks(ctx)
}

func (s *fakeStore) NotebookView(ctx context.Context, id int) ([]data.Notebook, error) {
	notebook, ok := s.notebooks[id]
	if !ok {
		return nil, errTestNotFound
	}
	return []data.Notebook{notebook}, nil
}

func (s *fakeStore) CreateDraft(ctx context.Context, params data.CreateDraftParams) (int, error) {
	s.createDraftCalled = true
	s.createDraftParams = params
	return 1, nil
}

func (s *fakeStore) UpdateDraft(ctx context.Context, params data.UpdateDraftParams) (int, error) {
	s.createRevisionCalled = true
	s.createRevisionParams = data.CreateNotebookRevisionParams{
		DocumentID: params.NotebookID,
		AuthorID:   params.AuthorID,
		Title:      params.Title,
		Body:       params.Body,
		Status:     "draft",
	}
	return 2, nil
}

func (s *fakeStore) ListRecentDrafts(ctx context.Context, authorID int) ([]data.NotebookRevision, error) {
	var drafts []data.NotebookRevision
	for _, revisions := range s.revisions {
		for _, revision := range revisions {
			if revision.AuthorID == authorID && revision.Status == "draft" {
				drafts = append(drafts, revision)
			}
		}
	}
	return drafts, nil
}

func (s *fakeStore) ListAuditEvents(ctx context.Context) ([]data.AuditEvent, error) {
	return s.auditEvents, nil
}

func (s *fakeStore) ListTeams(ctx context.Context) ([]data.Team, error) {
	return nil, nil
}

func (s *fakeStore) GetTeam(ctx context.Context, id int) (data.Team, error) {
	return data.Team{}, errTestNotFound
}

func (s *fakeStore) AddMembership(ctx context.Context, userID, teamID int, role string) error {
	return nil
}

func (s *fakeStore) CreateNotebookRevision(ctx context.Context, params data.CreateNotebookRevisionParams) (int, error) {
	s.createRevisionCalled = true
	s.createRevisionParams = params
	return 1, nil
}

func (s *fakeStore) ListNotebookRevisions(ctx context.Context, notebookID int) ([]data.NotebookRevision, error) {
	return s.revisions[notebookID], nil
}

func (s *fakeStore) CreateTag(ctx context.Context, name string) (int, error) {
	return 1, nil
}

func (s *fakeStore) AttachTag(ctx context.Context, notebookID, tagID int) error {
	return nil
}

func (s *fakeStore) ListNotebookTags(ctx context.Context, notebookID int) ([]data.Tag, error) {
	return s.tags[notebookID], nil
}

func (s *fakeStore) InsertAuditLog(ctx context.Context, params data.InsertAuditLogParams) error {
	return nil
}

func (s *fakeStore) ApproveRevisionTx(ctx context.Context, revisionID, notebookID, reviewerID int) error {
	s.approveCalled = true
	s.approvedRevID = revisionID
	s.approvedDocID = notebookID
	s.approvedUser = reviewerID
	return nil
}

func (s *fakeStore) RejectRevisionTx(ctx context.Context, revisionID, reviewerID int) error {
	s.rejectCalled = true
	s.updateStatusCalled = true
	s.updateStatusParams = data.UpdateRevisionStatusParams{
		ID:     revisionID,
		Status: "rejected",
	}
	return nil
}

func (s *fakeStore) ListSubmittedRevisions(ctx context.Context) ([]data.NotebookRevision, error) {
	return s.submitted, nil
}

func (s *fakeStore) UpdateRevisionStatus(ctx context.Context, params data.UpdateRevisionStatusParams) error {
	s.updateStatusCalled = true
	s.updateStatusParams = params
	return nil
}

func (s *fakeStore) CreateModerationFlag(ctx context.Context, params data.CreateModerationFlagParams) (int, error) {
	return 1, nil
}

func (s *fakeStore) ListModerationFlags(ctx context.Context) ([]data.ModerationFlag, error) {
	return s.flags, nil
}

func (s *fakeStore) ResolveModerationFlag(ctx context.Context, id int) error {
	return nil
}

func newTestApplication(store *fakeStore, templates map[string]*template.Template) *Application {
	if store == nil {
		store = newFakeStore()
	}
	if templates == nil {
		templates = map[string]*template.Template{}
	}

	sessionManager := scs.New()
	sessionManager.Cookie.Secure = false

	return &Application{
		store:          store,
		sessionManager: sessionManager,
		templateCache:  templates,
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func parseTemplate(name, body string) *template.Template {
	return template.Must(template.New(name).Parse(`{{define "base"}}` + body + `{{end}}`))
}

func addRouteParam(r *http.Request, key, value string) *http.Request {
	rctx, ok := r.Context().Value(chi.RouteCtxKey).(*chi.Context)
	if !ok {
		rctx = chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	}
	rctx.URLParams.Add(key, value)
	return r
}

func serveWithSession(app *Application, handler http.HandlerFunc, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	app.sessionManager.LoadAndSave(handler).ServeHTTP(rr, r)
	return rr
}
