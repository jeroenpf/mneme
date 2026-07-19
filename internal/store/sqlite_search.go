package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// --- websearch → FTS5 query translation ------------------------------------

// toFTS5Query translates a websearch-style query (implicit AND, "quoted
// phrases", the `or` operator, and `-exclusion`) into an FTS5 MATCH expression.
// Every term becomes a double-quoted phrase (embedded quotes doubled), so no
// input token is ever interpreted as an FTS5 operator or special character —
// the function can never produce a string FTS5 rejects as a syntax error.
//
// It returns ok=false when the query yields no positive term (empty, whitespace,
// punctuation-only, or exclusion-only); the caller then skips the FTS channel.
func toFTS5Query(q string) (string, bool) {
	type token struct {
		text string
		neg  bool
		isOr bool
	}
	var toks []token
	rs := []rune(q)
	n := len(rs)
	i := 0
	for i < n {
		for i < n && unicode.IsSpace(rs[i]) {
			i++
		}
		if i >= n {
			break
		}
		neg := false
		if rs[i] == '-' {
			neg = true
			i++
			if i >= n || unicode.IsSpace(rs[i]) {
				continue // a lone '-' is not an exclusion
			}
		}
		if rs[i] == '"' {
			i++
			start := i
			for i < n && rs[i] != '"' {
				i++
			}
			phrase := string(rs[start:i])
			if i < n {
				i++ // consume the closing quote
			}
			if hasSearchable(phrase) {
				toks = append(toks, token{text: phrase, neg: neg})
			}
			continue
		}
		start := i
		for i < n && !unicode.IsSpace(rs[i]) && rs[i] != '"' {
			i++
		}
		word := string(rs[start:i])
		if word == "" {
			continue
		}
		if !neg && strings.EqualFold(word, "or") {
			toks = append(toks, token{isOr: true})
			continue
		}
		if hasSearchable(word) {
			toks = append(toks, token{text: word, neg: neg})
		}
	}

	// Positive terms group into AND-clauses split on `or`; negatives are
	// collected globally and excluded with a single NOT.
	var groups [][]string
	var cur []string
	var negs []string
	flush := func() {
		if len(cur) > 0 {
			groups = append(groups, cur)
			cur = nil
		}
	}
	for _, tk := range toks {
		switch {
		case tk.isOr:
			flush()
		case tk.neg:
			negs = append(negs, tk.text)
		default:
			cur = append(cur, tk.text)
		}
	}
	flush()

	if len(groups) == 0 {
		return "", false
	}

	orParts := make([]string, len(groups))
	for gi, g := range groups {
		andParts := make([]string, len(g))
		for ti, term := range g {
			andParts[ti] = quoteFTS(term)
		}
		orParts[gi] = "(" + strings.Join(andParts, " AND ") + ")"
	}
	out := strings.Join(orParts, " OR ")

	if len(negs) > 0 {
		negParts := make([]string, len(negs))
		for ni, term := range negs {
			negParts[ni] = quoteFTS(term)
		}
		out = "(" + out + ") NOT (" + strings.Join(negParts, " OR ") + ")"
	}
	return out, true
}

// quoteFTS wraps a term as an FTS5 phrase literal, doubling any embedded quote.
func quoteFTS(term string) string {
	return `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
}

// hasSearchable reports whether s contains at least one letter or digit, so
// punctuation-only tokens (which FTS5 would tokenize to nothing) are dropped.
func hasSearchable(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// --- unified hybrid search (the dialect-free seam) -------------------------

// sqliteFTSBranch describes one type's FTS5 table for the candidate producer:
// its virtual table, base table, the base column projected as the hit title,
// the bm25 per-column weights reproducing the Postgres setweight A/B/C tiers
// ({A,B,C} = {1.0, 0.4, 0.2}), and which FTS column snippet() draws the excerpt
// from. Documents pin the snippet to the title column (0): their body column is
// raw JSON (indexed for matching, but noisy as an excerpt), and Postgres
// likewise excerpts the title when a document has no embedding chunk to quote.
// The other types have only clean text columns, so -1 auto-selects the best.
type sqliteFTSBranch struct {
	ftsTable   string
	baseTable  string
	titleCol   string
	weights    string
	snippetCol int
}

var sqliteFTSBranches = map[string]sqliteFTSBranch{
	"documents": {"documents_fts", "documents", "title", "1.0, 0.4, 0.4, 0.2", 0},
	"decisions": {"decisions_fts", "decisions", "title", "1.0, 0.4, 0.4, 0.2, 0.2", -1},
	"snippets":  {"snippets_fts", "snippets", "title", "1.0, 0.4, 0.2", -1},
	"solutions": {"solutions_fts", "solutions", "error_description", "1.0, 0.4", -1},
	"journal":   {"journal_fts", "journal_entries", "summary", "1.0, 0.4, 0.2, 0.2", -1},
	"memory":    {"memories_fts", "memories", "key", "1.0, 0.4, 0.2", -1},
}

// Search runs unified hybrid search across the requested content types via the
// dialect-free runHybridSearch, feeding it this backend's FTS and vector
// candidate producers.
func (s *SQLiteStore) Search(ctx context.Context, q string, f SearchFilter) ([]*models.SearchHit, error) {
	types, err := validateSearchTypes(f.Types)
	if err != nil {
		return nil, err
	}
	return runHybridSearch(ctx, s, q, types, f, s.vectorMaxDist)
}

// ftsCandidates is the SQLite FTS half of the search seam: bm25-ranked full-text
// matches across the requested types with snippet() excerpts (<<...>>
// delimiters, matching the Postgres ts_headline), returned in one global rank
// order so runHybridSearch can reciprocal-rank them across types. bm25 is
// negated so a larger value is a better match (matching the Postgres rank DESC).
func (s *SQLiteStore) ftsCandidates(ctx context.Context, q string, types []string, f SearchFilter) ([]candidate, error) {
	match, ok := toFTS5Query(q)
	if !ok {
		return []candidate{}, nil
	}
	hasProject := f.Project != nil
	var args []any
	branches := make([]string, 0, len(types))
	for _, t := range types {
		br := sqliteFTSBranches[t]
		args = append(args, match)
		proj := ""
		if hasProject {
			// Project scope also admits global (NULL) rows, matching the FTS
			// path in Postgres.
			proj = " AND (b.project = ? OR b.project IS NULL)"
			args = append(args, *f.Project)
		}
		branches = append(branches, fmt.Sprintf(
			`SELECT '%[1]s' AS type, b.id AS id, b.%[4]s AS title,
			   snippet(%[2]s, %[7]d, '<<', '>>', '…', 28) AS excerpt,
			   b.project AS project,
			   -bm25(%[2]s, %[5]s) AS rank,
			   b.updated_at AS updated_at
			 FROM %[2]s
			 JOIN %[3]s b ON b.rowid = %[2]s.rowid
			 WHERE %[2]s MATCH ?%[6]s`,
			t, br.ftsTable, br.baseTable, br.titleCol, br.weights, proj, br.snippetCol))
	}
	query := "WITH hits AS (\n" + strings.Join(branches, "\nUNION ALL\n") + `
)
SELECT type, id, title, excerpt, project, updated_at
FROM hits
ORDER BY rank DESC, updated_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("fts candidates: %w", err)
	}
	defer rows.Close()
	return collectSQLiteFTSCandidates(rows)
}

// fillPublicIDs looks up each hit's public id from its base table, grouping by
// table so it runs at most one query per type present on the final page.
func (s *SQLiteStore) fillPublicIDs(ctx context.Context, hits []*models.SearchHit) error {
	idsByTable := map[string][]string{}
	for _, h := range hits {
		if table, ok := publicIDTable(h.Type); ok {
			idsByTable[table] = append(idsByTable[table], h.ID)
		}
	}
	pub := map[string]map[string]string{} // table -> id -> public_id
	for table, ids := range idsByTable {
		m := map[string]string{}
		args := make([]any, len(ids))
		for i, id := range ids {
			args[i] = id
		}
		q := `SELECT id, public_id FROM ` + table + ` WHERE id IN (` + placeholders(len(ids)) + `)`
		rows, err := s.db.QueryContext(ctx, q, args...)
		if err != nil {
			return fmt.Errorf("fill public ids (%s): %w", table, err)
		}
		for rows.Next() {
			var id, publicID string
			if err := rows.Scan(&id, &publicID); err != nil {
				rows.Close()
				return fmt.Errorf("scan public id: %w", err)
			}
			m[id] = publicID
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate public ids (%s): %w", table, err)
		}
		pub[table] = m
	}
	for _, h := range hits {
		if table, ok := publicIDTable(h.Type); ok {
			h.PublicID = pub[table][h.ID]
		}
	}
	return nil
}

func collectSQLiteFTSCandidates(rows *sql.Rows) ([]candidate, error) {
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

// vectorCandidates is the SQLite vector half of the seam: brute-force cosine in
// Go over the packed float32 BLOBs, keeping the best chunk per live source. It
// excludes orphaned vectors (source since deleted) via the live-source set. The
// relevance floor and result limit are applied by runHybridSearch, not here.
func (s *SQLiteStore) vectorCandidates(ctx context.Context, vec []float32, types []string, f SearchFilter) ([]candidate, error) {
	if len(vec) == 0 || len(types) == 0 {
		return []candidate{}, nil
	}
	live, err := s.liveSourceSet(ctx)
	if err != nil {
		return nil, err
	}

	var b sqliteQB
	typeArgs := make([]any, len(types))
	for i, t := range types {
		typeArgs[i] = t
	}
	b.add("source_type IN ("+placeholders(len(types))+")", typeArgs...)
	if f.Project != nil {
		b.add("(project = ? OR project IS NULL)", *f.Project)
	}
	q := `SELECT source_type, source_id, source_title, chunk_text, project, created_at, embedding
		FROM embeddings` + b.whereClause()

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("vector candidates: %w", err)
	}
	defer rows.Close()

	type key struct{ typ, id string }
	best := map[key]*candidate{}
	for rows.Next() {
		var typ, id, title, chunk string
		var project *string
		var updatedAt time.Time
		var blob []byte
		if err := rows.Scan(&typ, &id, &title, &chunk, &project, &updatedAt, &blob); err != nil {
			return nil, fmt.Errorf("scan vector candidate: %w", err)
		}
		k := key{typ, id}
		if !live[[2]string{typ, id}] {
			continue // orphaned vector
		}
		emb, err := unpackFloat32(blob)
		if err != nil {
			return nil, err
		}
		sim := cosineSim(vec, emb)
		if cur, ok := best[k]; ok && *cur.Similarity >= sim {
			continue
		}
		simCopy := sim
		best[k] = &candidate{
			Type: typ, ID: id, Title: title, Excerpt: truncateRunes(chunk, 240),
			Project: project, UpdatedAt: updatedAt, Similarity: &simCopy,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vector candidates: %w", err)
	}

	out := make([]candidate, 0, len(best))
	for _, c := range best {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return *out[i].Similarity > *out[j].Similarity })
	return out, nil
}

// liveSourceSet returns the set of (type, id) that still resolve to a live base
// row, so vectorCandidates can drop orphaned embeddings.
func (s *SQLiteStore) liveSourceSet(ctx context.Context) (map[[2]string]bool, error) {
	refs, err := s.SourceRefs(ctx)
	if err != nil {
		return nil, err
	}
	live := make(map[[2]string]bool, len(refs))
	for _, r := range refs {
		live[[2]string{r.Type, r.ID}] = true
	}
	return live, nil
}

// cosineSim computes cosine similarity (1 = identical direction, 0 = orthogonal,
// -1 = opposite). Mismatched or empty vectors score 0.
func cosineSim(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		av, bv := float64(a[i]), float64(b[i])
		dot += av * bv
		na += av * av
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// truncateRunes returns the first max runes of s (character-safe, matching
// Postgres left()), avoiding a split mid-UTF-8-sequence.
func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// --- type-scoped FTS search ------------------------------------------------

// prefixColumns qualifies each comma-separated column in cols with prefix,
// preserving order so the shared row scanners still apply.
func prefixColumns(cols, prefix string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = prefix + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}

func (s *SQLiteStore) SearchDocuments(ctx context.Context, query string, f Filter) ([]*models.Document, error) {
	match, ok := toFTS5Query(query)
	if !ok {
		return []*models.Document{}, nil
	}
	var b sqliteQB
	b.add("documents_fts MATCH ?", match)
	if f.Project != nil {
		b.add("b.project = ?", *f.Project)
	}
	if f.Type != nil {
		b.add("b.type = ?", *f.Type)
	}
	if f.Status != nil {
		b.add("b.status = ?", *f.Status)
	}
	for _, tag := range f.Tags {
		b.add("EXISTS (SELECT 1 FROM json_each(b.tags) WHERE value = ?)", tag)
	}
	q := `SELECT ` + prefixColumns(documentColumns, "b.") + `
		FROM documents_fts JOIN documents b ON b.rowid = documents_fts.rowid` +
		b.whereClause() +
		` ORDER BY bm25(documents_fts, 1.0, 0.4, 0.4, 0.2), b.updated_at DESC` +
		b.limitOffset(f)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	defer rows.Close()
	return collectSQLiteDocuments(rows)
}

func (s *SQLiteStore) SearchDecisions(ctx context.Context, query string, f DecisionFilter) ([]*models.Decision, error) {
	match, ok := toFTS5Query(query)
	if !ok {
		return []*models.Decision{}, nil
	}
	var b sqliteQB
	b.add("decisions_fts MATCH ?", match)
	if f.Project != nil {
		b.add("b.project = ?", *f.Project)
	}
	if f.Status != nil {
		b.add("b.status = ?", string(*f.Status))
	}
	q := `SELECT ` + prefixColumns(decisionColumns, "b.") + `
		FROM decisions_fts JOIN decisions b ON b.rowid = decisions_fts.rowid` +
		b.whereClause() +
		` ORDER BY bm25(decisions_fts, 1.0, 0.4, 0.4, 0.2, 0.2), b.created_at DESC` +
		b.limit(f.Limit)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search decisions: %w", err)
	}
	defer rows.Close()
	out := []*models.Decision{}
	for rows.Next() {
		d, err := scanSQLiteDecision(rows)
		if err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) SearchSnippets(ctx context.Context, query string, f SnippetFilter) ([]*models.Snippet, error) {
	match, ok := toFTS5Query(query)
	if !ok {
		return []*models.Snippet{}, nil
	}
	var b sqliteQB
	b.add("snippets_fts MATCH ?", match)
	if f.Project != nil {
		b.add("b.project = ?", *f.Project)
	}
	if f.Language != nil {
		b.add("b.language = ?", *f.Language)
	}
	for _, tag := range f.Tags {
		b.add("EXISTS (SELECT 1 FROM json_each(b.tags) WHERE value = ?)", tag)
	}
	q := `SELECT ` + prefixColumns(snippetColumns, "b.") + `
		FROM snippets_fts JOIN snippets b ON b.rowid = snippets_fts.rowid` +
		b.whereClause() +
		` ORDER BY bm25(snippets_fts, 1.0, 0.4, 0.2), b.created_at DESC` +
		b.limit(f.Limit)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search snippets: %w", err)
	}
	defer rows.Close()
	out := []*models.Snippet{}
	for rows.Next() {
		sn, err := scanSQLiteSnippet(rows)
		if err != nil {
			return nil, fmt.Errorf("scan snippet: %w", err)
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) SearchSolutions(ctx context.Context, query string, f SolutionFilter) ([]*models.Solution, error) {
	match, ok := toFTS5Query(query)
	if !ok {
		return []*models.Solution{}, nil
	}
	var b sqliteQB
	b.add("solutions_fts MATCH ?", match)
	if f.Project != nil {
		b.add("b.project = ?", *f.Project)
	}
	for _, tag := range f.Tags {
		b.add("EXISTS (SELECT 1 FROM json_each(b.tags) WHERE value = ?)", tag)
	}
	q := `SELECT ` + prefixColumns(solutionColumns, "b.") + `
		FROM solutions_fts JOIN solutions b ON b.rowid = solutions_fts.rowid` +
		b.whereClause() +
		` ORDER BY bm25(solutions_fts, 1.0, 0.4), b.created_at DESC` +
		b.limit(f.Limit)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search solutions: %w", err)
	}
	defer rows.Close()
	out := []*models.Solution{}
	for rows.Next() {
		sol, err := scanSQLiteSolution(rows)
		if err != nil {
			return nil, fmt.Errorf("scan solution: %w", err)
		}
		out = append(out, sol)
	}
	return out, rows.Err()
}
