package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jeroenpf/mneme/internal/models"
)

const sqliteRelationColumns = `id, from_id, to_ref, to_id, rel_type, origin, created_at`

func (s *SQLiteStore) ReplaceAutoMentions(ctx context.Context, fromID string, mentions []*models.Relation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("replace auto mentions: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM relations WHERE from_id = ? AND origin = 'auto'`, fromID); err != nil {
		return fmt.Errorf("replace auto mentions: delete: %w", err)
	}
	for _, m := range mentions {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO relations (from_id, to_ref, to_id, rel_type, origin)
			 VALUES (?, ?, ?, ?, 'auto')`,
			fromID, m.ToRef, m.ToID, m.RelType); err != nil {
			return fmt.Errorf("replace auto mentions: insert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("replace auto mentions: commit: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CreateRelation(ctx context.Context, rel *models.Relation) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT OR IGNORE INTO relations (from_id, to_ref, to_id, rel_type, origin)
		 VALUES (?, ?, ?, ?, ?)
		 RETURNING id, created_at`,
		rel.FromID, rel.ToRef, rel.ToID, rel.RelType, rel.Origin,
	).Scan(&rel.ID, &rel.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // duplicate edge — silent no-op
	}
	if err != nil {
		return fmt.Errorf("create relation: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteExplicitRelations(ctx context.Context, fromID, toRef string, relType *string) (int64, error) {
	q := `DELETE FROM relations WHERE from_id = ? AND to_ref = ? AND origin = 'explicit'`
	args := []any{fromID, toRef}
	if relType != nil {
		q += ` AND rel_type = ?`
		args = append(args, *relType)
	}
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("delete explicit relations: %w", err)
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) ListRelations(ctx context.Context, publicID, altRef string) (outgoing, incoming []*models.Relation, err error) {
	outgoing, err = s.queryRelations(ctx,
		`SELECT `+sqliteRelationColumns+` FROM relations WHERE from_id = ? ORDER BY created_at, id`, publicID)
	if err != nil {
		return nil, nil, fmt.Errorf("list relations: outgoing: %w", err)
	}
	incoming, err = s.queryRelations(ctx,
		`SELECT `+sqliteRelationColumns+` FROM relations
		 WHERE to_id = ? OR to_ref = ? OR (? <> '' AND to_ref = ?)
		 ORDER BY created_at, id`, publicID, publicID, altRef, altRef)
	if err != nil {
		return nil, nil, fmt.Errorf("list relations: incoming: %w", err)
	}
	return outgoing, incoming, nil
}

func (s *SQLiteStore) queryRelations(ctx context.Context, q string, args ...any) ([]*models.Relation, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Relation
	for rows.Next() {
		r := &models.Relation{}
		if err := rows.Scan(&r.ID, &r.FromID, &r.ToRef, &r.ToID, &r.RelType, &r.Origin, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CountRelations(ctx context.Context) (int64, error) {
	var n int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM relations`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count relations: %w", err)
	}
	return n, nil
}
