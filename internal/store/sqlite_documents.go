package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	idspkg "github.com/jeroenpf/mneme/internal/ids"
	"github.com/jeroenpf/mneme/internal/models"
)

// translateSQLiteDocErr maps SQLite constraint failures on documents writes to
// the store's typed errors: the project FK to ErrInvalidProject, a duplicate
// primary key to ErrDuplicateID. Mirrors the Postgres translateWriteErr.
func translateSQLiteDocErr(op string, err error) error {
	switch {
	case isSQLiteFKViolation(err):
		return ErrInvalidProject
	case isSQLiteUniqueViolation(err):
		return ErrDuplicateID
	}
	return fmt.Errorf("%s: %w", op, err)
}

// scanSQLiteDocument reads a documents row, decoding the JSON TEXT columns
// (tags/meta/body) that Postgres stores natively. Timestamps come back as
// time.Time via modernc's TIMESTAMP conversion.
func scanSQLiteDocument(row rowScanner) (*models.Document, error) {
	d := &models.Document{}
	var tags, meta, body string
	err := row.Scan(
		&d.ID, &d.PublicID, &d.Title, &d.Project, &d.Category, &d.Type, &d.Status,
		&d.Ticket, &d.Repo, &tags, &d.PhaseCurrent, &d.PhaseTotal, &meta, &body,
		&d.CreatedAt, &d.UpdatedAt, &d.Revision,
	)
	if err != nil {
		return d, err
	}
	if d.Tags, err = scanJSONArray(tags); err != nil {
		return nil, err
	}
	if d.Meta, err = scanJSONObject(meta); err != nil {
		return nil, err
	}
	if d.Body, err = scanJSONObject(body); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *SQLiteStore) CreateDocument(ctx context.Context, doc *models.Document) error {
	pub, err := mintPublicID(idspkg.KindDocument)
	if err != nil {
		return err
	}
	meta, err := jsonObject(doc.Meta)
	if err != nil {
		return err
	}
	body, err := jsonObject(doc.Body)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO documents (
			id, public_id, title, project, category, type, status,
			ticket, repo, tags, phase_current, phase_total, meta, body
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		RETURNING created_at, updated_at, revision`
	err = s.db.QueryRowContext(ctx, q,
		doc.ID, pub, doc.Title, doc.Project, doc.Category, doc.Type, doc.Status,
		doc.Ticket, doc.Repo, jsonArray(doc.Tags), doc.PhaseCurrent, doc.PhaseTotal,
		meta, body,
	).Scan(&doc.CreatedAt, &doc.UpdatedAt, &doc.Revision)
	if err != nil {
		return translateSQLiteDocErr("insert document", err)
	}
	doc.PublicID = pub
	return nil
}

func (s *SQLiteStore) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	q := `SELECT ` + documentColumns + ` FROM documents WHERE id = ?`
	doc, err := scanSQLiteDocument(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return doc, nil
}

func (s *SQLiteStore) GetDocumentByPublicID(ctx context.Context, publicID string) (*models.Document, error) {
	q := `SELECT ` + documentColumns + ` FROM documents WHERE public_id = ?`
	doc, err := scanSQLiteDocument(s.db.QueryRowContext(ctx, q, publicID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get document by public id: %w", err)
	}
	return doc, nil
}

func (s *SQLiteStore) UpdateDocument(ctx context.Context, doc *models.Document, expected *int) error {
	meta, err := jsonObject(doc.Meta)
	if err != nil {
		return err
	}
	body, err := jsonObject(doc.Body)
	if err != nil {
		return err
	}
	// updated_at is set explicitly so RETURNING reflects the new value (SQLite
	// RETURNING sees pre-trigger row state); the set_updated_at trigger's guard
	// then no-ops because NEW.updated_at != OLD.updated_at. revision is bumped
	// in the same statement, guarded by the optional expected-revision check.
	const q = `
		UPDATE documents SET
			title = ?, project = ?, category = ?, type = ?, status = ?,
			ticket = ?, repo = ?, tags = ?, phase_current = ?, phase_total = ?,
			meta = ?, body = ?, revision = revision + 1,
			updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now')
		WHERE id = ? AND (? IS NULL OR revision = ?)
		RETURNING updated_at, revision`
	err = s.db.QueryRowContext(ctx, q,
		doc.Title, doc.Project, doc.Category, doc.Type, doc.Status,
		doc.Ticket, doc.Repo, jsonArray(doc.Tags), doc.PhaseCurrent, doc.PhaseTotal,
		meta, body, doc.ID, expected, expected,
	).Scan(&doc.UpdatedAt, &doc.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return s.updateDocNoRow(ctx, doc.ID, expected)
	}
	if err != nil {
		return translateSQLiteDocErr("update document", err)
	}
	return nil
}

// updateDocNoRow disambiguates a zero-row UPDATE: no such id → ErrNotFound;
// the id exists but its revision moved past expected → *RevisionConflictError.
func (s *SQLiteStore) updateDocNoRow(ctx context.Context, id string, expected *int) error {
	var current int
	err := s.db.QueryRowContext(ctx, `SELECT revision FROM documents WHERE id = ?`, id).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
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
	return ErrNotFound
}

func (s *SQLiteStore) ListDocuments(ctx context.Context, f Filter) ([]*models.Document, error) {
	var b sqliteQB
	if f.Project != nil {
		b.add("project = ?", *f.Project)
	}
	if f.Type != nil {
		b.add("type = ?", *f.Type)
	}
	if f.Status != nil {
		b.add("status = ?", *f.Status)
	}
	for _, tag := range f.Tags {
		b.add("EXISTS (SELECT 1 FROM json_each(documents.tags) WHERE value = ?)", tag)
	}
	q := `SELECT ` + documentColumns + ` FROM documents` +
		b.whereClause() + ` ORDER BY updated_at DESC` + b.limitOffset(f)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()
	return collectSQLiteDocuments(rows)
}

func collectSQLiteDocuments(rows *sql.Rows) ([]*models.Document, error) {
	out := []*models.Document{}
	for rows.Next() {
		d, err := scanSQLiteDocument(rows)
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
