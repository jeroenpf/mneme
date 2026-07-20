package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jeroenpf/mneme/internal/models"
)

// scanSQLiteDocumentRevision reads a document_revisions row, decoding the JSON
// TEXT columns (target_ids/meta/body) that Postgres stores natively.
func scanSQLiteDocumentRevision(row rowScanner) (*models.DocumentRevision, error) {
	r := &models.DocumentRevision{}
	var targets, meta, body string
	err := row.Scan(
		&r.ID, &r.DocumentID, &r.Revision, &r.Op, &r.Actor, &targets,
		&r.Title, &r.Status, &meta, &body, &r.CreatedAt,
	)
	if err != nil {
		return r, err
	}
	if r.TargetIDs, err = scanJSONArray(targets); err != nil {
		return nil, err
	}
	if r.Meta, err = scanJSONObject(meta); err != nil {
		return nil, err
	}
	if r.Body, err = scanJSONObject(body); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *SQLiteStore) AppendDocumentRevision(ctx context.Context, rev *models.DocumentRevision) error {
	meta, err := jsonObject(rev.Meta)
	if err != nil {
		return err
	}
	body, err := jsonObject(rev.Body)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO document_revisions (
			document_id, revision, op, actor, target_ids, title, status, meta, body
		) VALUES (?,?,?,?,?,?,?,?,?)
		RETURNING id, created_at`
	err = s.db.QueryRowContext(ctx, q,
		rev.DocumentID, rev.Revision, rev.Op, rev.Actor, jsonArray(rev.TargetIDs),
		rev.Title, rev.Status, meta, body,
	).Scan(&rev.ID, &rev.CreatedAt)
	if err != nil {
		return fmt.Errorf("append document revision: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListDocumentRevisions(ctx context.Context, documentID string, limit int) ([]*models.DocumentRevision, error) {
	q := `SELECT ` + documentRevisionColumns + ` FROM document_revisions
		WHERE document_id = ? ORDER BY revision DESC`
	args := []any{documentID}
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list document revisions: %w", err)
	}
	defer rows.Close()
	out := []*models.DocumentRevision{}
	for rows.Next() {
		r, err := scanSQLiteDocumentRevision(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document revision: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetDocumentRevision(ctx context.Context, documentID string, revision int) (*models.DocumentRevision, error) {
	q := `SELECT ` + documentRevisionColumns + ` FROM document_revisions
		WHERE document_id = ? AND revision = ?`
	r, err := scanSQLiteDocumentRevision(s.db.QueryRowContext(ctx, q, documentID, revision))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document revision: %w", err)
	}
	return r, nil
}
