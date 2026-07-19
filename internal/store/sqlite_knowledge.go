package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	idspkg "github.com/jeroenpfeil/mneme/internal/ids"
	"github.com/jeroenpfeil/mneme/internal/models"
)

// translateSQLiteProjectFKErr maps a project foreign-key failure to
// ErrInvalidProject; anything else is wrapped. Shared by the four
// project-scoped knowledge types, whose only translatable constraint is the
// project FK (their ids are app-minted UUIDs, so a PK collision never occurs).
func translateSQLiteProjectFKErr(op string, err error) error {
	if isSQLiteFKViolation(err) {
		return ErrInvalidProject
	}
	return fmt.Errorf("%s: %w", op, err)
}

// nowExpr stamps a column with the same millisecond-precision UTC format the
// schema defaults use, so explicit writes and DB defaults are comparable.
const nowExpr = `strftime('%Y-%m-%d %H:%M:%f', 'now')`

// --- Decisions -------------------------------------------------------------

func scanSQLiteDecision(row rowScanner) (*models.Decision, error) {
	d := &models.Decision{}
	err := row.Scan(
		&d.ID, &d.PublicID, &d.Title, &d.Project, &d.Decision, &d.Rationale,
		&d.Alternatives, &d.Consequences, &d.Status, &d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func (s *SQLiteStore) CreateDecision(ctx context.Context, d *models.Decision) error {
	d.ID = newUUID()
	pub, err := mintPublicID(idspkg.KindDecision)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO decisions (id, public_id, title, project, decision, rationale, alternatives, consequences, status)
		VALUES (?,?,?,?,?,?,?,?,?)
		RETURNING created_at, updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		d.ID, pub, d.Title, d.Project, d.Decision, d.Rationale, d.Alternatives, d.Consequences, d.Status,
	).Scan(&d.CreatedAt, &d.UpdatedAt); err != nil {
		return translateSQLiteProjectFKErr("create decision", err)
	}
	d.PublicID = pub
	return nil
}

func (s *SQLiteStore) GetDecision(ctx context.Context, id string) (*models.Decision, error) {
	q := `SELECT ` + decisionColumns + ` FROM decisions WHERE id = ?`
	d, err := scanSQLiteDecision(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get decision: %w", err)
	}
	return d, nil
}

func (s *SQLiteStore) GetDecisionByPublicID(ctx context.Context, publicID string) (*models.Decision, error) {
	q := `SELECT ` + decisionColumns + ` FROM decisions WHERE public_id = ?`
	d, err := scanSQLiteDecision(s.db.QueryRowContext(ctx, q, publicID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get decision by public id: %w", err)
	}
	return d, nil
}

func (s *SQLiteStore) UpdateDecision(ctx context.Context, d *models.Decision) error {
	const q = `
		UPDATE decisions
		SET title = ?, project = ?, decision = ?, rationale = ?,
		    alternatives = ?, consequences = ?, status = ?, updated_at = ` + nowExpr + `
		WHERE id = ?
		RETURNING updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		d.Title, d.Project, d.Decision, d.Rationale, d.Alternatives, d.Consequences, d.Status, d.ID,
	).Scan(&d.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return translateSQLiteProjectFKErr("update decision", err)
	}
	return nil
}

func (s *SQLiteStore) ListDecisions(ctx context.Context, f DecisionFilter) ([]*models.Decision, error) {
	var b sqliteQB
	if f.Project != nil {
		b.add("project = ?", *f.Project)
	}
	if f.Status != nil {
		b.add("status = ?", string(*f.Status))
	}
	q := `SELECT ` + decisionColumns + ` FROM decisions` +
		b.whereClause() + ` ORDER BY created_at DESC` + b.limit(f.Limit)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list decisions: %w", err)
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

// --- Snippets --------------------------------------------------------------

func scanSQLiteSnippet(row rowScanner) (*models.Snippet, error) {
	sn := &models.Snippet{}
	var tags string
	err := row.Scan(
		&sn.ID, &sn.PublicID, &sn.Title, &sn.Project, &sn.Language, &sn.Content,
		&tags, &sn.Description, &sn.CreatedAt, &sn.UpdatedAt,
	)
	if err != nil {
		return sn, err
	}
	if sn.Tags, err = scanJSONArray(tags); err != nil {
		return nil, err
	}
	return sn, nil
}

func (s *SQLiteStore) CreateSnippet(ctx context.Context, sn *models.Snippet) error {
	sn.ID = newUUID()
	pub, err := mintPublicID(idspkg.KindSnippet)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO snippets (id, public_id, title, project, language, content, tags, description)
		VALUES (?,?,?,?,?,?,?,?)
		RETURNING created_at, updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		sn.ID, pub, sn.Title, sn.Project, sn.Language, sn.Content, jsonArray(sn.Tags), sn.Description,
	).Scan(&sn.CreatedAt, &sn.UpdatedAt); err != nil {
		return translateSQLiteProjectFKErr("create snippet", err)
	}
	sn.PublicID = pub
	return nil
}

func (s *SQLiteStore) GetSnippet(ctx context.Context, id string) (*models.Snippet, error) {
	q := `SELECT ` + snippetColumns + ` FROM snippets WHERE id = ?`
	sn, err := scanSQLiteSnippet(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get snippet: %w", err)
	}
	return sn, nil
}

func (s *SQLiteStore) GetSnippetByPublicID(ctx context.Context, publicID string) (*models.Snippet, error) {
	q := `SELECT ` + snippetColumns + ` FROM snippets WHERE public_id = ?`
	sn, err := scanSQLiteSnippet(s.db.QueryRowContext(ctx, q, publicID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get snippet by public id: %w", err)
	}
	return sn, nil
}

func (s *SQLiteStore) UpdateSnippet(ctx context.Context, sn *models.Snippet) error {
	const q = `
		UPDATE snippets
		SET title = ?, project = ?, language = ?, content = ?,
		    tags = ?, description = ?, updated_at = ` + nowExpr + `
		WHERE id = ?
		RETURNING updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		sn.Title, sn.Project, sn.Language, sn.Content, jsonArray(sn.Tags), sn.Description, sn.ID,
	).Scan(&sn.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return translateSQLiteProjectFKErr("update snippet", err)
	}
	return nil
}

func (s *SQLiteStore) ListSnippets(ctx context.Context, f SnippetFilter) ([]*models.Snippet, error) {
	var b sqliteQB
	if f.Project != nil {
		b.add("project = ?", *f.Project)
	}
	if f.Language != nil {
		b.add("language = ?", *f.Language)
	}
	for _, tag := range f.Tags {
		b.add("EXISTS (SELECT 1 FROM json_each(snippets.tags) WHERE value = ?)", tag)
	}
	q := `SELECT ` + snippetColumns + ` FROM snippets` +
		b.whereClause() + ` ORDER BY created_at DESC` + b.limit(f.Limit)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list snippets: %w", err)
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

// --- Journal ---------------------------------------------------------------

func scanSQLiteJournal(row rowScanner) (*models.JournalEntry, error) {
	e := &models.JournalEntry{}
	var acc, def string
	err := row.Scan(
		&e.ID, &e.PublicID, &e.Project, &e.SessionRef, &e.Summary,
		&acc, &def, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return e, err
	}
	if e.Accomplished, err = scanJSONArray(acc); err != nil {
		return nil, err
	}
	if e.Deferred, err = scanJSONArray(def); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *SQLiteStore) CreateJournalEntry(ctx context.Context, e *models.JournalEntry) error {
	e.ID = newUUID()
	pub, err := mintPublicID(idspkg.KindJournal)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO journal_entries (id, public_id, project, session_ref, summary, accomplished, deferred)
		VALUES (?,?,?,?,?,?,?)
		RETURNING created_at, updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		e.ID, pub, e.Project, e.SessionRef, e.Summary, jsonArray(e.Accomplished), jsonArray(e.Deferred),
	).Scan(&e.CreatedAt, &e.UpdatedAt); err != nil {
		return translateSQLiteProjectFKErr("create journal entry", err)
	}
	e.PublicID = pub
	return nil
}

func (s *SQLiteStore) GetJournalEntry(ctx context.Context, id string) (*models.JournalEntry, error) {
	q := `SELECT ` + journalColumns + ` FROM journal_entries WHERE id = ?`
	e, err := scanSQLiteJournal(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get journal entry: %w", err)
	}
	return e, nil
}

func (s *SQLiteStore) GetJournalEntryByPublicID(ctx context.Context, publicID string) (*models.JournalEntry, error) {
	q := `SELECT ` + journalColumns + ` FROM journal_entries WHERE public_id = ?`
	e, err := scanSQLiteJournal(s.db.QueryRowContext(ctx, q, publicID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get journal entry by public id: %w", err)
	}
	return e, nil
}

func (s *SQLiteStore) UpdateJournalEntry(ctx context.Context, e *models.JournalEntry) error {
	const q = `
		UPDATE journal_entries
		SET project = ?, session_ref = ?, summary = ?,
		    accomplished = ?, deferred = ?, updated_at = ` + nowExpr + `
		WHERE id = ?
		RETURNING updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		e.Project, e.SessionRef, e.Summary, jsonArray(e.Accomplished), jsonArray(e.Deferred), e.ID,
	).Scan(&e.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return translateSQLiteProjectFKErr("update journal entry", err)
	}
	return nil
}

func (s *SQLiteStore) ListJournalEntries(ctx context.Context, f JournalFilter) ([]*models.JournalEntry, error) {
	var b sqliteQB
	if f.Project != nil {
		b.add("project = ?", *f.Project)
	}
	if f.Since != nil {
		// Compare against the stored fixed-width millisecond format so the
		// string comparison is chronological (binding a time.Time would use a
		// different serialization).
		b.add("created_at >= ?", f.Since.UTC().Format("2006-01-02 15:04:05.000"))
	}
	q := `SELECT ` + journalColumns + ` FROM journal_entries` +
		b.whereClause() + ` ORDER BY created_at DESC` + b.limit(f.Limit)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}
	defer rows.Close()
	out := []*models.JournalEntry{}
	for rows.Next() {
		e, err := scanSQLiteJournal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan journal entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// --- Solutions -------------------------------------------------------------

func scanSQLiteSolution(row rowScanner) (*models.Solution, error) {
	sol := &models.Solution{}
	var tags string
	err := row.Scan(
		&sol.ID, &sol.PublicID, &sol.Project, &sol.ErrorDescription, &sol.Solution,
		&tags, &sol.SourceURL, &sol.CreatedAt, &sol.UpdatedAt,
	)
	if err != nil {
		return sol, err
	}
	if sol.Tags, err = scanJSONArray(tags); err != nil {
		return nil, err
	}
	return sol, nil
}

func (s *SQLiteStore) CreateSolution(ctx context.Context, sol *models.Solution) error {
	sol.ID = newUUID()
	pub, err := mintPublicID(idspkg.KindSolution)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO solutions (id, public_id, project, error_description, solution, tags, source_url)
		VALUES (?,?,?,?,?,?,?)
		RETURNING created_at, updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		sol.ID, pub, sol.Project, sol.ErrorDescription, sol.Solution, jsonArray(sol.Tags), sol.SourceURL,
	).Scan(&sol.CreatedAt, &sol.UpdatedAt); err != nil {
		return translateSQLiteProjectFKErr("create solution", err)
	}
	sol.PublicID = pub
	return nil
}

func (s *SQLiteStore) GetSolution(ctx context.Context, id string) (*models.Solution, error) {
	q := `SELECT ` + solutionColumns + ` FROM solutions WHERE id = ?`
	sol, err := scanSQLiteSolution(s.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get solution: %w", err)
	}
	return sol, nil
}

func (s *SQLiteStore) GetSolutionByPublicID(ctx context.Context, publicID string) (*models.Solution, error) {
	q := `SELECT ` + solutionColumns + ` FROM solutions WHERE public_id = ?`
	sol, err := scanSQLiteSolution(s.db.QueryRowContext(ctx, q, publicID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get solution by public id: %w", err)
	}
	return sol, nil
}

func (s *SQLiteStore) UpdateSolution(ctx context.Context, sol *models.Solution) error {
	const q = `
		UPDATE solutions
		SET project = ?, error_description = ?, solution = ?,
		    tags = ?, source_url = ?, updated_at = ` + nowExpr + `
		WHERE id = ?
		RETURNING updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		sol.Project, sol.ErrorDescription, sol.Solution, jsonArray(sol.Tags), sol.SourceURL, sol.ID,
	).Scan(&sol.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return translateSQLiteProjectFKErr("update solution", err)
	}
	return nil
}

func (s *SQLiteStore) ListSolutions(ctx context.Context, f SolutionFilter) ([]*models.Solution, error) {
	var b sqliteQB
	if f.Project != nil {
		b.add("project = ?", *f.Project)
	}
	for _, tag := range f.Tags {
		b.add("EXISTS (SELECT 1 FROM json_each(solutions.tags) WHERE value = ?)", tag)
	}
	q := `SELECT ` + solutionColumns + ` FROM solutions` +
		b.whereClause() + ` ORDER BY created_at DESC` + b.limit(f.Limit)

	rows, err := s.db.QueryContext(ctx, q, b.args...)
	if err != nil {
		return nil, fmt.Errorf("list solutions: %w", err)
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
