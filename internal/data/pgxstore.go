package data

import (
	"context"
	"errors"
	"fmt"

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
		ON CONFLICT (user_id, team_id) DO UPDATE SET role = EXCLUDED.role
	`

	_, err := s.DB.Exec(ctx, query, userID, teamID, role)
	if err != nil {
		return err
	}

	return nil
}

func (s *PgxStore) CreateUser(ctx context.Context, params CreateUserParams) (int, error) {
	const query = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash, updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	var id int
	if err := s.DB.QueryRow(ctx, query, params.Email, params.PasswordHash).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *PgxStore) DeleteNotebook(ctx context.Context, notebookID int) error {
	const query = `DELETE FROM notebooks WHERE id = $1`
	_, err := s.DB.Exec(ctx, query, notebookID)
	return err
}

func (s *PgxStore) DeleteNotebookRevision(ctx context.Context, revisionID int) error {
	const query = `DELETE FROM notebook_revisions WHERE id = $1`
	_, err := s.DB.Exec(ctx, query, revisionID)
	return err
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
		SELECT nr.id, nr.notebook_id, nr.author_id, nr.title, nr.body, nr.status,
		       COALESCE(a.note, '') AS review_note, nr.created_at, nr.updated_at
		FROM notebook_revisions nr
		LEFT JOIN approvals a ON a.notebook_revision_id = nr.id
		WHERE nr.notebook_id = $1
		ORDER BY nr.created_at DESC
	`

	rows, err := s.DB.Query(ctx, query, notebookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []NotebookRevision
	for rows.Next() {
		var rev NotebookRevision
		err := rows.Scan(&rev.ID, &rev.NotebookID, &rev.AuthorID, &rev.Title, &rev.Body, &rev.Status, &rev.ReviewNote, &rev.CreatedAt, &rev.UpdatedAt)
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
        INSERT INTO notebook_tags (notebook_id, tag_id) VALUES ($1, $2) 
        ON CONFLICT (notebook_id, tag_id) DO NOTHING`

	_, err := s.DB.Exec(ctx, query, notebookID, tagID)
	return err
}

func (s *PgxStore) ListNotebookTags(ctx context.Context, notebookID int) ([]Tag, error) {
	query := `
        SELECT t.id, t.name 
        FROM tags t 
        JOIN notebook_tags nt ON t.id = nt.tag_id 
        WHERE nt.notebook_id = $1 
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
	const query = `
		SELECT u.id, u.email, u.password_hash, COALESCE(m.role, 'member')
		FROM users u
		LEFT JOIN memberships m ON m.user_id = u.id
		WHERE u.email = $1
		ORDER BY
			CASE m.role
				WHEN 'admin' THEN 1
				WHEN 'reviewer' THEN 2
				ELSE 3
			END
		LIMIT 1
	`
	return scanUser(s.DB.QueryRow(ctx, query, email))
}

func (s *PgxStore) GetUser(ctx context.Context, id int) (User, error) {
	const query = `
		SELECT u.id, u.email, u.password_hash, COALESCE(m.role, 'member')
		FROM users u
		LEFT JOIN memberships m ON m.user_id = u.id
		WHERE u.id = $1
		ORDER BY
			CASE m.role
				WHEN 'admin' THEN 1
				WHEN 'reviewer' THEN 2
				ELSE 3
			END
		LIMIT 1
	`
	return scanUser(s.DB.QueryRow(ctx, query, id))
}

func (s *PgxStore) CheckPassword(user User, password string) bool {
	return bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)) == nil
}

func (s *PgxStore) ListNotebooks(ctx context.Context) ([]Notebook, error) {
	const query = `
		SELECT id, title, content, slug, visibility, is_published, created_at, updated_at
		FROM notebooks
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
		var slug, visibility *string
		if err := rows.Scan(&notebook.ID, &notebook.Title, &notebook.Content, &slug, &visibility, &notebook.IsPublished, &notebook.CreatedAt, &notebook.UpdatedAt); err != nil {
			return nil, err
		}
		if slug != nil {
			notebook.Slug = *slug
		}
		if visibility != nil {
			notebook.Visibility = *visibility
		}
		notebooks = append(notebooks, notebook)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notebooks, nil
}

func (s *PgxStore) SearchNotebooks(ctx context.Context, queryText string) ([]Notebook, error) {
	const query = `
		SELECT id, title, content, slug, visibility, is_published, created_at, updated_at
		FROM notebooks
		WHERE title ILIKE '%' || $1 || '%'
		   OR content ILIKE '%' || $1 || '%'
		   OR slug ILIKE '%' || $1 || '%'
		   OR EXISTS (
				SELECT 1
				FROM notebook_tags nt
				JOIN tags t ON t.id = nt.tag_id
				WHERE nt.notebook_id = notebooks.id
				  AND t.name ILIKE '%' || $1 || '%'
		   )
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(ctx, query, queryText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notebooks := []Notebook{}
	for rows.Next() {
		var notebook Notebook
		var slug, visibility *string
		if err := rows.Scan(&notebook.ID, &notebook.Title, &notebook.Content, &slug, &visibility, &notebook.IsPublished, &notebook.CreatedAt, &notebook.UpdatedAt); err != nil {
			return nil, err
		}
		if slug != nil {
			notebook.Slug = *slug
		}
		if visibility != nil {
			notebook.Visibility = *visibility
		}
		notebooks = append(notebooks, notebook)
	}

	return notebooks, rows.Err()
}

func (s *PgxStore) NotebookView(ctx context.Context, id int) ([]Notebook, error) {
	const query = `
		SELECT id, title, content, slug, visibility, is_published, created_at, updated_at
		FROM notebooks
		WHERE id = $1
	`

	row := s.DB.QueryRow(ctx, query, id)
	var notebook Notebook
	var slug, visibility *string
	if err := row.Scan(&notebook.ID, &notebook.Title, &notebook.Content, &slug, &visibility, &notebook.IsPublished, &notebook.CreatedAt, &notebook.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	if slug != nil {
		notebook.Slug = *slug
	}
	if visibility != nil {
		notebook.Visibility = *visibility
	}

	return []Notebook{notebook}, nil
}

func (s *PgxStore) CreateDraft(ctx context.Context, params CreateDraftParams) (int, error) {
	const query = `
		INSERT INTO notebooks (title, content, slug, visibility, is_published, created_at, updated_at)
		VALUES ($1, $2, $3, $4, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var id int
	if err := s.DB.QueryRow(ctx, query, params.Title, params.Body, params.Slug, params.Visibility).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *PgxStore) UpdateDraft(ctx context.Context, params UpdateDraftParams) (revisionID int, err error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, rollbackErr)
		}
	}()

	const updateNotebookQuery = `
		UPDATE notebooks
		SET title = $1, content = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	if _, err = tx.Exec(ctx, updateNotebookQuery, params.Title, params.Body, params.NotebookID); err != nil {
		return 0, err
	}

	const insertRevisionQuery = `
		INSERT INTO notebook_revisions (notebook_id, author_id, title, body, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'draft', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`
	if err = tx.QueryRow(ctx, insertRevisionQuery, params.NotebookID, params.AuthorID, params.Title, params.Body).Scan(&revisionID); err != nil {
		return 0, err
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return revisionID, nil
}

func (s *PgxStore) ListRecentDrafts(ctx context.Context, authorID int) ([]NotebookRevision, error) {
	const query = `
		SELECT nr.id, nr.notebook_id, nr.author_id, nr.title, nr.body, nr.status,
		       COALESCE(a.note, '') AS review_note, nr.created_at, nr.updated_at
		FROM notebook_revisions nr
		LEFT JOIN approvals a ON a.notebook_revision_id = nr.id
		WHERE nr.author_id = $1
		ORDER BY nr.updated_at DESC
		LIMIT 6
	`

	rows, err := s.DB.Query(ctx, query, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []NotebookRevision
	for rows.Next() {
		var rev NotebookRevision
		if err := rows.Scan(&rev.ID, &rev.NotebookID, &rev.AuthorID, &rev.Title, &rev.Body, &rev.Status, &rev.ReviewNote, &rev.CreatedAt, &rev.UpdatedAt); err != nil {
			return nil, err
		}
		revisions = append(revisions, rev)
	}
	return revisions, rows.Err()
}

func (s *PgxStore) ListAuditEvents(ctx context.Context) ([]AuditEvent, error) {
	const query = `
		SELECT ae.id, COALESCE(ae.actor_id, 0), ae.event_type, ae.entity_type, ae.entity_id,
			   COALESCE(u.email, ''), COALESCE(ae.details::text, ''), ae.created_at
		FROM audit_events ae
		LEFT JOIN users u ON ae.actor_id = u.id
		ORDER BY ae.created_at DESC
		LIMIT 100
	`

	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(&event.ID, &event.ActorID, &event.EventType, &event.EntityType, &event.EntityID, &event.ActorEmail, &event.Details, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func scanUser(row pgx.Row) (User, error) {
	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, err
		}
		return User{}, err
	}
	return user, nil
}

func (s *PgxStore) InsertAuditLog(ctx context.Context, params InsertAuditLogParams) error {
	const query = `
		INSERT INTO audit_events (actor_id, event_type, entity_type, entity_id, details)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`

	var details interface{}
	if params.Details == "" {
		details = nil
	} else {
		details = params.Details
	}

	_, err := s.DB.Exec(ctx, query, params.UserID, params.Action, params.EntityType, params.EntityID, details)
	return err
}
func (s *PgxStore) ApproveRevisionTx(ctx context.Context, revisionID, notebookID, reviewerID int, note string) (err error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, rollbackErr)
		}
	}()

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
		SET current_published_revision_id = $1,
			title = nr.title,
			content = nr.body,
			is_published = true,
			updated_at = CURRENT_TIMESTAMP
		FROM notebook_revisions nr
		WHERE notebooks.id = $2
		  AND nr.id = $1
	`
	_, err = tx.Exec(ctx, updateNotebookQuery, revisionID, notebookID)
	if err != nil {
		return err
	}

	const auditQuery = `
		INSERT INTO audit_events (actor_id, event_type, entity_type, entity_id, details, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, CURRENT_TIMESTAMP)
	`
	var details interface{}
	if note == "" {
		details = nil
	} else {
		details = fmt.Sprintf(`{"note": %q}`, note)
	}
	_, err = tx.Exec(ctx, auditQuery, reviewerID, "revision_approved", "notebook_revision", revisionID, details)
	if err != nil {
		return err
	}

	const approvalQuery = `
		INSERT INTO approvals (notebook_revision_id, reviewer_id, decision, note, created_at)
		VALUES ($1, $2, 'approved', $3, CURRENT_TIMESTAMP)
	`
	_, err = tx.Exec(ctx, approvalQuery, revisionID, reviewerID, note)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	return err
}

func (s *PgxStore) RejectRevisionTx(ctx context.Context, revisionID, reviewerID int, note string) (err error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, rollbackErr)
		}
	}()

	const updateRevisionQuery = `
		UPDATE notebook_revisions
		SET status = 'rejected', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, updateRevisionQuery, revisionID)
	if err != nil {
		return err
	}

	const approvalQuery = `
		INSERT INTO approvals (notebook_revision_id, reviewer_id, decision, note, created_at)
		VALUES ($1, $2, 'rejected', $3, CURRENT_TIMESTAMP)
	`
	_, err = tx.Exec(ctx, approvalQuery, revisionID, reviewerID, note)
	if err != nil {
		return err
	}

	const auditQuery = `
		INSERT INTO audit_events (actor_id, event_type, entity_type, entity_id, details, created_at)
		VALUES ($1, 'revision_rejected', 'notebook_revision', $2, $3::jsonb, CURRENT_TIMESTAMP)
	`
	var details interface{}
	if note == "" {
		details = nil
	} else {
		details = fmt.Sprintf(`{"note": %q}`, note)
	}
	_, err = tx.Exec(ctx, auditQuery, reviewerID, revisionID, details)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	return err
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

func (s *PgxStore) CreateModerationFlag(ctx context.Context, params CreateModerationFlagParams) (int, error) {
	const query = `
		INSERT INTO moderation_flags (notebook_id, moderator_id, reason, created_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var id int
	err := s.DB.QueryRow(ctx, query, params.NotebookID, params.ModeratorID, params.Reason).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *PgxStore) ListModerationFlags(ctx context.Context) ([]ModerationFlag, error) {
	const query = `
		SELECT id, notebook_id, COALESCE(moderator_id, 0), reason, resolved_at, created_at
		FROM moderation_flags
		ORDER BY resolved_at NULLS FIRST, created_at DESC
		LIMIT 100
	`

	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []ModerationFlag
	for rows.Next() {
		var flag ModerationFlag
		if err := rows.Scan(&flag.ID, &flag.NotebookID, &flag.ModeratorID, &flag.Reason, &flag.ResolvedAt, &flag.CreatedAt); err != nil {
			return nil, err
		}
		flags = append(flags, flag)
	}
	return flags, rows.Err()
}

func (s *PgxStore) ResolveModerationFlag(ctx context.Context, id int) error {
	const query = `
		UPDATE moderation_flags
		SET resolved_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := s.DB.Exec(ctx, query, id)
	return err
}
