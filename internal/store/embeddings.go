package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/jeroenpf/mneme/internal/models"
)

// SourceRef identifies an embeddable source row.
type SourceRef struct {
	Type string
	ID   string
}

// TypeStatus reports, per source type, how its live sources and stored
// vectors divide into reconciliation buckets:
//
//   - Embedded   — live sources with at least one vector (Reconciled+Stale).
//   - Reconciled — live sources whose vectors are all on the current model.
//   - Stale      — live sources holding at least one vector on an older model.
//   - Missing    — live sources with no vector at all (Total-Embedded).
//   - Orphaned   — vectors whose source_id no longer resolves to a live row.
//   - Failed     — live sources whose last embed attempt failed terminally
//     (from embed_failures); 0 when none are currently failing.
type TypeStatus struct {
	Type       string `json:"type"`
	Total      int    `json:"total"`
	Embedded   int    `json:"embedded"`
	Reconciled int    `json:"reconciled"`
	Missing    int    `json:"missing"`
	Stale      int    `json:"stale"`
	Orphaned   int    `json:"orphaned"`
	Failed     int    `json:"failed"`
}

// sourceTables maps a SearchTypes value to its table, for enumeration and
// coverage. documents.id is TEXT, the rest UUID → ::text.
var sourceTables = []struct{ typ, table string }{
	{"documents", "documents"},
	{"decisions", "decisions"},
	{"snippets", "snippets"},
	{"solutions", "solutions"},
	{"journal", "journal_entries"},
}

// UpsertEmbeddings inserts/updates rows keyed by (source_type, source_id,
// chunk_id). Embedding vectors are passed via pgvector.NewVector (encoded
// through its driver.Valuer, so no pool-level type registration is needed).
func (s *PostgresStore) UpsertEmbeddings(ctx context.Context, rows []models.Embedding) error {
	if len(rows) == 0 {
		return nil
	}
	const q = `
		INSERT INTO embeddings
		  (source_type, source_id, chunk_id, chunk_text, embedding, project, source_title, model)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (source_type, source_id, chunk_id) DO UPDATE
		SET chunk_text = EXCLUDED.chunk_text,
		    embedding  = EXCLUDED.embedding,
		    project    = EXCLUDED.project,
		    source_title = EXCLUDED.source_title,
		    model      = EXCLUDED.model`
	batch := &pgx.Batch{}
	for _, r := range rows {
		batch.Queue(q, r.SourceType, r.SourceID, r.ChunkID, r.ChunkText,
			pgvector.NewVector(r.Embedding), r.Project, r.SourceTitle, r.Model)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range rows {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("upsert embeddings: %w", err)
		}
	}
	return nil
}

// DeleteEmbeddingsExcept prunes chunks of a source whose chunk_id is not in
// keep — used after a re-embed to drop sections that no longer exist.
func (s *PostgresStore) DeleteEmbeddingsExcept(ctx context.Context, sourceType, sourceID string, keep []string) error {
	if len(keep) == 0 {
		_, err := s.pool.Exec(ctx,
			`DELETE FROM embeddings WHERE source_type=$1 AND source_id=$2`, sourceType, sourceID)
		return err
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM embeddings WHERE source_type=$1 AND source_id=$2 AND chunk_id <> ALL($3)`,
		sourceType, sourceID, keep)
	if err != nil {
		return fmt.Errorf("prune embeddings: %w", err)
	}
	return nil
}

// EmbeddingsFor returns chunk_id → chunk_text for a source, so a re-embed
// can skip unchanged chunks.
func (s *PostgresStore) EmbeddingsFor(ctx context.Context, sourceType, sourceID string) (map[string]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT chunk_id, chunk_text FROM embeddings WHERE source_type=$1 AND source_id=$2`,
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
func (s *PostgresStore) SourceRefs(ctx context.Context) ([]SourceRef, error) {
	parts := make([]string, 0, len(sourceTables))
	for _, st := range sourceTables {
		parts = append(parts, fmt.Sprintf(`SELECT '%s' AS type, id::text AS id FROM %s`, st.typ, st.table))
	}
	rows, err := s.pool.Query(ctx, joinUnion(parts))
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
// embedding model. Reconciled and Missing are derived from the queried Total,
// Embedded, and Stale counts (see TypeStatus). Failed is left nil — terminal
// failure tracking arrives in P3-t5.
func (s *PostgresStore) EmbeddingStatus(ctx context.Context, model string) ([]TypeStatus, error) {
	out := make([]TypeStatus, 0, len(sourceTables))
	for _, st := range sourceTables {
		var ts TypeStatus
		ts.Type = st.typ
		// One row: total live sources, live sources with any vector, live
		// sources with an outdated-model vector, and orphaned source_ids.
		if err := s.pool.QueryRow(ctx, fmt.Sprintf(`
			SELECT
			  (SELECT count(*) FROM %[1]s) AS total,
			  (SELECT count(*) FROM %[1]s t
			     WHERE EXISTS (SELECT 1 FROM embeddings e
			                    WHERE e.source_type=$1 AND e.source_id=t.id::text)) AS embedded,
			  (SELECT count(*) FROM %[1]s t
			     WHERE EXISTS (SELECT 1 FROM embeddings e
			                    WHERE e.source_type=$1 AND e.source_id=t.id::text AND e.model<>$2)) AS stale,
			  (SELECT count(DISTINCT e.source_id) FROM embeddings e
			     WHERE e.source_type=$1
			       AND NOT EXISTS (SELECT 1 FROM %[1]s t WHERE t.id::text=e.source_id)) AS orphaned,
			  (SELECT count(*) FROM embed_failures f
			     JOIN %[1]s t ON t.id::text = f.source_id
			    WHERE f.source_type=$1) AS failed`,
			st.table), st.typ, model).Scan(&ts.Total, &ts.Embedded, &ts.Stale, &ts.Orphaned, &ts.Failed); err != nil {
			return nil, fmt.Errorf("embedding status %s: %w", st.typ, err)
		}
		ts.Reconciled = ts.Embedded - ts.Stale
		ts.Missing = ts.Total - ts.Embedded
		out = append(out, ts)
	}
	return out, nil
}

// HasStaleModelEmbeddings reports whether a source has any stored vector on a
// model other than model — the signal that a model switch has left the source
// mixing vector spaces and must be fully re-embedded.
func (s *PostgresStore) HasStaleModelEmbeddings(ctx context.Context, sourceType, sourceID, model string) (bool, error) {
	var stale bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM embeddings
		   WHERE source_type=$1 AND source_id=$2 AND model <> $3)`,
		sourceType, sourceID, model).Scan(&stale); err != nil {
		return false, fmt.Errorf("stale-model check: %w", err)
	}
	return stale, nil
}

// DeleteOrphanEmbeddings sweeps vectors whose source_id no longer resolves to
// a live row of that type, across every embeddable type, and returns the total
// number of rows removed. Reconciliation only enqueues live sources, so a
// deleted source's chunks are never re-processed — this is how they get
// collected. Idempotent: a clean index sweeps zero rows.
func (s *PostgresStore) DeleteOrphanEmbeddings(ctx context.Context) (int64, error) {
	var deleted int64
	for _, st := range sourceTables {
		tag, err := s.pool.Exec(ctx, fmt.Sprintf(
			`DELETE FROM embeddings e
			  WHERE e.source_type = $1
			    AND NOT EXISTS (SELECT 1 FROM %s t WHERE t.id::text = e.source_id)`, st.table), st.typ)
		if err != nil {
			return deleted, fmt.Errorf("sweep orphans %s: %w", st.typ, err)
		}
		deleted += tag.RowsAffected()
	}
	return deleted, nil
}

// RecordEmbedFailure upserts a terminal embed failure for a source, keeping
// the latest error and accruing the attempt count and last_failed_at.
func (s *PostgresStore) RecordEmbedFailure(ctx context.Context, sourceType, sourceID, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO embed_failures (source_type, source_id, error)
		VALUES ($1, $2, $3)
		ON CONFLICT (source_type, source_id) DO UPDATE
		SET error = EXCLUDED.error,
		    attempts = embed_failures.attempts + 1,
		    last_failed_at = now()`,
		sourceType, sourceID, errMsg)
	if err != nil {
		return fmt.Errorf("record embed failure: %w", err)
	}
	return nil
}

// ClearEmbedFailure removes any recorded failure for a source (called after a
// successful embed or a purge). Absent rows are a no-op.
func (s *PostgresStore) ClearEmbedFailure(ctx context.Context, sourceType, sourceID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM embed_failures WHERE source_type=$1 AND source_id=$2`, sourceType, sourceID)
	if err != nil {
		return fmt.Errorf("clear embed failure: %w", err)
	}
	return nil
}

// FailedSourceRefs lists every source with a recorded terminal failure, so a
// manual retry can re-enqueue them. Includes failures for since-deleted
// sources — re-processing self-cleans them (the purge path clears the row).
func (s *PostgresStore) FailedSourceRefs(ctx context.Context) ([]SourceRef, error) {
	rows, err := s.pool.Query(ctx,
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

func joinUnion(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += "\nUNION ALL\n" + p
	}
	return out
}
