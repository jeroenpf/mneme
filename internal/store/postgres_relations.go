package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/jeroenpf/mneme/internal/models"
)

const relationColumns = `id, from_id, to_ref, to_id, rel_type, origin, created_at`

func (s *PostgresStore) ReplaceAutoMentions(ctx context.Context, fromID string, mentions []*models.Relation) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("replace auto mentions: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM relations WHERE from_id = $1 AND origin = 'auto'`, fromID); err != nil {
		return fmt.Errorf("replace auto mentions: delete: %w", err)
	}
	for _, m := range mentions {
		if _, err := tx.Exec(ctx,
			`INSERT INTO relations (from_id, to_ref, to_id, rel_type, origin)
			 VALUES ($1, $2, $3, $4, 'auto')
			 ON CONFLICT (from_id, to_ref, rel_type) DO NOTHING`,
			fromID, m.ToRef, m.ToID, m.RelType); err != nil {
			return fmt.Errorf("replace auto mentions: insert: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("replace auto mentions: commit: %w", err)
	}
	return nil
}

func (s *PostgresStore) CreateRelation(ctx context.Context, rel *models.Relation) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO relations (from_id, to_ref, to_id, rel_type, origin)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (from_id, to_ref, rel_type) DO NOTHING
		 RETURNING id, created_at`,
		rel.FromID, rel.ToRef, rel.ToID, rel.RelType, rel.Origin,
	).Scan(&rel.ID, &rel.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // duplicate edge — silent no-op
	}
	if err != nil {
		return fmt.Errorf("create relation: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteExplicitRelations(ctx context.Context, fromID, toRef string, relType *string) (int64, error) {
	q := `DELETE FROM relations WHERE from_id = $1 AND to_ref = $2 AND origin = 'explicit'`
	args := []any{fromID, toRef}
	if relType != nil {
		q += ` AND rel_type = $3`
		args = append(args, *relType)
	}
	tag, err := s.pool.Exec(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("delete explicit relations: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *PostgresStore) ListRelations(ctx context.Context, publicID, altRef string) (outgoing, incoming []*models.Relation, err error) {
	outgoing, err = s.queryRelations(ctx,
		`SELECT `+relationColumns+` FROM relations WHERE from_id = $1 ORDER BY created_at, id`, publicID)
	if err != nil {
		return nil, nil, fmt.Errorf("list relations: outgoing: %w", err)
	}
	incoming, err = s.queryRelations(ctx,
		`SELECT `+relationColumns+` FROM relations
		 WHERE to_id = $1 OR to_ref = $1 OR ($2 <> '' AND to_ref = $2)
		 ORDER BY created_at, id`, publicID, altRef)
	if err != nil {
		return nil, nil, fmt.Errorf("list relations: incoming: %w", err)
	}
	return outgoing, incoming, nil
}

func (s *PostgresStore) queryRelations(ctx context.Context, q string, args ...any) ([]*models.Relation, error) {
	rows, err := s.pool.Query(ctx, q, args...)
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

func (s *PostgresStore) CountRelations(ctx context.Context) (int64, error) {
	var n int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM relations`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count relations: %w", err)
	}
	return n, nil
}
