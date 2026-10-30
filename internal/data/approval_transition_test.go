package data

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// transitionTx is a small pgx.Tx fake for the transaction helpers. It records
// statements so the tests can verify both lock order and the write boundary.
type transitionTx struct {
	notebookExists     bool
	notebookID         int
	revisionNotebookID int
	resolvedNotebookID int
	teamID             int
	revisionExists     bool
	status             string
	member             bool
	conditionalRows    int64
	deleteRows         int64
	queryLog           []string
	execLog            []string
	execRows           []int64
}

func (tx *transitionTx) Begin(context.Context) (pgx.Tx, error) { return tx, nil }
func (tx *transitionTx) Commit(context.Context) error          { return nil }
func (tx *transitionTx) Rollback(context.Context) error        { return nil }
func (tx *transitionTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (tx *transitionTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *transitionTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *transitionTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (tx *transitionTx) Conn() *pgx.Conn { return nil }
func (tx *transitionTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query")
}
func (tx *transitionTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	tx.execLog = append(tx.execLog, sql)
	rows := int64(1)
	if strings.Contains(sql, "UPDATE notebook_revisions") {
		rows = tx.conditionalRows
	} else if strings.Contains(sql, "DELETE FROM notebooks") {
		rows = tx.deleteRows
	} else if len(tx.execRows) > 0 {
		rows, tx.execRows = tx.execRows[0], tx.execRows[1:]
	}
	return pgconn.NewCommandTag("UPDATE " + itoa(rows)), nil
}
func (tx *transitionTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	tx.queryLog = append(tx.queryLog, sql)
	switch {
	case strings.Contains(sql, "SELECT id FROM notebooks"):
		if !tx.notebookExists {
			return transitionRow{err: pgx.ErrNoRows}
		}
		return transitionRow{values: []any{tx.notebookID}}
	case strings.Contains(sql, "SELECT notebook_id, status FROM notebook_revisions"):
		if !tx.revisionExists {
			return transitionRow{err: pgx.ErrNoRows}
		}
		revisionNotebookID := tx.revisionNotebookID
		if revisionNotebookID == 0 {
			revisionNotebookID = tx.notebookID
		}
		return transitionRow{values: []any{revisionNotebookID, tx.status}}
	case strings.Contains(sql, "SELECT notebook_id FROM notebook_revisions"):
		if !tx.revisionExists {
			return transitionRow{err: pgx.ErrNoRows}
		}
		if tx.resolvedNotebookID != 0 {
			return transitionRow{values: []any{tx.resolvedNotebookID}}
		}
		revisionNotebookID := tx.revisionNotebookID
		if revisionNotebookID == 0 {
			revisionNotebookID = tx.notebookID
		}
		return transitionRow{values: []any{revisionNotebookID}}
	case strings.Contains(sql, "SELECT COALESCE(team_id, 0) FROM notebooks"):
		if !tx.notebookExists {
			return transitionRow{err: pgx.ErrNoRows}
		}
		return transitionRow{values: []any{tx.teamID}}
	case strings.Contains(sql, "ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE"):
		if !tx.revisionExists {
			return transitionRow{err: pgx.ErrNoRows}
		}
		now := time.Now()
		return transitionRow{values: []any{3, tx.notebookID, 7, "title", "body", tx.status, now, now}}
	case strings.Contains(sql, "FROM memberships"):
		if !tx.member {
			return transitionRow{err: pgx.ErrNoRows}
		}
		return transitionRow{values: []any{1}}
	default:
		return transitionRow{err: errors.New("unexpected QueryRow: " + sql)}
	}
}

type transitionRow struct {
	values []any
	err    error
}

func (r transitionRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("unexpected scan shape")
	}
	for i, value := range r.values {
		switch out := dest[i].(type) {
		case *int:
			*out = value.(int)
		case *string:
			*out = value.(string)
		case *time.Time:
			*out = value.(time.Time)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func submittedTransitionTx() *transitionTx {
	return &transitionTx{notebookExists: true, notebookID: 9, revisionExists: true, status: "submitted", conditionalRows: 1, deleteRows: 1, member: true}
}

func TestApproveRevisionTxChecksLocksAndConditionalTransition(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*transitionTx)
		wantErr  error
		wantExec int
	}{
		{name: "missing notebook", setup: func(tx *transitionTx) { tx.notebookExists = false }, wantErr: ErrNotFound},
		{name: "missing revision", setup: func(tx *transitionTx) { tx.revisionExists = false }, wantErr: ErrNotFound},
		{name: "revision belongs to another notebook", setup: func(tx *transitionTx) { tx.revisionNotebookID = 10 }, wantErr: ErrNotFound},
		{name: "draft cannot be approved", setup: func(tx *transitionTx) { tx.status = "draft" }, wantErr: ErrInvalidTransition},
		{name: "rejected cannot be approved", setup: func(tx *transitionTx) { tx.status = "rejected" }, wantErr: ErrInvalidTransition},
		{name: "already decided", setup: func(tx *transitionTx) { tx.status = "approved" }, wantErr: ErrInvalidTransition},
		{name: "notebook update affected zero rows", setup: func(tx *transitionTx) { tx.execRows = []int64{0} }, wantErr: ErrNotFound, wantExec: 2},
		{name: "conditional update lost race", setup: func(tx *transitionTx) { tx.conditionalRows = 0 }, wantErr: ErrInvalidTransition, wantExec: 1},
		{name: "submitted revision", setup: func(tx *transitionTx) {}, wantExec: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := submittedTransitionTx()
			tt.setup(tx)
			err := approveRevisionTx(context.Background(), tx, 3, 9, 42, "ok")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if len(tx.execLog) != tt.wantExec {
				t.Fatalf("executed %d writes, want %d", len(tx.execLog), tt.wantExec)
			}
			if len(tx.execLog) > 0 && (!strings.Contains(tx.execLog[0], "notebook_id = $3") || !strings.Contains(tx.execLog[0], "status = 'submitted'")) {
				t.Fatalf("transition update lacks submitted-state or notebook guard: %q", tx.execLog[0])
			}
			if len(tx.queryLog) >= 2 && (!strings.Contains(tx.queryLog[0], "FROM notebooks") || !strings.Contains(tx.queryLog[0], "FOR UPDATE") || !strings.Contains(tx.queryLog[1], "FROM notebook_revisions") || !strings.Contains(tx.queryLog[1], "FOR UPDATE")) {
				t.Fatalf("locks were not acquired notebook-first: %q", tx.queryLog[:2])
			}
		})
	}
}

func TestRejectRevisionTxChecksStateBeforeWriting(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*transitionTx)
		wantErr  error
		wantExec int
	}{
		{name: "missing revision", setup: func(tx *transitionTx) { tx.revisionExists = false }, wantErr: ErrNotFound},
		{name: "draft cannot be rejected", setup: func(tx *transitionTx) { tx.status = "draft" }, wantErr: ErrInvalidTransition},
		{name: "approved cannot be rejected", setup: func(tx *transitionTx) { tx.status = "approved" }, wantErr: ErrInvalidTransition},
		{name: "changed notebook association", setup: func(tx *transitionTx) { tx.resolvedNotebookID = 9; tx.revisionNotebookID = 10 }, wantErr: ErrNotFound},
		{name: "already decided", setup: func(tx *transitionTx) { tx.status = "rejected" }, wantErr: ErrInvalidTransition},
		{name: "conditional update lost race", setup: func(tx *transitionTx) { tx.conditionalRows = 0 }, wantErr: ErrInvalidTransition, wantExec: 1},
		{name: "submitted revision", setup: func(tx *transitionTx) {}, wantExec: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := submittedTransitionTx()
			tt.setup(tx)
			err := rejectRevisionTx(context.Background(), tx, 3, 42, "no")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if len(tx.execLog) != tt.wantExec {
				t.Fatalf("executed %d writes, want %d", len(tx.execLog), tt.wantExec)
			}
			if len(tx.execLog) > 0 && (!strings.Contains(tx.execLog[0], "notebook_id = $3") || !strings.Contains(tx.execLog[0], "status = 'submitted'")) {
				t.Fatalf("transition update lacks submitted-state or notebook guard: %q", tx.execLog[0])
			}
			if len(tx.queryLog) >= 3 && (!strings.Contains(tx.queryLog[1], "FROM notebooks") || !strings.Contains(tx.queryLog[1], "FOR UPDATE") || !strings.Contains(tx.queryLog[2], "FROM notebook_revisions") || !strings.Contains(tx.queryLog[2], "FOR UPDATE")) {
				t.Fatalf("rejection locks were not acquired notebook-first: %q", tx.queryLog)
			}
		})
	}
}

func TestDeleteNotebookAsTxRequiresAdminAndChecksDeleteCount(t *testing.T) {
	t.Run("published notebook admin can delete", func(t *testing.T) {
		tx := submittedTransitionTx()
		tx.status = "approved"
		got, err := deleteNotebookAsTx(context.Background(), tx, 9, 42, true)
		if err != nil {
			t.Fatal(err)
		}
		if got.Revision.NotebookID != 9 || got.Revision.ID != 3 || got.Revision.Status != "approved" || !got.DeletedNotebook {
			t.Fatalf("unexpected delete result: %+v", got)
		}
		if len(tx.execLog) != 1 || !strings.Contains(tx.execLog[0], "DELETE FROM notebooks") {
			t.Fatalf("expected one notebook delete, got %q", tx.execLog)
		}
		if !strings.Contains(tx.queryLog[2], "role = 'admin'") {
			t.Fatalf("admin role was not checked: %q", tx.queryLog)
		}
	})

	t.Run("nonmember denied without delete", func(t *testing.T) {
		tx := submittedTransitionTx()
		tx.status = "approved"
		tx.member = false
		_, err := deleteNotebookAsTx(context.Background(), tx, 9, 42, true)
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("error = %v, want ErrForbidden", err)
		}
		if len(tx.execLog) != 0 {
			t.Fatalf("denied delete performed writes: %q", tx.execLog)
		}
	})

	t.Run("zero affected rows is not found", func(t *testing.T) {
		tx := submittedTransitionTx()
		tx.status = "approved"
		tx.deleteRows = 0
		_, err := deleteNotebookAsTx(context.Background(), tx, 9, 42, true)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("error = %v, want ErrNotFound", err)
		}
	})
}
