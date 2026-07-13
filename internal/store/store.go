package store

import (
	"context"
	"errors"
	"time"

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

// SnippetFilter narrows ListSnippets / SearchSnippets. A nil/empty field
// is "no constraint". Project nil does NOT mean "global only" — it means
// "any project". Tags matches snippets containing ALL listed tags
// (tags @> filter).
type SnippetFilter struct {
	Project  *string
	Language *string
	Tags     []string
	Limit    int // 0 → no limit
}

// JournalFilter narrows ListJournalEntries. A nil/zero field is "no
// constraint". Project nil means "any project" (not "global only").
// Since bounds created_at from below (created_at >= Since).
type JournalFilter struct {
	Project *string
	Since   *time.Time
	Limit   int // 0 → no limit
}

// SolutionFilter narrows ListSolutions / SearchSolutions. A nil/empty
// field is "no constraint". Project nil means "any project" (not "global
// only"). Tags matches solutions containing ALL listed tags (tags @> filter).
type SolutionFilter struct {
	Project *string
	Tags    []string
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

	// GetProject returns the project with the given slug, or ErrNotFound.
	GetProject(ctx context.Context, slug string) (*models.Project, error)

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

	// SetEnv upserts an env entry by (project, key), filling e.ID and
	// e.UpdatedAt from the DB. Returns ErrInvalidProject when project is
	// an unknown slug. The upsert replaces value AND description.
	SetEnv(ctx context.Context, e *models.EnvEntry) error

	// ListEnv returns a project's env entries ordered by key.
	ListEnv(ctx context.Context, project string) ([]*models.EnvEntry, error)

	// DeleteEnv removes one entry by (project, key). Returns ErrNotFound
	// when no row matched.
	DeleteEnv(ctx context.Context, project, key string) error

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

	// CreateSnippet inserts a snippet and fills sn.ID/sn.CreatedAt/
	// sn.UpdatedAt from the DB. Returns ErrInvalidProject when a
	// project-scoped snippet references an unknown slug.
	CreateSnippet(ctx context.Context, sn *models.Snippet) error

	// GetSnippet returns ErrNotFound when id has no row.
	GetSnippet(ctx context.Context, id string) (*models.Snippet, error)

	// UpdateSnippet writes all mutable columns by id (updated_at is
	// trigger-managed). Returns ErrNotFound when id has no row and
	// ErrInvalidProject on an unknown project slug.
	UpdateSnippet(ctx context.Context, sn *models.Snippet) error

	// ListSnippets returns snippets newest-first, filtered by the
	// non-nil SnippetFilter fields.
	ListSnippets(ctx context.Context, f SnippetFilter) ([]*models.Snippet, error)

	// SearchSnippets runs websearch_to_tsquery('english', q) against
	// snippets.search_vector, ordered by ts_rank desc then newest-first.
	SearchSnippets(ctx context.Context, q string, f SnippetFilter) ([]*models.Snippet, error)

	// CreateJournalEntry inserts an entry and fills e.ID/e.CreatedAt/
	// e.UpdatedAt from the DB. Returns ErrInvalidProject when a
	// project-scoped entry references an unknown slug.
	CreateJournalEntry(ctx context.Context, e *models.JournalEntry) error

	// GetJournalEntry returns ErrNotFound when id has no row.
	GetJournalEntry(ctx context.Context, id string) (*models.JournalEntry, error)

	// UpdateJournalEntry writes all mutable columns by id (updated_at is
	// trigger-managed). Returns ErrNotFound when id has no row and
	// ErrInvalidProject on an unknown project slug.
	UpdateJournalEntry(ctx context.Context, e *models.JournalEntry) error

	// ListJournalEntries returns entries newest-first, filtered by the
	// non-nil JournalFilter fields.
	ListJournalEntries(ctx context.Context, f JournalFilter) ([]*models.JournalEntry, error)

	// CreateSolution inserts a solution and fills sol.ID/sol.CreatedAt/
	// sol.UpdatedAt from the DB. Returns ErrInvalidProject when a
	// project-scoped solution references an unknown slug.
	CreateSolution(ctx context.Context, sol *models.Solution) error

	// GetSolution returns ErrNotFound when id has no row.
	GetSolution(ctx context.Context, id string) (*models.Solution, error)

	// UpdateSolution writes all mutable columns by id (updated_at is
	// trigger-managed). Returns ErrNotFound when id has no row and
	// ErrInvalidProject on an unknown project slug.
	UpdateSolution(ctx context.Context, sol *models.Solution) error

	// ListSolutions returns solutions newest-first, filtered by the
	// non-nil SolutionFilter fields.
	ListSolutions(ctx context.Context, f SolutionFilter) ([]*models.Solution, error)

	// SearchSolutions runs websearch_to_tsquery('english', q) against
	// solutions.search_vector, ordered by ts_rank desc then newest-first.
	SearchSolutions(ctx context.Context, q string, f SolutionFilter) ([]*models.Solution, error)

	// Search runs a unified FTS query across the requested content types
	// (all of SearchTypes when Types is empty), ranked cross-type by
	// reciprocal rank. Returns ErrInvalidSearchType for an unknown type.
	Search(ctx context.Context, q string, f SearchFilter) ([]*models.SearchHit, error)

	// UpsertEmbeddings inserts/updates embedding rows keyed by
	// (source_type, source_id, chunk_id).
	UpsertEmbeddings(ctx context.Context, rows []models.Embedding) error
	// DeleteEmbeddingsExcept prunes a source's chunks not in keep.
	DeleteEmbeddingsExcept(ctx context.Context, sourceType, sourceID string, keep []string) error
	// EmbeddingsFor returns chunk_id→chunk_text for a source.
	EmbeddingsFor(ctx context.Context, sourceType, sourceID string) (map[string]string, error)
	// SourceRefs enumerates every embeddable source.
	SourceRefs(ctx context.Context) ([]SourceRef, error)
	// EmbeddingCoverage reports embedded/total sources per type.
	EmbeddingCoverage(ctx context.Context) ([]TypeCoverage, error)

	// Ping verifies the underlying connection is alive — used by the
	// /health endpoint.
	Ping(ctx context.Context) error

	// Close releases pooled resources.
	Close()
}
