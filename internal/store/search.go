package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// ErrInvalidSearchType is returned when SearchFilter.Types names a type
// outside SearchTypes.
var ErrInvalidSearchType = errors.New("invalid search type")

// SearchTypes is the canonical set of searchable content types.
var SearchTypes = []string{"documents", "decisions", "snippets", "solutions", "journal"}

const defaultSearchLimit = 20

// SearchFilter narrows Search. Empty Types => all of SearchTypes.
type SearchFilter struct {
	Types   []string
	Project *string
	Limit   int       // 0 => defaultSearchLimit
	Vector  []float32 // nil => FTS-only (2.8a behaviour); set => hybrid RRF fusion
}

// searchBranch is the per-type UNION-ALL fragment. `excerpt` is the text
// column ts_headline highlights; documents fall back to body::text (JSON,
// noisy — acceptable v1, refined when block-to-text render exists).
type searchBranch struct {
	typ     string
	table   string
	title   string // SQL expr yielding the hit Title
	excerpt string // SQL expr yielding the ts_headline source
}

var searchBranches = map[string]searchBranch{
	"documents": {"documents", "documents", "title", "coalesce(body::text, title)"},
	"decisions": {"decisions", "decisions", "title", "coalesce(nullif(rationale,''), decision)"},
	"snippets":  {"snippets", "snippets", "title", "coalesce(nullif(description,''), content)"},
	"solutions": {"solutions", "solutions", "error_description", "solution"},
	"journal":   {"journal", "journal_entries", "summary", "summary"},
}

func validateSearchTypes(types []string) ([]string, error) {
	if len(types) == 0 {
		return SearchTypes, nil
	}
	for _, t := range types {
		if _, ok := searchBranches[t]; !ok {
			return nil, fmt.Errorf("%w: %q", ErrInvalidSearchType, t)
		}
	}
	return types, nil
}

// Search runs unified hybrid search across the requested content types. It is
// a thin adapter over the dialect-free runHybridSearch, feeding it this
// backend's FTS and vector candidate producers; the fusion contract (RRF,
// relevance floor, FTS-only path, limit) lives in hybrid.go.
func (s *PostgresStore) Search(ctx context.Context, q string, f SearchFilter) ([]*models.SearchHit, error) {
	types, err := validateSearchTypes(f.Types)
	if err != nil {
		return nil, err
	}
	return runHybridSearch(ctx, s, q, types, f, s.vectorMaxDist)
}

// ftsCandidates is the Postgres FTS half of the search seam: per-type
// full-text matches with ts_headline excerpts. Rows are ordered by lexical
// rank so runHybridSearch's per-type row counter reproduces the reciprocal
// ranks the original PARTITION BY type window function produced.
func (s *PostgresStore) ftsCandidates(ctx context.Context, q string, types []string, f SearchFilter) ([]candidate, error) {
	// $1 = query text; $2 = project when filtering.
	args := []any{q}
	projClause := ""
	if f.Project != nil {
		args = append(args, *f.Project)
		projClause = " AND project = $2"
	}

	branches := make([]string, 0, len(types))
	for _, t := range types {
		b := searchBranches[t]
		branches = append(branches, fmt.Sprintf(
			`SELECT '%s' AS type, id::text AS id, %s AS title,
			   ts_headline('english', %s, websearch_to_tsquery('english', $1),
			     'MaxFragments=1, MaxWords=28, MinWords=8, StartSel=<<, StopSel=>>') AS excerpt,
			   project,
			   ts_rank(search_vector, websearch_to_tsquery('english', $1)) AS rank,
			   updated_at
			 FROM %s
			 WHERE search_vector @@ websearch_to_tsquery('english', $1)%s`,
			b.typ, b.title, b.excerpt, b.table, projClause))
	}

	sql := "WITH hits AS (\n" + strings.Join(branches, "\nUNION ALL\n") + `
)
SELECT type, id, title, excerpt, project, updated_at
FROM hits
ORDER BY rank DESC, updated_at DESC`

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("fts candidates: %w", err)
	}
	defer rows.Close()
	return collectFTSCandidates(rows)
}

// vectorCandidates is the Postgres vector half of the seam: the best chunk per
// live source, ranked by cosine similarity. Orphaned vectors (source deleted,
// not yet swept) are excluded via the live_sources join, matching the FTS path
// which reads the live tables directly. The relevance floor and the result
// limit are applied by runHybridSearch, not here.
func (s *PostgresStore) vectorCandidates(ctx context.Context, vec []float32, types []string, f SearchFilter) ([]candidate, error) {
	// $1 = query vector; $2 = project when filtering; then the types array.
	args := []any{pgvector.NewVector(vec)}
	qvecRef := "$1"
	projClause := ""
	if f.Project != nil {
		args = append(args, *f.Project)
		projClause = " AND e.project = $2"
	}
	args = append(args, types)
	typesRef := fmt.Sprintf("$%d", len(args))

	liveParts := make([]string, 0, len(types))
	for _, t := range types {
		b := searchBranches[t]
		liveParts = append(liveParts, fmt.Sprintf(
			`SELECT '%s' AS type, id::text AS id FROM %s`, b.typ, b.table))
	}

	sql := "WITH live_sources AS (\n" + strings.Join(liveParts, "\nUNION ALL\n") + `
),
vhits AS (
  SELECT DISTINCT ON (e.source_type, e.source_id)
         e.source_type AS type, e.source_id AS id, e.source_title AS title,
         left(e.chunk_text, 240) AS excerpt, e.project, e.created_at AS updated_at,
         1 - (e.embedding <=> ` + qvecRef + `) AS sim
  FROM embeddings e
  JOIN live_sources ls ON ls.type = e.source_type AND ls.id = e.source_id
  WHERE e.source_type = ANY(` + typesRef + `)` + projClause + `
  ORDER BY e.source_type, e.source_id, e.embedding <=> ` + qvecRef + `
)
SELECT type, id, title, excerpt, project, updated_at, sim
FROM vhits
ORDER BY sim DESC`

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("vector candidates: %w", err)
	}
	defer rows.Close()
	return collectVectorCandidates(rows)
}

func collectFTSCandidates(rows pgx.Rows) ([]candidate, error) {
	out := []candidate{}
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.Type, &c.ID, &c.Title, &c.Excerpt, &c.Project, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan fts candidate: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fts candidates: %w", err)
	}
	return out, nil
}

func collectVectorCandidates(rows pgx.Rows) ([]candidate, error) {
	out := []candidate{}
	for rows.Next() {
		var c candidate
		var sim float64
		if err := rows.Scan(&c.Type, &c.ID, &c.Title, &c.Excerpt, &c.Project, &c.UpdatedAt, &sim); err != nil {
			return nil, fmt.Errorf("scan vector candidate: %w", err)
		}
		c.Similarity = &sim
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vector candidates: %w", err)
	}
	return out, nil
}
