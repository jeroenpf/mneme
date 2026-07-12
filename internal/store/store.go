package store

import (
	"context"
	"errors"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// ErrNotFound is returned when a document lookup misses.
var ErrNotFound = errors.New("document not found")

// ErrInvalidProject is returned when a write references a project slug
// that does not exist in the projects table (FK violation on
// documents.project).
var ErrInvalidProject = errors.New("invalid project")

// ErrDuplicateID is returned when CreateDocument hits a PK conflict on
// documents.id. Callers performing slug allocation should retry with a
// new candidate.
var ErrDuplicateID = errors.New("duplicate document id")

// ErrDuplicateProject is returned when CreateProject hits a UNIQUE
// violation on projects.slug — the project already exists.
var ErrDuplicateProject = errors.New("duplicate project")

// MemoryFilter narrows ListMemory. A nil field is "no constraint".
type MemoryFilter struct {
	Scope   *models.MemoryScope
	Project *string
	Area    *string
}

// DecisionFilter narrows ListDecisions / SearchDecisions. A nil field is
// "no constraint". Project nil does NOT mean "global only" — it means
// "any project"; there is no dedicated global-only filter (callers that
// need it can filter the returned slice on Project == nil).
type DecisionFilter struct {
	Project *string
	Status  *models.DecisionStatus
	Limit   int // 0 → no limit
}

// Filter narrows ListDocuments / SearchDocuments. Zero/nil fields are
// treated as "no constraint".
type Filter struct {
	Project *string
	Type    *string
	Status  *string
	Tags    []string // matches documents containing ALL of these tags

	Limit  int // 0 → no limit applied (callers should set a sane default)
	Offset int
}

// Store is the data-layer contract. Implementations must be safe for
// concurrent use.
type Store interface {
	CreateDocument(ctx context.Context, doc *models.Document) error

	// GetDocument returns ErrNotFound when id has no row.
	GetDocument(ctx context.Context, id string) (*models.Document, error)

	// UpdateDocument writes all mutable columns from doc (everything
	// except id, created_at, search_vector). updated_at is managed by
	// the trigger added in migration 004. Returns ErrNotFound when id
	// has no row.
	UpdateDocument(ctx context.Context, doc *models.Document) error

	// ArchiveDocument sets status='archived'. Returns ErrNotFound when
	// id has no row.
	ArchiveDocument(ctx context.Context, id string) error

	ListDocuments(ctx context.Context, f Filter) ([]*models.Document, error)

	// SearchDocuments runs plainto_tsquery('english', q) against
	// documents.search_vector and orders by ts_rank desc. Additional
	// Filter fields are AND-ed onto the query.
	SearchDocuments(ctx context.Context, q string, f Filter) ([]*models.Document, error)

	// ListProjects returns every project with per-status document counts.
	// Used by /api/v1/projects and the registry UI's stats row.
	ListProjects(ctx context.Context) ([]*models.ProjectStats, error)

	// CreateProject inserts a project and fills p.ID and p.CreatedAt from
	// the DB defaults. Returns ErrDuplicateProject when p.Slug already
	// exists. The caller is responsible for normalizing p.Slug.
	CreateProject(ctx context.Context, p *models.Project) error

	// ListMemory returns raw (un-merged) memory entries matching the
	// filter, ordered by (scope, project, area, key). The hierarchy
	// merge lives in the MCP get_memory handler, not here.
	ListMemory(ctx context.Context, f MemoryFilter) ([]*models.Memory, error)

	// SetMemory upserts a memory entry by (scope, project, area, key),
	// filling m.ID and m.UpdatedAt from the DB. Returns ErrInvalidProject
	// when a project/area-scoped entry references an unknown project slug.
	SetMemory(ctx context.Context, m *models.Memory) error

	// DeleteMemory removes the entry with the given exact identity.
	// project/area are nil for scopes that don't use them. Returns
	// ErrNotFound when no row matched.
	DeleteMemory(ctx context.Context, scope models.MemoryScope, project, area *string, key string) error

	// CreateDecision inserts a decision and fills d.ID/d.CreatedAt/
	// d.UpdatedAt from the DB. Returns ErrInvalidProject when a
	// project-scoped decision references an unknown slug.
	CreateDecision(ctx context.Context, d *models.Decision) error

	// GetDecision returns ErrNotFound when id has no row.
	GetDecision(ctx context.Context, id string) (*models.Decision, error)

	// UpdateDecision writes all mutable columns by id (updated_at is
	// trigger-managed). Returns ErrNotFound when id has no row and
	// ErrInvalidProject on an unknown project slug.
	UpdateDecision(ctx context.Context, d *models.Decision) error

	// ListDecisions returns decisions newest-first, filtered by the
	// non-nil DecisionFilter fields.
	ListDecisions(ctx context.Context, f DecisionFilter) ([]*models.Decision, error)

	// SearchDecisions runs websearch_to_tsquery('english', q) against
	// decisions.search_vector, ordered by ts_rank desc then newest-first.
	SearchDecisions(ctx context.Context, q string, f DecisionFilter) ([]*models.Decision, error)

	// Ping verifies the underlying connection is alive — used by the
	// /health endpoint.
	Ping(ctx context.Context) error

	// Close releases pooled resources.
	Close()
}
