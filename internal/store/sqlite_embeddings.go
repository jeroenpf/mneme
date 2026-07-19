package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// UpsertEmbeddings inserts/updates rows keyed by (source_type, source_id,
// chunk_id), storing the vector as a packed float32 BLOB. Runs in one
// transaction so a batch re-embed is atomic. On conflict the existing row's id
// is kept (excluded columns overwrite the rest).
func (s *SQLiteStore) UpsertEmbeddings(ctx context.Context, rows []models.Embedding) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after a successful Commit
	const q = `
		INSERT INTO embeddings
		  (id, source_type, source_id, chunk_id, chunk_text, embedding, project, source_title, model)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT (source_type, source_id, chunk_id) DO UPDATE SET
		  chunk_text   = excluded.chunk_text,
		  embedding    = excluded.embedding,
		  project      = excluded.project,
		  source_title = excluded.source_title,
		  model        = excluded.model`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare upsert: %w", err)
	}
	defer stmt.Close()
	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx,
			newUUID(), r.SourceType, r.SourceID, r.ChunkID, r.ChunkText,
			packFloat32(r.Embedding), r.Project, r.SourceTitle, r.Model,
		); err != nil {
			return fmt.Errorf("upsert embeddings: %w", err)
		}
	}
	return tx.Commit()
}

// DeleteEmbeddingsExcept prunes a source's chunks whose chunk_id is not in keep.
func (s *SQLiteStore) DeleteEmbeddingsExcept(ctx context.Context, sourceType, sourceID string, keep []string) error {
	if len(keep) == 0 {
		_, err := s.db.ExecContext(ctx,
			`DELETE FROM embeddings WHERE source_type = ? AND source_id = ?`, sourceType, sourceID)
		if err != nil {
			return fmt.Errorf("prune embeddings: %w", err)
		}
		return nil
	}
	args := make([]any, 0, len(keep)+2)
	args = append(args, sourceType, sourceID)
	for _, k := range keep {
		args = append(args, k)
	}
	q := `DELETE FROM embeddings WHERE source_type = ? AND source_id = ? AND chunk_id NOT IN (` +
		placeholders(len(keep)) + `)`
	if _, err := s.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("prune embeddings: %w", err)
	}
	return nil
}

// placeholders returns "?,?,...,?" with n marks.
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// EmbeddingsFor returns chunk_id → chunk_text for a source.
func (s *SQLiteStore) EmbeddingsFor(ctx context.Context, sourceType, sourceID string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT chunk_id, chunk_text FROM embeddings WHERE source_type = ? AND source_id = ?`,
		sourceType, sourceID)
	if err != nil {
		return nil, fmt.Errorf("embeddings for: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, txt string
		if err := rows.Scan(&id, &txt); err != nil {
			return nil, err
		}
		out[id] = txt
	}
	return out, rows.Err()
}

// SourceRefs enumerates every embeddable source across the five types.
func (s *SQLiteStore) SourceRefs(ctx context.Context) ([]SourceRef, error) {
	parts := make([]string, 0, len(sourceTables))
	for _, st := range sourceTables {
		parts = append(parts, fmt.Sprintf(`SELECT '%s' AS type, id AS id FROM %s`, st.typ, st.table))
	}
	rows, err := s.db.QueryContext(ctx, joinUnion(parts))
	if err != nil {
		return nil, fmt.Errorf("source refs: %w", err)
	}
	defer rows.Close()
	out := []SourceRef{}
	for rows.Next() {
		var r SourceRef
		if err := rows.Scan(&r.Type, &r.ID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// EmbeddingStatus reports per-type reconciliation buckets against the current
// model. Mirrors the Postgres query (no ::text casts — SQLite ids are TEXT).
func (s *SQLiteStore) EmbeddingStatus(ctx context.Context, model string) ([]TypeStatus, error) {
	out := make([]TypeStatus, 0, len(sourceTables))
	for _, st := range sourceTables {
		var ts TypeStatus
		ts.Type = st.typ
		q := fmt.Sprintf(`
			SELECT
			  (SELECT count(*) FROM %[1]s) AS total,
			  (SELECT count(*) FROM %[1]s t
			     WHERE EXISTS (SELECT 1 FROM embeddings e
			                    WHERE e.source_type = ? AND e.source_id = t.id)) AS embedded,
			  (SELECT count(*) FROM %[1]s t
			     WHERE EXISTS (SELECT 1 FROM embeddings e
			                    WHERE e.source_type = ? AND e.source_id = t.id AND e.model <> ?)) AS stale,
			  (SELECT count(DISTINCT e.source_id) FROM embeddings e
			     WHERE e.source_type = ?
			       AND NOT EXISTS (SELECT 1 FROM %[1]s t WHERE t.id = e.source_id)) AS orphaned,
			  (SELECT count(*) FROM embed_failures f
			     JOIN %[1]s t ON t.id = f.source_id
			    WHERE f.source_type = ?) AS failed`, st.table)
		if err := s.db.QueryRowContext(ctx, q, st.typ, st.typ, model, st.typ, st.typ).
			Scan(&ts.Total, &ts.Embedded, &ts.Stale, &ts.Orphaned, &ts.Failed); err != nil {
			return nil, fmt.Errorf("embedding status %s: %w", st.typ, err)
		}
		ts.Reconciled = ts.Embedded - ts.Stale
		ts.Missing = ts.Total - ts.Embedded
		out = append(out, ts)
	}
	return out, nil
}

// HasStaleModelEmbeddings reports whether a source has any vector on a model
// other than model.
func (s *SQLiteStore) HasStaleModelEmbeddings(ctx context.Context, sourceType, sourceID, model string) (bool, error) {
	var stale bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM embeddings
		   WHERE source_type = ? AND source_id = ? AND model <> ?)`,
		sourceType, sourceID, model).Scan(&stale); err != nil {
		return false, fmt.Errorf("stale-model check: %w", err)
	}
	return stale, nil
}

// DeleteOrphanEmbeddings sweeps vectors whose source_id no longer resolves to a
// live row of that type, returning the total removed.
func (s *SQLiteStore) DeleteOrphanEmbeddings(ctx context.Context) (int64, error) {
	var deleted int64
	for _, st := range sourceTables {
		res, err := s.db.ExecContext(ctx, fmt.Sprintf(
			`DELETE FROM embeddings
			  WHERE source_type = ?
			    AND NOT EXISTS (SELECT 1 FROM %s t WHERE t.id = embeddings.source_id)`, st.table), st.typ)
		if err != nil {
			return deleted, fmt.Errorf("sweep orphans %s: %w", st.typ, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return deleted, fmt.Errorf("sweep orphans rows %s: %w", st.typ, err)
		}
		deleted += n
	}
	return deleted, nil
}

// RecordEmbedFailure upserts a terminal embed failure, accruing attempts and
// refreshing last_failed_at.
func (s *SQLiteStore) RecordEmbedFailure(ctx context.Context, sourceType, sourceID, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO embed_failures (source_type, source_id, error)
		VALUES (?, ?, ?)
		ON CONFLICT (source_type, source_id) DO UPDATE SET
		  error = excluded.error,
		  attempts = embed_failures.attempts + 1,
		  last_failed_at = `+nowExpr,
		sourceType, sourceID, errMsg)
	if err != nil {
		return fmt.Errorf("record embed failure: %w", err)
	}
	return nil
}

// ClearEmbedFailure removes any recorded failure for a source (no-op if absent).
func (s *SQLiteStore) ClearEmbedFailure(ctx context.Context, sourceType, sourceID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM embed_failures WHERE source_type = ? AND source_id = ?`, sourceType, sourceID)
	if err != nil {
		return fmt.Errorf("clear embed failure: %w", err)
	}
	return nil
}

// FailedSourceRefs lists every source with a recorded terminal failure.
func (s *SQLiteStore) FailedSourceRefs(ctx context.Context) ([]SourceRef, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT source_type, source_id FROM embed_failures ORDER BY last_failed_at`)
	if err != nil {
		return nil, fmt.Errorf("failed source refs: %w", err)
	}
	defer rows.Close()
	out := []SourceRef{}
	for rows.Next() {
		var r SourceRef
		if err := rows.Scan(&r.Type, &r.ID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
