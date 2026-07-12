package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

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
	Limit   int // 0 => defaultSearchLimit
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

	// $1 = query text; $2 = project (when filtering); $3 = limit.
	args := []any{q}
	projClause := ""
	if f.Project != nil {
		args = append(args, *f.Project)
		projClause = " AND project = $2"
	}
	args = append(args, limit)
	limitRef := fmt.Sprintf("$%d", len(args))

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

	sql := `WITH hits AS (
` + strings.Join(branches, "\nUNION ALL\n") + `
), ranked AS (
  SELECT type, id, title, excerpt, project, updated_at,
         1.0 / (` + fmt.Sprintf("%d", rrfK) + ` + row_number() OVER (
           PARTITION BY type ORDER BY rank DESC, updated_at DESC)) AS score
  FROM hits
)
SELECT type, id, title, excerpt, project, score, updated_at
FROM ranked
ORDER BY score DESC, updated_at DESC
LIMIT ` + limitRef

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
		if err := rows.Scan(&h.Type, &h.ID, &h.Title, &h.Excerpt, &h.Project, &h.Score, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan search hit: %w", err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search hits: %w", err)
	}
	return out, nil
}
