package store

import (
	"errors"
	"fmt"

	"context"

	"github.com/jackc/pgx/v5"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// documentRevisionColumns is the shared SELECT list for document_revisions,
// used by both backends (SQLite decodes the JSON TEXT columns after scanning).
const documentRevisionColumns = `
	id, document_id, revision, op, actor, target_ids,
	title, status, meta, body, created_at`

func scanDocumentRevision(row pgx.Row) (*models.DocumentRevision, error) {
	r := &models.DocumentRevision{}
	err := row.Scan(
		&r.ID, &r.DocumentID, &r.Revision, &r.Op, &r.Actor, &r.TargetIDs,
		&r.Title, &r.Status, &r.Meta, &r.Body, &r.CreatedAt,
	)
	return r, err
}

func (s *PostgresStore) AppendDocumentRevision(ctx context.Context, rev *models.DocumentRevision) error {
	const q = `
		INSERT INTO document_revisions (
			document_id, revision, op, actor, target_ids, title, status, meta, body
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at`
	err := s.pool.QueryRow(ctx, q,
		rev.DocumentID, rev.Revision, rev.Op, rev.Actor, ensureTags(rev.TargetIDs),
		rev.Title, rev.Status, ensureJSONMap(rev.Meta), ensureJSONMap(rev.Body),
	).Scan(&rev.ID, &rev.CreatedAt)
	if err != nil {
		return fmt.Errorf("append document revision: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListDocumentRevisions(ctx context.Context, documentID string, limit int) ([]*models.DocumentRevision, error) {
	q := `SELECT ` + documentRevisionColumns + ` FROM document_revisions
		WHERE document_id = $1 ORDER BY revision DESC`
	args := []any{documentID}
	if limit > 0 {
		q += ` LIMIT $2`
		args = append(args, limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list document revisions: %w", err)
	}
	defer rows.Close()
	out := []*models.DocumentRevision{}
	for rows.Next() {
		r, err := scanDocumentRevision(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document revision: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetDocumentRevision(ctx context.Context, documentID string, revision int) (*models.DocumentRevision, error) {
	q := `SELECT ` + documentRevisionColumns + ` FROM document_revisions
		WHERE document_id = $1 AND revision = $2`
	r, err := scanDocumentRevision(s.pool.QueryRow(ctx, q, documentID, revision))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document revision: %w", err)
	}
	return r, nil
}
