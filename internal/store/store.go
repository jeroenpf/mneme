package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jeroenpf/mneme/internal/dsn"
	"github.com/jeroenpf/mneme/internal/models"
)

// New builds a Store from a DSN, dispatching on the scheme: sqlite: / file: /
// a *.db path selects the pure-Go SQLiteStore (the self-contained binary);
// anything else defaults to the PostgresStore. Everything above this seam — the
// API, MCP tools, bundle, and live hub — is backend-agnostic.
func New(ctx context.Context, connDSN string) (Store, error) {
	if dsn.IsSQLite(connDSN) {
		return NewSQLiteStore(ctx, connDSN)
	}
	return NewPostgresStore(ctx, connDSN)
}

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

// ErrRevisionConflict is the sentinel a RevisionConflictError matches under
// errors.Is — an optimistic-concurrency check failed because the document was
// modified since the caller's expected revision. Use errors.As to recover the
// current revision for a useful conflict response.
var ErrRevisionConflict = errors.New("revision conflict")

// RevisionConflictError is returned by UpdateDocument when the caller supplied
// an expected revision that no longer matches the stored one. It carries the
// document id, the current revision, and the ids the intervening write(s)
// changed (from the audit log) so callers (REST 412 / MCP) can tell the user
// exactly what to re-read.
type RevisionConflictError struct {
	DocumentID string
	Current    int
	ChangedIDs []string
}

func (e *RevisionConflictError) Error() string {
	if len(e.ChangedIDs) > 0 {
		return fmt.Sprintf("revision conflict on %s: current revision is %d; changed since yours: %v",
			e.DocumentID, e.Current, e.ChangedIDs)
	}
	return fmt.Sprintf("revision conflict on %s: current revision is %d", e.DocumentID, e.Current)
}

// Is lets errors.Is(err, ErrRevisionConflict) match a *RevisionConflictError.
func (e *RevisionConflictError) Is(target error) bool { return target == ErrRevisionConflict }

// changedTargetsSince collects the distinct, sorted target ids recorded for
// revisions after sinceRevision — what a writer basing an edit on sinceRevision
// missed. Best-effort: a history read error yields no ids rather than masking
// the conflict itself. Backend-agnostic (reads through the Store interface).
func changedTargetsSince(ctx context.Context, s Store, documentID string, sinceRevision int) []string {
	revs, err := s.ListDocumentRevisions(ctx, documentID, 0)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range revs {
		if r.Revision <= sinceRevision {
			continue
		}
		for _, id := range r.TargetIDs {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	sort.Strings(out)
	return out
}

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

	// GetDocumentByPublicID returns the document whose doc_… public id
	// matches, or ErrNotFound. resolve_reference uses it to turn a pasted
	// reference into a document (and to reach its blocks and tasks).
	GetDocumentByPublicID(ctx context.Context, publicID string) (*models.Document, error)

	// UpdateDocument writes all mutable columns from doc (everything
	// except id, created_at, search_vector), atomically bumping the
	// document's revision and scanning the new value back into doc.Revision.
	// updated_at is managed by the trigger added in migration 004. When
	// expected is non-nil, the write applies only if the stored revision
	// equals *expected, otherwise it returns *RevisionConflictError
	// (errors.Is ErrRevisionConflict) without modifying the row. Returns
	// ErrNotFound when id has no row.
	UpdateDocument(ctx context.Context, doc *models.Document, expected *int) error

	// ArchiveDocument sets status='archived'. Returns ErrNotFound when
	// id has no row.
	ArchiveDocument(ctx context.Context, id string) error

	// AppendDocumentRevision records an immutable snapshot of a document write
	// — the audit trail and history source (roadmap P6). Fills rev.ID and
	// rev.CreatedAt from the DB. A duplicate (document_id, revision) is rejected.
	AppendDocumentRevision(ctx context.Context, rev *models.DocumentRevision) error

	// ListDocumentRevisions returns a document's revision snapshots newest-first,
	// capped by limit (0 = no limit). Empty when the document has no history.
	ListDocumentRevisions(ctx context.Context, documentID string, limit int) ([]*models.DocumentRevision, error)

	// GetDocumentRevision returns one snapshot by (documentID, revision), or
	// ErrNotFound when that revision was never recorded.
	GetDocumentRevision(ctx context.Context, documentID string, revision int) (*models.DocumentRevision, error)

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

	// GetProjectByPublicID returns the project whose prj_… public id matches,
	// or ErrNotFound.
	GetProjectByPublicID(ctx context.Context, publicID string) (*models.Project, error)

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

	// GetDecisionByPublicID returns the decision whose dec_… public id
	// matches, or ErrNotFound.
	GetDecisionByPublicID(ctx context.Context, publicID string) (*models.Decision, error)

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

	// GetSnippetByPublicID returns the snippet whose snip_… public id
	// matches, or ErrNotFound.
	GetSnippetByPublicID(ctx context.Context, publicID string) (*models.Snippet, error)

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

	// GetJournalEntryByPublicID returns the entry whose jrnl_… public id
	// matches, or ErrNotFound.
	GetJournalEntryByPublicID(ctx context.Context, publicID string) (*models.JournalEntry, error)

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

	// GetSolutionByPublicID returns the solution whose sol_… public id
	// matches, or ErrNotFound.
	GetSolutionByPublicID(ctx context.Context, publicID string) (*models.Solution, error)

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
	// DeleteOrphanEmbeddings sweeps vectors whose source_id no longer
	// resolves to a live source row, returning the count removed.
	DeleteOrphanEmbeddings(ctx context.Context) (int64, error)
	// HasStaleModelEmbeddings reports whether a source has any vector on a
	// model other than the given one (i.e. a model switch needs a re-embed).
	HasStaleModelEmbeddings(ctx context.Context, sourceType, sourceID, model string) (bool, error)
	// EmbeddingsFor returns chunk_id→chunk_text for a source.
	EmbeddingsFor(ctx context.Context, sourceType, sourceID string) (map[string]string, error)
	// SourceRefs enumerates every embeddable source.
	SourceRefs(ctx context.Context) ([]SourceRef, error)
	// EmbeddingStatus reports per-type reconciliation buckets (reconciled,
	// missing, stale, orphaned, failed) against the current embedding model.
	EmbeddingStatus(ctx context.Context, model string) ([]TypeStatus, error)

	// RecordEmbedFailure upserts a terminal embed failure for a source
	// (latest error, incremented attempt count).
	RecordEmbedFailure(ctx context.Context, sourceType, sourceID, errMsg string) error
	// ClearEmbedFailure removes a source's recorded failure (no-op if absent).
	ClearEmbedFailure(ctx context.Context, sourceType, sourceID string) error
	// FailedSourceRefs lists sources with a recorded failure, for manual retry.
	FailedSourceRefs(ctx context.Context) ([]SourceRef, error)

	// SetSearchMaxDist sets the hybrid-search relevance floor (cosine
	// distance): vector candidates beyond it are dropped so a vague query
	// returns nothing rather than the whole corpus. <= 0 disables the floor.
	// Called once at startup from config, before the store is shared.
	SetSearchMaxDist(d float64)

	// Ping verifies the underlying connection is alive — used by the
	// /health endpoint.
	Ping(ctx context.Context) error

	// Close releases pooled resources.
	Close()
}
