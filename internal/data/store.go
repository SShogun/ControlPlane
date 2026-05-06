package data

import (
	"context"
	"time"
)

type User struct {
	ID           int
	Email        string
	PasswordHash []byte
	Role         string
}
type CreateUserParams struct {
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
	ReviewNote string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Tag struct {
	ID        int
	Name      string
	CreatedAt time.Time
}

type AuditEvent struct {
	ID         int
	ActorID    int
	EventType  string
	EntityType string
	EntityID   int
	CreatedAt  time.Time
	ActorEmail string
	Details    string
}

type ModerationFlag struct {
	ID          int
	NotebookID  int
	ModeratorID int
	Reason      string
	ResolvedAt  *time.Time
	CreatedAt   time.Time
}

type NotebookTag struct {
	NotebookID int
	TagID      int
}

type UserStore interface {
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUser(ctx context.Context, id int) (User, error)
	CheckPassword(user User, password string) bool
	CreateUser(ctx context.Context, params CreateUserParams) (int, error)
	ListNotebooks(ctx context.Context) ([]Notebook, error)
	SearchNotebooks(ctx context.Context, query string) ([]Notebook, error)
	NotebookView(ctx context.Context, id int) ([]Notebook, error)
	CreateDraft(ctx context.Context, params CreateDraftParams) (int, error)
	UpdateDraft(ctx context.Context, params UpdateDraftParams) (int, error)
	ListRecentDrafts(ctx context.Context, authorID int) ([]NotebookRevision, error)
	ListAuditEvents(ctx context.Context) ([]AuditEvent, error)
	DeleteNotebook(ctx context.Context, notebookID int) error
	DeleteNotebookRevision(ctx context.Context, revisionID int) error
	ListTeams(ctx context.Context) ([]Team, error)
	GetTeam(ctx context.Context, id int) (Team, error)
	AddMembership(ctx context.Context, userID, teamID int, role string) error
	CreateNotebookRevision(ctx context.Context, params CreateNotebookRevisionParams) (int, error)
	ListNotebookRevisions(ctx context.Context, notebookID int) ([]NotebookRevision, error)
	CreateTag(ctx context.Context, name string) (int, error)
	AttachTag(ctx context.Context, notebookID, tagID int) error
	ListNotebookTags(ctx context.Context, notebookID int) ([]Tag, error)
	InsertAuditLog(ctx context.Context, params InsertAuditLogParams) error
	ApproveRevisionTx(ctx context.Context, revisionID, notebookID, reviewerID int, note string) error
	RejectRevisionTx(ctx context.Context, revisionID, reviewerID int, note string) error
	ListSubmittedRevisions(ctx context.Context) ([]NotebookRevision, error)
	UpdateRevisionStatus(ctx context.Context, params UpdateRevisionStatusParams) error
	CreateModerationFlag(ctx context.Context, params CreateModerationFlagParams) (int, error)
	ListModerationFlags(ctx context.Context) ([]ModerationFlag, error)
	ResolveModerationFlag(ctx context.Context, id int) error
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
	AuthorID   int
	Title      string
	Slug       string
	Visibility string
	Body       string
}

type UpdateDraftParams struct {
	NotebookID int
	AuthorID   int
	Title      string
	Body       string
}

type InsertAuditLogParams struct {
	UserID     int
	Action     string
	EntityType string
	EntityID   int
	Details    string
}

type UpdateRevisionStatusParams struct {
	ID     int
	Status string
}

type CreateModerationFlagParams struct {
	NotebookID  int
	ModeratorID int
	Reason      string
}
