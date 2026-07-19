package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// SourceRef identifies an embeddable source row.
type SourceRef struct {
	Type string
	ID   string
}

// TypeCoverage reports how many sources of a type have at least one embedding.
type TypeCoverage struct {
	Type     string `json:"type"`
	Embedded int    `json:"embedded"`
	Total    int    `json:"total"`
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

// EmbeddingCoverage returns, per type, how many sources have at least one
// embedding vs the total number of sources.
func (s *PostgresStore) EmbeddingCoverage(ctx context.Context) ([]TypeCoverage, error) {
	out := make([]TypeCoverage, 0, len(sourceTables))
	for _, st := range sourceTables {
		var total, embedded int
		if err := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s`, st.table)).Scan(&total); err != nil {
			return nil, fmt.Errorf("coverage total %s: %w", st.typ, err)
		}
		if err := s.pool.QueryRow(ctx, fmt.Sprintf(
			`SELECT count(DISTINCT e.source_id)
			   FROM embeddings e
			   JOIN %s t ON t.id::text = e.source_id
			  WHERE e.source_type=$1`, st.table), st.typ).Scan(&embedded); err != nil {
			return nil, fmt.Errorf("coverage embedded %s: %w", st.typ, err)
		}
		out = append(out, TypeCoverage{Type: st.typ, Embedded: embedded, Total: total})
	}
	return out, nil
}

func joinUnion(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += "\nUNION ALL\n" + p
	}
	return out
}
