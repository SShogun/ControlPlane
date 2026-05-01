package data

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int
	Email        string
	PasswordHash []byte
}

type Team struct {
	ID        int
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Membership struct {
	UserID    int
	TeamID    int
	Role      string
	CreatedAt time.Time
}

type NotebookRevision struct {
	ID         int
	NotebookID int
	AuthorID   int
	Title      string
	Body       string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Tag struct {
	ID        int
	Name      string
	CreatedAt time.Time
}

type NotebookTag struct {
	NotebookID int
	TagID      int
}

type UserStore interface {
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUser(ctx context.Context, id int) (User, error)
	CheckPassword(user User, password string) bool
	ListNotebooks(ctx context.Context) ([]Notebook, error)
	NotebookView(ctx context.Context, id int) ([]Notebook, error)
	CreateDraft(ctx context.Context, params CreateDraftParams) (int, error)
	// 2nd set of methods
	ListTeams(ctx context.Context) ([]Team, error)
	GetTeam(ctx context.Context, id int) (Team, error)
	AddMembership(ctx context.Context, userID, teamID int, role string) error
	CreateNotebookRevision(ctx context.Context, params CreateNotebookRevisionParams) (int, error)
	ListNotebookRevisions(ctx context.Context, notebookID int) ([]NotebookRevision, error)
	CreateTag(ctx context.Context, name string) (int, error)
	AttachTag(ctx context.Context, notebookID, tagID int) error
	ListNotebookTags(ctx context.Context, notebookID int) ([]Tag, error)
	// Audit logging
	InsertAuditLog(ctx context.Context, params InsertAuditLogParams) error
}

type Notebook struct {
	ID                         int
	TeamID                     int
	Title                      string
	Content                    string
	Slug                       string
	Visibility                 string
	IsPublished                bool
	CurrentPublishedRevisionID int
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type CreateNotebookRevisionParams struct {
	DocumentID int
	AuthorID   int
	Title      string
	Body       string
	Status     string
}

type CreateDraftParams struct {
	AuthorID int
	Title    string
	Body     string
}

type InsertAuditLogParams struct {
	UserID     int
	Action     string
	EntityType string
	EntityID   int
}

type SessionManager interface {
	RenewToken(ctx context.Context) error
	Put(ctx context.Context, key string, value interface{})
}

type Application struct {
	store          UserStore
	sessionManager SessionManager
}

func (app *Application) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.PostForm.Get("email")
	plaintextPassword := r.PostForm.Get("password")

	user, err := app.store.GetUserByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(plaintextPassword))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	app.sessionManager.Put(r.Context(), "userID", user.ID)
	app.sessionManager.Put(r.Context(), "flash", "Welcome back!")

	http.Redirect(w, r, "/notebook/list", http.StatusSeeOther)
}

func CheckPassword(user User, password string) bool {
	return bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)) == nil
}
