package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeroenpf/mneme/internal/models"
)

// PostgreSQL SQLSTATE codes we translate to typed errors. See
// https://www.postgresql.org/docs/16/errcodes-appendix.html
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// translateWriteErr maps known PG SQLSTATEs on documents writes to
// typed store errors. Returns the original (wrapped) error when no
// mapping applies.
func translateWriteErr(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == pgUniqueViolation:
			return ErrDuplicateID
		case pgErr.Code == pgForeignKeyViolation && pgErr.ConstraintName == "documents_project_fkey":
			return ErrInvalidProject
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}

// PostgresStore is the production Store implementation. It owns the
// pgxpool, so callers construct it once at startup and Close() it on
// shutdown.
type PostgresStore struct {
	pool *pgxpool.Pool
	// vectorMaxDist is the hybrid-search relevance floor (cosine distance):
	// vector candidates beyond it are dropped so a vague query returns
	// nothing rather than the whole corpus. <= 0 disables it. Set once at
	// startup via SetSearchMaxDist; keyword (FTS) matches ignore it.
	vectorMaxDist float64
}

// SetSearchMaxDist sets the hybrid-search relevance floor (cosine distance).
// Call once at startup from config, before the store is shared across
// requests. <= 0 disables the floor (always-ranked semantic results).
func (s *PostgresStore) SetSearchMaxDist(d float64) { s.vectorMaxDist = d }

// NewPostgresStore builds a PostgresStore from a DSN. It opens the pool,
// applies the modest connection limits used by Mneme, and pings before
// returning. store.New dispatches here for postgres:// DSNs.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

// NewWithPool is for tests — wraps an already-open pool. The caller
// retains ownership; Close() still closes the pool, so don't share it.
func NewWithPool(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Close() { s.pool.Close() }

func (s *PostgresStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Pool exposes the underlying pool for callers that need it (e.g. for
// pgx-specific operations not yet on the Store interface).
func (s *PostgresStore) Pool() *pgxpool.Pool { return s.pool }

// documentColumns is the SELECT list used by every Get/List/Search query.
// Keep this in lockstep with scanDocument.
const documentColumns = `
	id, public_id, title, project, category, type, status,
	ticket, repo, tags, phase_current, phase_total,
	meta, body, created_at, updated_at, revision`

func scanDocument(row pgx.Row) (*models.Document, error) {
	d := &models.Document{}
	err := row.Scan(
		&d.ID, &d.PublicID, &d.Title, &d.Project, &d.Category, &d.Type, &d.Status,
		&d.Ticket, &d.Repo, &d.Tags, &d.PhaseCurrent, &d.PhaseTotal,
		&d.Meta, &d.Body, &d.CreatedAt, &d.UpdatedAt, &d.Revision,
	)
	return d, err
}

func ensureJSONMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func ensureTags(t []string) []string {
	if t == nil {
		return []string{}
	}
	return t
}

func (s *PostgresStore) CreateDocument(ctx context.Context, doc *models.Document) error {
	const q = `
		INSERT INTO documents (
			id, title, project, category, type, status,
			ticket, repo, tags, phase_current, phase_total,
			meta, body
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13
		)
		RETURNING public_id, created_at, updated_at, revision`
	row := s.pool.QueryRow(ctx, q,
		doc.ID, doc.Title, doc.Project, doc.Category, doc.Type, doc.Status,
		doc.Ticket, doc.Repo, ensureTags(doc.Tags), doc.PhaseCurrent, doc.PhaseTotal,
		ensureJSONMap(doc.Meta), ensureJSONMap(doc.Body),
	)
	if err := row.Scan(&doc.PublicID, &doc.CreatedAt, &doc.UpdatedAt, &doc.Revision); err != nil {
		return translateWriteErr("insert document", err)
	}
	return nil
}

func (s *PostgresStore) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	q := `SELECT ` + documentColumns + ` FROM documents WHERE id = $1`
	doc, err := scanDocument(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return doc, nil
}

func (s *PostgresStore) GetDocumentByPublicID(ctx context.Context, publicID string) (*models.Document, error) {
	q := `SELECT ` + documentColumns + ` FROM documents WHERE public_id = $1`
	doc, err := scanDocument(s.pool.QueryRow(ctx, q, publicID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document by public id: %w", err)
	}
	return doc, nil
}

func (s *PostgresStore) UpdateDocument(ctx context.Context, doc *models.Document, expected *int) error {
	const q = `
		UPDATE documents SET
			title = $2, project = $3, category = $4, type = $5, status = $6,
			ticket = $7, repo = $8, tags = $9, phase_current = $10, phase_total = $11,
			meta = $12, body = $13, revision = revision + 1
		WHERE id = $1 AND ($14::int IS NULL OR revision = $14)
		RETURNING updated_at, revision`
	row := s.pool.QueryRow(ctx, q,
		doc.ID, doc.Title, doc.Project, doc.Category, doc.Type, doc.Status,
		doc.Ticket, doc.Repo, ensureTags(doc.Tags), doc.PhaseCurrent, doc.PhaseTotal,
		ensureJSONMap(doc.Meta), ensureJSONMap(doc.Body), expected,
	)
	if err := row.Scan(&doc.UpdatedAt, &doc.Revision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.updateDocNoRow(ctx, doc.ID, expected)
		}
		return translateWriteErr("update document", err)
	}
	return nil
}

// updateDocNoRow disambiguates a zero-row UPDATE: no such id → ErrNotFound;
// the id exists but its revision moved past expected → *RevisionConflictError
// carrying the current revision so the caller can report what to re-read.
func (s *PostgresStore) updateDocNoRow(ctx context.Context, id string, expected *int) error {
	var current int
	err := s.pool.QueryRow(ctx, `SELECT revision FROM documents WHERE id = $1`, id).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update document revision check: %w", err)
	}
	if expected != nil {
		return &RevisionConflictError{
			DocumentID: id,
			Current:    current,
			ChangedIDs: changedTargetsSince(ctx, s, id, *expected),
		}
	}
	// Row exists and no expected-revision guard was set, yet nothing updated —
	// should not happen; surface a generic not-found rather than lie about success.
	return ErrNotFound
}

func (s *PostgresStore) ArchiveDocument(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE documents SET status = 'archived' WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("archive document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// queryBuilder accumulates WHERE clauses, args, and trailing ORDER /
// LIMIT / OFFSET. Used by both ListDocuments and SearchDocuments.
type queryBuilder struct {
	where []string
	args  []any
}

func (b *queryBuilder) addArg(v any) string {
	b.args = append(b.args, v)
	return fmt.Sprintf("$%d", len(b.args))
}

func (b *queryBuilder) addFilter(f Filter) {
	if f.Project != nil {
		b.where = append(b.where, "project = "+b.addArg(*f.Project))
	}
	if f.Type != nil {
		b.where = append(b.where, "type = "+b.addArg(*f.Type))
	}
	if f.Status != nil {
		b.where = append(b.where, "status = "+b.addArg(*f.Status))
	}
	if len(f.Tags) > 0 {
		b.where = append(b.where, "tags @> "+b.addArg(f.Tags))
	}
}

func (b *queryBuilder) whereClause() string {
	if len(b.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(b.where, " AND ")
}

func (b *queryBuilder) limitOffset(f Filter) string {
	out := ""
	if f.Limit > 0 {
		out += " LIMIT " + b.addArg(f.Limit)
	}
	if f.Offset > 0 {
		out += " OFFSET " + b.addArg(f.Offset)
	}
	return out
}

func (s *PostgresStore) ListDocuments(ctx context.Context, f Filter) ([]*models.Document, error) {
	b := &queryBuilder{}
	b.addFilter(f)
	q := `SELECT ` + documentColumns + ` FROM documents` +
		b.whereClause() +
		` ORDER BY updated_at DESC` +
		b.limitOffset(f)

	rows, err := s.pool.Query(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()
	return collectDocuments(rows)
}

func (s *PostgresStore) SearchDocuments(ctx context.Context, query string, f Filter) ([]*models.Document, error) {
	b := &queryBuilder{}
	// websearch_to_tsquery accepts google-style syntax: AND-by-default,
	// "quoted phrases", `or` keyword, and `-exclusion` — strictly more
	// useful than plainto_tsquery for LLM-generated queries while
	// preserving the same single-term behavior.
	qRef := b.addArg(query)
	b.where = append(b.where,
		"search_vector @@ websearch_to_tsquery('english', "+qRef+")")
	b.addFilter(f)

	sql := `SELECT ` + documentColumns + ` FROM documents` +
		b.whereClause() +
		` ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', ` + qRef + `)) DESC,` +
		` updated_at DESC` +
		b.limitOffset(f)

	rows, err := s.pool.Query(ctx, sql, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	defer rows.Close()
	return collectDocuments(rows)
}

func (s *PostgresStore) CreateProject(ctx context.Context, p *models.Project) error {
	const q = `
		INSERT INTO projects (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id, public_id, created_at`
	row := s.pool.QueryRow(ctx, q, p.Name, p.Slug, p.Description)
	if err := row.Scan(&p.ID, &p.PublicID, &p.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return ErrDuplicateProject
		}
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetProject(ctx context.Context, slug string) (*models.Project, error) {
	const q = `SELECT id, public_id, name, slug, description, created_at FROM projects WHERE slug = $1`
	p := &models.Project{}
	err := s.pool.QueryRow(ctx, q, slug).Scan(&p.ID, &p.PublicID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	return p, nil
}

func (s *PostgresStore) GetProjectByPublicID(ctx context.Context, publicID string) (*models.Project, error) {
	const q = `SELECT id, public_id, name, slug, description, created_at FROM projects WHERE public_id = $1`
	p := &models.Project{}
	err := s.pool.QueryRow(ctx, q, publicID).Scan(&p.ID, &p.PublicID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get project by public id: %w", err)
	}
	return p, nil
}

func (s *PostgresStore) SetMemory(ctx context.Context, m *models.Memory) error {
	const q = `
		INSERT INTO memories (scope, project, area, key, value)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (scope, project, area, key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = now()
		RETURNING id, updated_at`
	row := s.pool.QueryRow(ctx, q, m.Scope, m.Project, m.Area, m.Key, m.Value)
	if err := row.Scan(&m.ID, &m.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == pgForeignKeyViolation && pgErr.ConstraintName == "memories_project_fkey" {
			return ErrInvalidProject
		}
		return fmt.Errorf("set memory: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListMemory(ctx context.Context, f MemoryFilter) ([]*models.Memory, error) {
	var b queryBuilder
	if f.Scope != nil {
		b.where = append(b.where, "scope = "+b.addArg(string(*f.Scope)))
	}
	if f.Project != nil {
		b.where = append(b.where, "project = "+b.addArg(*f.Project))
	}
	if f.Area != nil {
		b.where = append(b.where, "area = "+b.addArg(*f.Area))
	}
	q := `SELECT id, scope, project, area, key, value, updated_at FROM memories`
	if len(b.where) > 0 {
		q += " WHERE " + strings.Join(b.where, " AND ")
	}
	q += " ORDER BY scope, project NULLS FIRST, area NULLS FIRST, key"

	rows, err := s.pool.Query(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list memory: %w", err)
	}
	defer rows.Close()
	out := []*models.Memory{}
	for rows.Next() {
		m := &models.Memory{}
		if err := rows.Scan(&m.ID, &m.Scope, &m.Project, &m.Area, &m.Key, &m.Value, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *PostgresStore) DeleteMemory(ctx context.Context, scope models.MemoryScope, project, area *string, key string) error {
	const q = `
		DELETE FROM memories
		WHERE scope = $1
		  AND project IS NOT DISTINCT FROM $2
		  AND area    IS NOT DISTINCT FROM $3
		  AND key = $4`
	tag, err := s.pool.Exec(ctx, q, scope, project, area, key)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SetEnv(ctx context.Context, e *models.EnvEntry) error {
	const q = `
		INSERT INTO env_entries (project, key, value, description)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (project, key)
		DO UPDATE SET value = EXCLUDED.value, description = EXCLUDED.description, updated_at = now()
		RETURNING id, updated_at`
	row := s.pool.QueryRow(ctx, q, e.Project, e.Key, e.Value, e.Description)
	if err := row.Scan(&e.ID, &e.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == pgForeignKeyViolation && pgErr.ConstraintName == "env_entries_project_fkey" {
			return ErrInvalidProject
		}
		return fmt.Errorf("set env: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListEnv(ctx context.Context, project string) ([]*models.EnvEntry, error) {
	const q = `SELECT id, project, key, value, description, updated_at
		FROM env_entries WHERE project = $1 ORDER BY key`
	rows, err := s.pool.Query(ctx, q, project)
	if err != nil {
		return nil, fmt.Errorf("list env: %w", err)
	}
	defer rows.Close()
	out := []*models.EnvEntry{}
	for rows.Next() {
		e := &models.EnvEntry{}
		if err := rows.Scan(&e.ID, &e.Project, &e.Key, &e.Value, &e.Description, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan env: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *PostgresStore) DeleteEnv(ctx context.Context, project, key string) error {
	const q = `DELETE FROM env_entries WHERE project = $1 AND key = $2`
	tag, err := s.pool.Exec(ctx, q, project, key)
	if err != nil {
		return fmt.Errorf("delete env: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListProjects(ctx context.Context) ([]*models.ProjectStats, error) {
	const q = `
		SELECT p.id, p.public_id, p.name, p.slug, p.description, p.created_at,
		       COUNT(d.id) FILTER (WHERE d.status = 'todo')        AS c_todo,
		       COUNT(d.id) FILTER (WHERE d.status = 'in-progress') AS c_in_progress,
		       COUNT(d.id) FILTER (WHERE d.status = 'complete')    AS c_complete,
		       COUNT(d.id) FILTER (WHERE d.status = 'blocked')     AS c_blocked,
		       COUNT(d.id) FILTER (WHERE d.status = 'archived')    AS c_archived,
		       COUNT(d.id)                                         AS c_total
		FROM projects p
		LEFT JOIN documents d ON d.project = p.slug
		GROUP BY p.id, p.public_id, p.name, p.slug, p.description, p.created_at
		ORDER BY p.name`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	out := []*models.ProjectStats{}
	for rows.Next() {
		ps := &models.ProjectStats{}
		if err := rows.Scan(
			&ps.ID, &ps.PublicID, &ps.Name, &ps.Slug, &ps.Description, &ps.CreatedAt,
			&ps.Counts.Todo, &ps.Counts.InProgress, &ps.Counts.Complete,
			&ps.Counts.Blocked, &ps.Counts.Archived, &ps.Counts.Total,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return out, nil
}

func collectDocuments(rows pgx.Rows) ([]*models.Document, error) {
	out := []*models.Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documents: %w", err)
	}
	return out, nil
}

// translateDecisionWriteErr maps the decisions FK violation to a typed
// error; anything else is wrapped. decisions has its own FK constraint
// name, so the documents-oriented translateWriteErr can't be reused.
func translateDecisionWriteErr(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == pgForeignKeyViolation && pgErr.ConstraintName == "decisions_project_fkey" {
		return ErrInvalidProject
	}
	return fmt.Errorf("%s: %w", op, err)
}

// decisionColumns is the SELECT list for every decision Get/List/Search.
// Keep in lockstep with scanDecision.
const decisionColumns = `
	id, public_id, title, project, decision, rationale, alternatives,
	consequences, status, created_at, updated_at`

func scanDecision(row pgx.Row) (*models.Decision, error) {
	d := &models.Decision{}
	err := row.Scan(
		&d.ID, &d.PublicID, &d.Title, &d.Project, &d.Decision, &d.Rationale,
		&d.Alternatives, &d.Consequences, &d.Status, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func collectDecisions(rows pgx.Rows) ([]*models.Decision, error) {
	out := []*models.Decision{}
	for rows.Next() {
		d, err := scanDecision(rows)
		if err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decisions: %w", err)
	}
	return out, nil
}

// decisionWhere appends the project/status constraints shared by
// ListDecisions and SearchDecisions.
func decisionWhere(b *queryBuilder, f DecisionFilter) {
	if f.Project != nil {
		b.where = append(b.where, "project = "+b.addArg(*f.Project))
	}
	if f.Status != nil {
		b.where = append(b.where, "status = "+b.addArg(string(*f.Status)))
	}
}

func decisionLimit(b *queryBuilder, f DecisionFilter) string {
	if f.Limit > 0 {
		return " LIMIT " + b.addArg(f.Limit)
	}
	return ""
}

func (s *PostgresStore) CreateDecision(ctx context.Context, d *models.Decision) error {
	const q = `
		INSERT INTO decisions (title, project, decision, rationale, alternatives, consequences, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, public_id, created_at, updated_at`
	row := s.pool.QueryRow(ctx, q, d.Title, d.Project, d.Decision, d.Rationale, d.Alternatives, d.Consequences, d.Status)
	if err := row.Scan(&d.ID, &d.PublicID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return translateDecisionWriteErr("create decision", err)
	}
	return nil
}

func (s *PostgresStore) GetDecision(ctx context.Context, id string) (*models.Decision, error) {
	q := `SELECT ` + decisionColumns + ` FROM decisions WHERE id = $1`
	d, err := scanDecision(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get decision: %w", err)
	}
	return d, nil
}

func (s *PostgresStore) GetDecisionByPublicID(ctx context.Context, publicID string) (*models.Decision, error) {
	q := `SELECT ` + decisionColumns + ` FROM decisions WHERE public_id = $1`
	d, err := scanDecision(s.pool.QueryRow(ctx, q, publicID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get decision by public id: %w", err)
	}
	return d, nil
}

func (s *PostgresStore) UpdateDecision(ctx context.Context, d *models.Decision) error {
	const q = `
		UPDATE decisions
		SET title = $2, project = $3, decision = $4, rationale = $5,
		    alternatives = $6, consequences = $7, status = $8
		WHERE id = $1
		RETURNING updated_at`
	row := s.pool.QueryRow(ctx, q, d.ID, d.Title, d.Project, d.Decision, d.Rationale, d.Alternatives, d.Consequences, d.Status)
	if err := row.Scan(&d.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return translateDecisionWriteErr("update decision", err)
	}
	return nil
}

func (s *PostgresStore) ListDecisions(ctx context.Context, f DecisionFilter) ([]*models.Decision, error) {
	b := &queryBuilder{}
	decisionWhere(b, f)
	q := `SELECT ` + decisionColumns + ` FROM decisions` +
		b.whereClause() +
		` ORDER BY created_at DESC` +
		decisionLimit(b, f)

	rows, err := s.pool.Query(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list decisions: %w", err)
	}
	defer rows.Close()
	return collectDecisions(rows)
}

func (s *PostgresStore) SearchDecisions(ctx context.Context, query string, f DecisionFilter) ([]*models.Decision, error) {
	b := &queryBuilder{}
	qRef := b.addArg(query)
	b.where = append(b.where,
		"search_vector @@ websearch_to_tsquery('english', "+qRef+")")
	decisionWhere(b, f)

	sql := `SELECT ` + decisionColumns + ` FROM decisions` +
		b.whereClause() +
		` ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', ` + qRef + `)) DESC,` +
		` created_at DESC` +
		decisionLimit(b, f)

	rows, err := s.pool.Query(ctx, sql, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search decisions: %w", err)
	}
	defer rows.Close()
	return collectDecisions(rows)
}

// translateSnippetWriteErr maps the snippets FK violation to a typed
// error; anything else is wrapped. snippets has its own FK constraint
// name, so the documents-oriented translateWriteErr can't be reused.
func translateSnippetWriteErr(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == pgForeignKeyViolation && pgErr.ConstraintName == "snippets_project_fkey" {
		return ErrInvalidProject
	}
	return fmt.Errorf("%s: %w", op, err)
}

// snippetColumns is the SELECT list for every snippet Get/List/Search.
// Keep in lockstep with scanSnippet.
const snippetColumns = `
	id, public_id, title, project, language, content, tags, description,
	created_at, updated_at`

func scanSnippet(row pgx.Row) (*models.Snippet, error) {
	sn := &models.Snippet{}
	err := row.Scan(
		&sn.ID, &sn.PublicID, &sn.Title, &sn.Project, &sn.Language, &sn.Content,
		&sn.Tags, &sn.Description, &sn.CreatedAt, &sn.UpdatedAt,
	)
	return sn, err
}

func collectSnippets(rows pgx.Rows) ([]*models.Snippet, error) {
	out := []*models.Snippet{}
	for rows.Next() {
		sn, err := scanSnippet(rows)
		if err != nil {
			return nil, fmt.Errorf("scan snippet: %w", err)
		}
		out = append(out, sn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snippets: %w", err)
	}
	return out, nil
}

// snippetWhere appends the project/language/tags constraints shared by
// ListSnippets and SearchSnippets.
func snippetWhere(b *queryBuilder, f SnippetFilter) {
	if f.Project != nil {
		b.where = append(b.where, "project = "+b.addArg(*f.Project))
	}
	if f.Language != nil {
		b.where = append(b.where, "language = "+b.addArg(*f.Language))
	}
	if len(f.Tags) > 0 {
		b.where = append(b.where, "tags @> "+b.addArg(f.Tags))
	}
}

func snippetLimit(b *queryBuilder, f SnippetFilter) string {
	if f.Limit > 0 {
		return " LIMIT " + b.addArg(f.Limit)
	}
	return ""
}

func (s *PostgresStore) CreateSnippet(ctx context.Context, sn *models.Snippet) error {
	const q = `
		INSERT INTO snippets (title, project, language, content, tags, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, public_id, created_at, updated_at`
	row := s.pool.QueryRow(ctx, q, sn.Title, sn.Project, sn.Language, sn.Content, ensureTags(sn.Tags), sn.Description)
	if err := row.Scan(&sn.ID, &sn.PublicID, &sn.CreatedAt, &sn.UpdatedAt); err != nil {
		return translateSnippetWriteErr("create snippet", err)
	}
	return nil
}

func (s *PostgresStore) GetSnippet(ctx context.Context, id string) (*models.Snippet, error) {
	q := `SELECT ` + snippetColumns + ` FROM snippets WHERE id = $1`
	sn, err := scanSnippet(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get snippet: %w", err)
	}
	return sn, nil
}

func (s *PostgresStore) GetSnippetByPublicID(ctx context.Context, publicID string) (*models.Snippet, error) {
	q := `SELECT ` + snippetColumns + ` FROM snippets WHERE public_id = $1`
	sn, err := scanSnippet(s.pool.QueryRow(ctx, q, publicID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get snippet by public id: %w", err)
	}
	return sn, nil
}

func (s *PostgresStore) UpdateSnippet(ctx context.Context, sn *models.Snippet) error {
	const q = `
		UPDATE snippets
		SET title = $2, project = $3, language = $4, content = $5,
		    tags = $6, description = $7
		WHERE id = $1
		RETURNING updated_at`
	row := s.pool.QueryRow(ctx, q, sn.ID, sn.Title, sn.Project, sn.Language, sn.Content, ensureTags(sn.Tags), sn.Description)
	if err := row.Scan(&sn.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return translateSnippetWriteErr("update snippet", err)
	}
	return nil
}

func (s *PostgresStore) ListSnippets(ctx context.Context, f SnippetFilter) ([]*models.Snippet, error) {
	b := &queryBuilder{}
	snippetWhere(b, f)
	q := `SELECT ` + snippetColumns + ` FROM snippets` +
		b.whereClause() +
		` ORDER BY created_at DESC` +
		snippetLimit(b, f)

	rows, err := s.pool.Query(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list snippets: %w", err)
	}
	defer rows.Close()
	return collectSnippets(rows)
}

func (s *PostgresStore) SearchSnippets(ctx context.Context, query string, f SnippetFilter) ([]*models.Snippet, error) {
	b := &queryBuilder{}
	qRef := b.addArg(query)
	b.where = append(b.where,
		"search_vector @@ websearch_to_tsquery('english', "+qRef+")")
	snippetWhere(b, f)

	sql := `SELECT ` + snippetColumns + ` FROM snippets` +
		b.whereClause() +
		` ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', ` + qRef + `)) DESC,` +
		` created_at DESC` +
		snippetLimit(b, f)

	rows, err := s.pool.Query(ctx, sql, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search snippets: %w", err)
	}
	defer rows.Close()
	return collectSnippets(rows)
}

// translateJournalWriteErr maps the journal FK violation to a typed
// error; anything else is wrapped. journal_entries has its own FK
// constraint name, so the documents-oriented translateWriteErr can't be
// reused.
func translateJournalWriteErr(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == pgForeignKeyViolation && pgErr.ConstraintName == "journal_entries_project_fkey" {
		return ErrInvalidProject
	}
	return fmt.Errorf("%s: %w", op, err)
}

// journalColumns is the SELECT list for every journal Get/List.
// Keep in lockstep with scanJournalEntry.
const journalColumns = `
	id, public_id, project, session_ref, summary, accomplished, deferred,
	created_at, updated_at`

func scanJournalEntry(row pgx.Row) (*models.JournalEntry, error) {
	e := &models.JournalEntry{}
	err := row.Scan(
		&e.ID, &e.PublicID, &e.Project, &e.SessionRef, &e.Summary,
		&e.Accomplished, &e.Deferred, &e.CreatedAt, &e.UpdatedAt,
	)
	return e, err
}

func collectJournalEntries(rows pgx.Rows) ([]*models.JournalEntry, error) {
	out := []*models.JournalEntry{}
	for rows.Next() {
		e, err := scanJournalEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan journal entry: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate journal entries: %w", err)
	}
	return out, nil
}

// journalWhere appends the project/since constraints shared by any journal
// list query.
func journalWhere(b *queryBuilder, f JournalFilter) {
	if f.Project != nil {
		b.where = append(b.where, "project = "+b.addArg(*f.Project))
	}
	if f.Since != nil {
		b.where = append(b.where, "created_at >= "+b.addArg(*f.Since))
	}
}

func journalLimit(b *queryBuilder, f JournalFilter) string {
	if f.Limit > 0 {
		return " LIMIT " + b.addArg(f.Limit)
	}
	return ""
}

func (s *PostgresStore) CreateJournalEntry(ctx context.Context, e *models.JournalEntry) error {
	const q = `
		INSERT INTO journal_entries (project, session_ref, summary, accomplished, deferred)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, public_id, created_at, updated_at`
	row := s.pool.QueryRow(ctx, q, e.Project, e.SessionRef, e.Summary, ensureTags(e.Accomplished), ensureTags(e.Deferred))
	if err := row.Scan(&e.ID, &e.PublicID, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return translateJournalWriteErr("create journal entry", err)
	}
	return nil
}

func (s *PostgresStore) GetJournalEntry(ctx context.Context, id string) (*models.JournalEntry, error) {
	q := `SELECT ` + journalColumns + ` FROM journal_entries WHERE id = $1`
	e, err := scanJournalEntry(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get journal entry: %w", err)
	}
	return e, nil
}

func (s *PostgresStore) GetJournalEntryByPublicID(ctx context.Context, publicID string) (*models.JournalEntry, error) {
	q := `SELECT ` + journalColumns + ` FROM journal_entries WHERE public_id = $1`
	e, err := scanJournalEntry(s.pool.QueryRow(ctx, q, publicID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get journal entry by public id: %w", err)
	}
	return e, nil
}

func (s *PostgresStore) UpdateJournalEntry(ctx context.Context, e *models.JournalEntry) error {
	const q = `
		UPDATE journal_entries
		SET project = $2, session_ref = $3, summary = $4,
		    accomplished = $5, deferred = $6
		WHERE id = $1
		RETURNING updated_at`
	row := s.pool.QueryRow(ctx, q, e.ID, e.Project, e.SessionRef, e.Summary, ensureTags(e.Accomplished), ensureTags(e.Deferred))
	if err := row.Scan(&e.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return translateJournalWriteErr("update journal entry", err)
	}
	return nil
}

func (s *PostgresStore) ListJournalEntries(ctx context.Context, f JournalFilter) ([]*models.JournalEntry, error) {
	b := &queryBuilder{}
	journalWhere(b, f)
	q := `SELECT ` + journalColumns + ` FROM journal_entries` +
		b.whereClause() +
		` ORDER BY created_at DESC` +
		journalLimit(b, f)

	rows, err := s.pool.Query(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}
	defer rows.Close()
	return collectJournalEntries(rows)
}

// translateSolutionWriteErr maps the solutions FK violation to a typed
// error; anything else is wrapped. solutions has its own FK constraint
// name, so the documents-oriented translateWriteErr can't be reused.
func translateSolutionWriteErr(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == pgForeignKeyViolation && pgErr.ConstraintName == "solutions_project_fkey" {
		return ErrInvalidProject
	}
	return fmt.Errorf("%s: %w", op, err)
}

// solutionColumns is the SELECT list for every solution Get/List/Search.
// Keep in lockstep with scanSolution.
const solutionColumns = `
	id, public_id, project, error_description, solution, tags, source_url,
	created_at, updated_at`

func scanSolution(row pgx.Row) (*models.Solution, error) {
	sol := &models.Solution{}
	err := row.Scan(
		&sol.ID, &sol.PublicID, &sol.Project, &sol.ErrorDescription, &sol.Solution,
		&sol.Tags, &sol.SourceURL, &sol.CreatedAt, &sol.UpdatedAt,
	)
	return sol, err
}

func collectSolutions(rows pgx.Rows) ([]*models.Solution, error) {
	out := []*models.Solution{}
	for rows.Next() {
		sol, err := scanSolution(rows)
		if err != nil {
			return nil, fmt.Errorf("scan solution: %w", err)
		}
		out = append(out, sol)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate solutions: %w", err)
	}
	return out, nil
}

// solutionWhere appends the project/tag constraints shared by ListSolutions
// and SearchSolutions.
func solutionWhere(b *queryBuilder, f SolutionFilter) {
	if f.Project != nil {
		b.where = append(b.where, "project = "+b.addArg(*f.Project))
	}
	if len(f.Tags) > 0 {
		b.where = append(b.where, "tags @> "+b.addArg(f.Tags))
	}
}

func solutionLimit(b *queryBuilder, f SolutionFilter) string {
	if f.Limit > 0 {
		return " LIMIT " + b.addArg(f.Limit)
	}
	return ""
}

func (s *PostgresStore) CreateSolution(ctx context.Context, sol *models.Solution) error {
	const q = `
		INSERT INTO solutions (project, error_description, solution, tags, source_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, public_id, created_at, updated_at`
	row := s.pool.QueryRow(ctx, q, sol.Project, sol.ErrorDescription, sol.Solution, ensureTags(sol.Tags), sol.SourceURL)
	if err := row.Scan(&sol.ID, &sol.PublicID, &sol.CreatedAt, &sol.UpdatedAt); err != nil {
		return translateSolutionWriteErr("create solution", err)
	}
	return nil
}

func (s *PostgresStore) GetSolution(ctx context.Context, id string) (*models.Solution, error) {
	q := `SELECT ` + solutionColumns + ` FROM solutions WHERE id = $1`
	sol, err := scanSolution(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get solution: %w", err)
	}
	return sol, nil
}

func (s *PostgresStore) GetSolutionByPublicID(ctx context.Context, publicID string) (*models.Solution, error) {
	q := `SELECT ` + solutionColumns + ` FROM solutions WHERE public_id = $1`
	sol, err := scanSolution(s.pool.QueryRow(ctx, q, publicID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get solution by public id: %w", err)
	}
	return sol, nil
}

func (s *PostgresStore) UpdateSolution(ctx context.Context, sol *models.Solution) error {
	const q = `
		UPDATE solutions
		SET project = $2, error_description = $3, solution = $4,
		    tags = $5, source_url = $6
		WHERE id = $1
		RETURNING updated_at`
	row := s.pool.QueryRow(ctx, q, sol.ID, sol.Project, sol.ErrorDescription, sol.Solution, ensureTags(sol.Tags), sol.SourceURL)
	if err := row.Scan(&sol.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return translateSolutionWriteErr("update solution", err)
	}
	return nil
}

func (s *PostgresStore) ListSolutions(ctx context.Context, f SolutionFilter) ([]*models.Solution, error) {
	b := &queryBuilder{}
	solutionWhere(b, f)
	q := `SELECT ` + solutionColumns + ` FROM solutions` +
		b.whereClause() +
		` ORDER BY created_at DESC` +
		solutionLimit(b, f)

	rows, err := s.pool.Query(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list solutions: %w", err)
	}
	defer rows.Close()
	return collectSolutions(rows)
}

func (s *PostgresStore) SearchSolutions(ctx context.Context, query string, f SolutionFilter) ([]*models.Solution, error) {
	b := &queryBuilder{}
	qRef := b.addArg(query)
	b.where = append(b.where,
		"search_vector @@ websearch_to_tsquery('english', "+qRef+")")
	solutionWhere(b, f)

	sql := `SELECT ` + solutionColumns + ` FROM solutions` +
		b.whereClause() +
		` ORDER BY ts_rank(search_vector, websearch_to_tsquery('english', ` + qRef + `)) DESC,` +
		` created_at DESC` +
		solutionLimit(b, f)

	rows, err := s.pool.Query(ctx, sql, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search solutions: %w", err)
	}
	defer rows.Close()
	return collectSolutions(rows)
}
