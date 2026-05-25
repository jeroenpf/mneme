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

	"github.com/jeroenpfeil/mneme/internal/models"
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
}

// New builds a PostgresStore from a DSN. It opens the pool, applies
// the Pi-friendly limits used by Mneme, and pings before returning.
func New(ctx context.Context, dsn string) (*PostgresStore, error) {
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
	id, title, project, category, type, status,
	ticket, repo, tags, phase_current, phase_total,
	meta, body, created_at, updated_at`

func scanDocument(row pgx.Row) (*models.Document, error) {
	d := &models.Document{}
	err := row.Scan(
		&d.ID, &d.Title, &d.Project, &d.Category, &d.Type, &d.Status,
		&d.Ticket, &d.Repo, &d.Tags, &d.PhaseCurrent, &d.PhaseTotal,
		&d.Meta, &d.Body, &d.CreatedAt, &d.UpdatedAt,
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
		RETURNING created_at, updated_at`
	row := s.pool.QueryRow(ctx, q,
		doc.ID, doc.Title, doc.Project, doc.Category, doc.Type, doc.Status,
		doc.Ticket, doc.Repo, ensureTags(doc.Tags), doc.PhaseCurrent, doc.PhaseTotal,
		ensureJSONMap(doc.Meta), ensureJSONMap(doc.Body),
	)
	if err := row.Scan(&doc.CreatedAt, &doc.UpdatedAt); err != nil {
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

func (s *PostgresStore) UpdateDocument(ctx context.Context, doc *models.Document) error {
	const q = `
		UPDATE documents SET
			title = $2, project = $3, category = $4, type = $5, status = $6,
			ticket = $7, repo = $8, tags = $9, phase_current = $10, phase_total = $11,
			meta = $12, body = $13
		WHERE id = $1
		RETURNING updated_at`
	row := s.pool.QueryRow(ctx, q,
		doc.ID, doc.Title, doc.Project, doc.Category, doc.Type, doc.Status,
		doc.Ticket, doc.Repo, ensureTags(doc.Tags), doc.PhaseCurrent, doc.PhaseTotal,
		ensureJSONMap(doc.Meta), ensureJSONMap(doc.Body),
	)
	if err := row.Scan(&doc.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return translateWriteErr("update document", err)
	}
	return nil
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
	// The tsquery arg is added first so $1 is stable for both the WHERE
	// clause and the ORDER BY ts_rank.
	qRef := b.addArg(query)
	b.where = append(b.where,
		"search_vector @@ plainto_tsquery('english', "+qRef+")")
	b.addFilter(f)

	sql := `SELECT ` + documentColumns + ` FROM documents` +
		b.whereClause() +
		` ORDER BY ts_rank(search_vector, plainto_tsquery('english', ` + qRef + `)) DESC,` +
		` updated_at DESC` +
		b.limitOffset(f)

	rows, err := s.pool.Query(ctx, sql, b.args...)
	if err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	defer rows.Close()
	return collectDocuments(rows)
}

func (s *PostgresStore) ListProjects(ctx context.Context) ([]*models.ProjectStats, error) {
	const q = `
		SELECT p.id, p.name, p.slug, p.description, p.created_at,
		       COUNT(d.id) FILTER (WHERE d.status = 'todo')        AS c_todo,
		       COUNT(d.id) FILTER (WHERE d.status = 'in-progress') AS c_in_progress,
		       COUNT(d.id) FILTER (WHERE d.status = 'complete')    AS c_complete,
		       COUNT(d.id) FILTER (WHERE d.status = 'blocked')     AS c_blocked,
		       COUNT(d.id) FILTER (WHERE d.status = 'archived')    AS c_archived,
		       COUNT(d.id)                                         AS c_total
		FROM projects p
		LEFT JOIN documents d ON d.project = p.slug
		GROUP BY p.id, p.name, p.slug, p.description, p.created_at
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
			&ps.ID, &ps.Name, &ps.Slug, &ps.Description, &ps.CreatedAt,
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
