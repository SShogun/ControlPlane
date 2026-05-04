package data

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type PgxStore struct {
	DB *pgxpool.Pool
}

func (s *PgxStore) ListTeams(ctx context.Context) ([]Team, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, name, created_at, updated_at FROM teams ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.ID, &team.Name, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, rows.Err()
}

func (s *PgxStore) GetTeam(ctx context.Context, id int) (Team, error) {
	const query = `
		SELECT id, name, created_at, updated_at
		FROM teams
		WHERE id = $1
	`

	row := s.DB.QueryRow(ctx, query, id)
	var team Team
	if err := row.Scan(&team.ID, &team.Name, &team.CreatedAt, &team.UpdatedAt); err != nil {
		return Team{}, err
	}
	return team, nil

}

func (s *PgxStore) AddMembership(ctx context.Context, userID, teamID int, role string) error {
	const query = `
		INSERT INTO memberships (user_id, team_id, role)
		VALUES ($1, $2, $3)
	`

	_, err := s.DB.Exec(ctx, query, userID, teamID, role)
	if err != nil {
		return err
	}

	return nil
}

func (s *PgxStore) CreateNotebookRevision(ctx context.Context, params CreateNotebookRevisionParams) (int, error) {
	const query = `
		INSERT INTO notebook_revisions (notebook_id, author_id, title, body, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var newID int
	err := s.DB.QueryRow(ctx, query, params.DocumentID, params.AuthorID, params.Title, params.Body, params.Status).Scan(&newID)
	if err != nil {
		return 0, err
	}
	return newID, nil
}

func (s *PgxStore) ListNotebookRevisions(ctx context.Context, notebookID int) ([]NotebookRevision, error) {

	const query = `
		SELECT id, notebook_id, author_id, title, body, status, created_at, updated_at
		FROM notebook_revisions
		WHERE notebook_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(ctx, query, notebookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []NotebookRevision
	for rows.Next() {
		var rev NotebookRevision
		err := rows.Scan(&rev.ID, &rev.NotebookID, &rev.AuthorID, &rev.Title, &rev.Body, &rev.Status, &rev.CreatedAt, &rev.UpdatedAt)
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, rev)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return revisions, err
}

func (s *PgxStore) CreateTag(ctx context.Context, name string) (int, error) {
	query := `
        INSERT INTO tags (name) VALUES ($1) 
        ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name 
        RETURNING id`

	var id int
	err := s.DB.QueryRow(ctx, query, name).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *PgxStore) AttachTag(ctx context.Context, notebookID, tagID int) error {
	query := `
        INSERT INTO document_tags (document_id, tag_id) VALUES ($1, $2) 
        ON CONFLICT (document_id, tag_id) DO NOTHING`

	_, err := s.DB.Exec(ctx, query, notebookID, tagID)
	return err
}

func (s *PgxStore) ListNotebookTags(ctx context.Context, notebookID int) ([]Tag, error) {
	query := `
        SELECT t.id, t.name 
        FROM tags t 
        JOIN document_tags dt ON t.id = dt.tag_id 
        WHERE dt.document_id = $1 
        ORDER BY t.name ASC`

	rows, err := s.DB.Query(ctx, query, notebookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Name)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (s *PgxStore) GetUserByEmail(ctx context.Context, email string) (User, error) {
	const query = `SELECT id, email, password_hash FROM users WHERE email = $1`
	return scanUser(s.DB.QueryRow(ctx, query, email))
}

func (s *PgxStore) GetUser(ctx context.Context, id int) (User, error) {
	const query = `SELECT id, email, password_hash FROM users WHERE id = $1`
	return scanUser(s.DB.QueryRow(ctx, query, id))
}

func (s *PgxStore) CheckPassword(user User, password string) bool {
	return bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)) == nil
}

func (s *PgxStore) ListNotebooks(ctx context.Context) ([]Notebook, error) {
	const query = `
		SELECT id, title, content, is_published, created_at, updated_at
		FROM notebooks
		WHERE is_published = 1
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notebooks := []Notebook{}
	for rows.Next() {
		var notebook Notebook
		if err := rows.Scan(&notebook.ID, &notebook.Title, &notebook.Content, &notebook.IsPublished, &notebook.CreatedAt, &notebook.UpdatedAt); err != nil {
			return nil, err
		}
		notebooks = append(notebooks, notebook)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notebooks, nil
}

func (s *PgxStore) NotebookView(ctx context.Context, id int) ([]Notebook, error) {
	const query = `
		SELECT id, title, content, is_published, created_at, updated_at
		FROM notebooks
		WHERE id = $1
	`

	row := s.DB.QueryRow(ctx, query, id)
	var notebook Notebook
	if err := row.Scan(&notebook.ID, &notebook.Title, &notebook.Content, &notebook.IsPublished, &notebook.CreatedAt, &notebook.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	return []Notebook{notebook}, nil
}

func (s *PgxStore) CreateDraft(ctx context.Context, params CreateDraftParams) (int, error) {
	const query = `
		INSERT INTO notebooks (title, content, is_published, created_at, updated_at)
		VALUES ($1, $2, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var id int
	if err := s.DB.QueryRow(ctx, query, params.Title, params.Body).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func scanUser(row pgx.Row) (User, error) {
	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, err
		}
		return User{}, err
	}
	return user, nil
}

func (s *PgxStore) InsertAuditLog(ctx context.Context, params InsertAuditLogParams) error {
	const query = `
		INSERT INTO audit_events (actor_id, event_type, entity_type, entity_id)
		VALUES ($1, $2, $3, $4)
	`

	_, err := s.DB.Exec(ctx, query, params.UserID, params.Action, params.EntityType, params.EntityID)
	return err
}
func (s *PgxStore) ApproveRevisionTx(ctx context.Context, revisionID, notebookID, reviewerID int) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const updateRevisionQuery = `
		UPDATE notebook_revisions
		SET status = 'approved', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, updateRevisionQuery, revisionID)
	if err != nil {
		return err
	}

	const updateNotebookQuery = `
		UPDATE notebooks
		SET current_published_revision_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err = tx.Exec(ctx, updateNotebookQuery, revisionID, notebookID)
	if err != nil {
		return err
	}

	const auditQuery = `
		INSERT INTO audit_events (actor_id, event_type, entity_type, entity_id, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	`
	_, err = tx.Exec(ctx, auditQuery, reviewerID, "revision_approved", "notebook_revision", revisionID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *PgxStore) ListSubmittedRevisions(ctx context.Context) ([]NotebookRevision, error) {
	const query = `
		SELECT id, notebook_id, author_id, title, body, status, created_at, updated_at
		FROM notebook_revisions
		WHERE status = 'submitted'
		ORDER BY created_at ASC
	`

	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []NotebookRevision
	for rows.Next() {
		var rev NotebookRevision
		err := rows.Scan(&rev.ID, &rev.NotebookID, &rev.AuthorID, &rev.Title, &rev.Body, &rev.Status, &rev.CreatedAt, &rev.UpdatedAt)
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, rev)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return revisions, nil
}

func (s *PgxStore) UpdateRevisionStatus(ctx context.Context, params UpdateRevisionStatusParams) error {
	const query = `
		UPDATE notebook_revisions
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	_, err := s.DB.Exec(ctx, query, params.Status, params.ID)
	return err
}
