package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	idspkg "github.com/jeroenpf/mneme/internal/ids"
	"github.com/jeroenpf/mneme/internal/models"
)

func (s *SQLiteStore) CreateProject(ctx context.Context, p *models.Project) error {
	p.ID = newUUID()
	pub, err := mintPublicID(idspkg.KindProject)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO projects (id, public_id, name, slug, description)
		VALUES (?,?,?,?,?)
		RETURNING created_at`
	if err := s.db.QueryRowContext(ctx, q, p.ID, pub, p.Name, p.Slug, p.Description).Scan(&p.CreatedAt); err != nil {
		if isSQLiteUniqueViolation(err) {
			return ErrDuplicateProject
		}
		return fmt.Errorf("insert project: %w", err)
	}
	p.PublicID = pub
	return nil
}

func (s *SQLiteStore) GetProject(ctx context.Context, slug string) (*models.Project, error) {
	const q = `SELECT id, public_id, name, slug, description, created_at FROM projects WHERE slug = ?`
	p := &models.Project{}
	err := s.db.QueryRowContext(ctx, q, slug).Scan(
		&p.ID, &p.PublicID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return p, nil
}

func (s *SQLiteStore) GetProjectByPublicID(ctx context.Context, publicID string) (*models.Project, error) {
	const q = `SELECT id, public_id, name, slug, description, created_at FROM projects WHERE public_id = ?`
	p := &models.Project{}
	err := s.db.QueryRowContext(ctx, q, publicID).Scan(
		&p.ID, &p.PublicID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project by public id: %w", err)
	}
	return p, nil
}

func (s *SQLiteStore) ListProjects(ctx context.Context) ([]*models.ProjectStats, error) {
	// COUNT(...) FILTER mirrors the Postgres query; LEFT JOIN + COUNT(d.id)
	// yields 0 for projects with no documents (NULLs are not counted).
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
	rows, err := s.db.QueryContext(ctx, q)
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

// --- Memory ----------------------------------------------------------------

func (s *SQLiteStore) SetMemory(ctx context.Context, m *models.Memory) error {
	// The conflict target matches the memories_identity expression index (which
	// COALESCEs the nullable scope columns), reproducing the Postgres
	// NULLS-NOT-DISTINCT upsert. RETURNING id yields the existing row's id on
	// update, the freshly-minted one on insert.
	const q = `
		INSERT INTO memories (id, scope, project, area, key, value, updated_at)
		VALUES (?,?,?,?,?,?, ` + nowExpr + `)
		ON CONFLICT (scope, COALESCE(project, ''), COALESCE(area, ''), key)
		DO UPDATE SET value = excluded.value, updated_at = ` + nowExpr + `
		RETURNING id, updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		newUUID(), m.Scope, m.Project, m.Area, m.Key, m.Value,
	).Scan(&m.ID, &m.UpdatedAt); err != nil {
		if isSQLiteFKViolation(err) {
			return ErrInvalidProject
		}
		return fmt.Errorf("set memory: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListMemory(ctx context.Context, f MemoryFilter) ([]*models.Memory, error) {
	var b sqliteQB
	if f.Scope != nil {
		b.add("scope = ?", string(*f.Scope))
	}
	if f.Project != nil {
		b.add("project = ?", *f.Project)
	}
	if f.Area != nil {
		b.add("area = ?", *f.Area)
	}
	// SQLite sorts NULLs first in ASC order, matching the Postgres
	// "project NULLS FIRST, area NULLS FIRST" ordering.
	q := `SELECT id, scope, project, area, key, value, updated_at FROM memories` +
		b.whereClause() + ` ORDER BY scope, project, area, key`

	rows, err := s.db.QueryContext(ctx, q, b.args...)
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

func (s *SQLiteStore) DeleteMemory(ctx context.Context, scope models.MemoryScope, project, area *string, key string) error {
	// SQLite's IS operator is null-safe, so a nil project/area binds to NULL and
	// matches NULL rows (the Postgres path uses IS NOT DISTINCT FROM).
	const q = `
		DELETE FROM memories
		WHERE scope = ? AND project IS ? AND area IS ? AND key = ?`
	res, err := s.db.ExecContext(ctx, q, string(scope), project, area, key)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete memory rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Env -------------------------------------------------------------------

func (s *SQLiteStore) SetEnv(ctx context.Context, e *models.EnvEntry) error {
	const q = `
		INSERT INTO env_entries (id, project, key, value, description, updated_at)
		VALUES (?,?,?,?,?, ` + nowExpr + `)
		ON CONFLICT (project, key)
		DO UPDATE SET value = excluded.value, description = excluded.description, updated_at = ` + nowExpr + `
		RETURNING id, updated_at`
	if err := s.db.QueryRowContext(ctx, q,
		newUUID(), e.Project, e.Key, e.Value, e.Description,
	).Scan(&e.ID, &e.UpdatedAt); err != nil {
		if isSQLiteFKViolation(err) {
			return ErrInvalidProject
		}
		return fmt.Errorf("set env: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListEnv(ctx context.Context, project string) ([]*models.EnvEntry, error) {
	const q = `SELECT id, project, key, value, description, updated_at
		FROM env_entries WHERE project = ? ORDER BY key`
	rows, err := s.db.QueryContext(ctx, q, project)
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

func (s *SQLiteStore) DeleteEnv(ctx context.Context, project, key string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM env_entries WHERE project = ? AND key = ?`, project, key)
	if err != nil {
		return fmt.Errorf("delete env: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete env rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
