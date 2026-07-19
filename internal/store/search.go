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

// rrfK is the reciprocal-rank-fusion constant. Larger flattens the
// contribution of rank position; 60 is the conventional default.
const rrfK = 60

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

// Search runs a unified FTS query across the requested content types,
// ranked cross-type by reciprocal rank (1/(k+rank) per type). Returns
// ErrInvalidSearchType for an unknown type.
func (s *PostgresStore) Search(ctx context.Context, q string, f SearchFilter) ([]*models.SearchHit, error) {
	types, err := validateSearchTypes(f.Types)
	if err != nil {
		return nil, err
	}
	limit := f.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	// $1 = query text; $2 = project (when filtering). The limit and, for
	// the hybrid path, the query vector + types array are appended after
	// the branches so their placeholder numbers depend on the path taken.
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

	// franked is the 2.8a FTS term: per-type reciprocal rank on ts_rank.
	ftsCTE := `hits AS (
` + strings.Join(branches, "\nUNION ALL\n") + `
), franked AS (
  SELECT type, id, title, excerpt, project, updated_at,
         1.0 / (` + fmt.Sprintf("%d", rrfK) + ` + row_number() OVER (
           PARTITION BY type ORDER BY rank DESC, updated_at DESC)) AS fts_score
  FROM hits
)`

	if f.Vector == nil {
		// 2.8a path: order franked directly (byte-for-byte the FTS query).
		args = append(args, limit)
		limitRef := fmt.Sprintf("$%d", len(args))
		sql := "WITH " + ftsCTE + `
SELECT type, id, title, excerpt, project, fts_score AS score, NULL::float8 AS similarity, updated_at
FROM franked
ORDER BY score DESC, updated_at DESC
LIMIT ` + limitRef
		return s.runSearch(ctx, sql, args)
	}

	// Hybrid path: add the vector term and reciprocal-rank-fuse it with the
	// FTS term. $qvec and $types are new args appended after $1/$2.
	args = append(args, pgvector.NewVector(f.Vector))
	qvecRef := fmt.Sprintf("$%d", len(args))
	args = append(args, types)
	typesRef := fmt.Sprintf("$%d", len(args))
	vProjClause := ""
	if f.Project != nil {
		// project is already bound at $2 by the FTS arg setup above.
		vProjClause = " AND e.project = $2"
	}
	// Relevance floor: drop vector candidates whose cosine distance exceeds
	// the threshold, so a vague/irrelevant query returns nothing rather than
	// the whole corpus. Only the vector side is gated — FTS (franked) matches
	// always pass. Disabled when vectorMaxDist <= 0.
	vFloorClause := ""
	if s.vectorMaxDist > 0 {
		args = append(args, s.vectorMaxDist)
		vFloorClause = " AND (e.embedding <=> " + qvecRef + ") < " + fmt.Sprintf("$%d", len(args))
	}
	args = append(args, limit)
	limitRef := fmt.Sprintf("$%d", len(args))

	// live_sources enumerates the id of every live source of the requested
	// types, so vhits can drop orphaned vectors (source deleted, not yet
	// swept). The FTS path reads live tables directly; this keeps the vector
	// path consistent — a deleted source never surfaces from either.
	liveParts := make([]string, 0, len(types))
	for _, t := range types {
		b := searchBranches[t]
		liveParts = append(liveParts, fmt.Sprintf(
			`SELECT '%s' AS type, id::text AS id FROM %s`, b.typ, b.table))
	}
	liveSourcesCTE := "live_sources AS (\n" + strings.Join(liveParts, "\nUNION ALL\n") + "\n)"

	sql := "WITH " + ftsCTE + `,
` + liveSourcesCTE + `,
vhits AS (
  SELECT DISTINCT ON (e.source_type, e.source_id)
         e.source_type AS type, e.source_id AS id, e.source_title AS title,
         left(e.chunk_text, 240) AS excerpt, e.project, e.created_at AS updated_at,
         1 - (e.embedding <=> ` + qvecRef + `) AS sim
  FROM embeddings e
  JOIN live_sources ls ON ls.type = e.source_type AND ls.id = e.source_id
  WHERE e.source_type = ANY(` + typesRef + `)` + vProjClause + vFloorClause + `
  ORDER BY e.source_type, e.source_id, e.embedding <=> ` + qvecRef + `
),
vranked AS (
  SELECT type, id, title, excerpt, project, updated_at, sim,
         1.0 / (` + fmt.Sprintf("%d", rrfK) + ` + row_number() OVER (ORDER BY sim DESC)) AS vec_score
  FROM vhits
),
fused AS (
  SELECT
    coalesce(f.type, v.type)             AS type,
    coalesce(f.id, v.id)                 AS id,
    coalesce(f.title, v.title)           AS title,
    coalesce(f.excerpt, v.excerpt)       AS excerpt,
    coalesce(f.project, v.project)       AS project,
    coalesce(f.updated_at, v.updated_at) AS updated_at,
    coalesce(f.fts_score, 0) + coalesce(v.vec_score, 0) AS score,
    v.sim                                AS similarity
  FROM franked f FULL OUTER JOIN vranked v ON f.type = v.type AND f.id = v.id
)
SELECT type, id, title, excerpt, project, score, similarity, updated_at
FROM fused
ORDER BY score DESC, updated_at DESC
LIMIT ` + limitRef
	return s.runSearch(ctx, sql, args)
}

// runSearch runs a prepared search SQL + args and collects the hits. Shared
// by the FTS-only and hybrid paths of Search.
func (s *PostgresStore) runSearch(ctx context.Context, sql string, args []any) ([]*models.SearchHit, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	return collectSearchHits(rows)
}

func collectSearchHits(rows pgx.Rows) ([]*models.SearchHit, error) {
	out := []*models.SearchHit{}
	for rows.Next() {
		h := &models.SearchHit{}
		if err := rows.Scan(&h.Type, &h.ID, &h.Title, &h.Excerpt, &h.Project, &h.Score, &h.Similarity, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan search hit: %w", err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search hits: %w", err)
	}
	return out, nil
}
