package store

import (
	"context"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// This file holds the SQLiteStore Store methods not yet implemented. Each phase
// of plan-sqlite-backend replaces a group of these stubs with real code (P3:
// CRUD + embeddings storage; P5: search). When the port is complete this file
// is empty. A compile-time assertion that *SQLiteStore satisfies Store lives
// beside the constructor once every method is real; until then the stubs keep
// the interface satisfied.

var _ Store = (*SQLiteStore)(nil)

// --- Documents -------------------------------------------------------------

func (s *SQLiteStore) CreateDocument(ctx context.Context, doc *models.Document) error {
	return errNotImplemented
}

func (s *SQLiteStore) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) UpdateDocument(ctx context.Context, doc *models.Document) error {
	return errNotImplemented
}

func (s *SQLiteStore) ArchiveDocument(ctx context.Context, id string) error {
	return errNotImplemented
}

func (s *SQLiteStore) ListDocuments(ctx context.Context, f Filter) ([]*models.Document, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SearchDocuments(ctx context.Context, q string, f Filter) ([]*models.Document, error) {
	return nil, errNotImplemented
}

// --- Projects --------------------------------------------------------------

func (s *SQLiteStore) ListProjects(ctx context.Context) ([]*models.ProjectStats, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) CreateProject(ctx context.Context, p *models.Project) error {
	return errNotImplemented
}

func (s *SQLiteStore) GetProject(ctx context.Context, slug string) (*models.Project, error) {
	return nil, errNotImplemented
}

// --- Memory ----------------------------------------------------------------

func (s *SQLiteStore) ListMemory(ctx context.Context, f MemoryFilter) ([]*models.Memory, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SetMemory(ctx context.Context, m *models.Memory) error {
	return errNotImplemented
}

func (s *SQLiteStore) DeleteMemory(ctx context.Context, scope models.MemoryScope, project, area *string, key string) error {
	return errNotImplemented
}

// --- Env -------------------------------------------------------------------

func (s *SQLiteStore) SetEnv(ctx context.Context, e *models.EnvEntry) error {
	return errNotImplemented
}

func (s *SQLiteStore) ListEnv(ctx context.Context, project string) ([]*models.EnvEntry, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) DeleteEnv(ctx context.Context, project, key string) error {
	return errNotImplemented
}

// --- Decisions -------------------------------------------------------------

func (s *SQLiteStore) CreateDecision(ctx context.Context, d *models.Decision) error {
	return errNotImplemented
}

func (s *SQLiteStore) GetDecision(ctx context.Context, id string) (*models.Decision, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) UpdateDecision(ctx context.Context, d *models.Decision) error {
	return errNotImplemented
}

func (s *SQLiteStore) ListDecisions(ctx context.Context, f DecisionFilter) ([]*models.Decision, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SearchDecisions(ctx context.Context, q string, f DecisionFilter) ([]*models.Decision, error) {
	return nil, errNotImplemented
}

// --- Snippets --------------------------------------------------------------

func (s *SQLiteStore) CreateSnippet(ctx context.Context, sn *models.Snippet) error {
	return errNotImplemented
}

func (s *SQLiteStore) GetSnippet(ctx context.Context, id string) (*models.Snippet, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) UpdateSnippet(ctx context.Context, sn *models.Snippet) error {
	return errNotImplemented
}

func (s *SQLiteStore) ListSnippets(ctx context.Context, f SnippetFilter) ([]*models.Snippet, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SearchSnippets(ctx context.Context, q string, f SnippetFilter) ([]*models.Snippet, error) {
	return nil, errNotImplemented
}

// --- Journal ---------------------------------------------------------------

func (s *SQLiteStore) CreateJournalEntry(ctx context.Context, e *models.JournalEntry) error {
	return errNotImplemented
}

func (s *SQLiteStore) GetJournalEntry(ctx context.Context, id string) (*models.JournalEntry, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) UpdateJournalEntry(ctx context.Context, e *models.JournalEntry) error {
	return errNotImplemented
}

func (s *SQLiteStore) ListJournalEntries(ctx context.Context, f JournalFilter) ([]*models.JournalEntry, error) {
	return nil, errNotImplemented
}

// --- Solutions -------------------------------------------------------------

func (s *SQLiteStore) CreateSolution(ctx context.Context, sol *models.Solution) error {
	return errNotImplemented
}

func (s *SQLiteStore) GetSolution(ctx context.Context, id string) (*models.Solution, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) UpdateSolution(ctx context.Context, sol *models.Solution) error {
	return errNotImplemented
}

func (s *SQLiteStore) ListSolutions(ctx context.Context, f SolutionFilter) ([]*models.Solution, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SearchSolutions(ctx context.Context, q string, f SolutionFilter) ([]*models.Solution, error) {
	return nil, errNotImplemented
}

// --- Unified search --------------------------------------------------------

func (s *SQLiteStore) Search(ctx context.Context, q string, f SearchFilter) ([]*models.SearchHit, error) {
	return nil, errNotImplemented
}

// --- Embeddings ------------------------------------------------------------

func (s *SQLiteStore) UpsertEmbeddings(ctx context.Context, rows []models.Embedding) error {
	return errNotImplemented
}

func (s *SQLiteStore) DeleteEmbeddingsExcept(ctx context.Context, sourceType, sourceID string, keep []string) error {
	return errNotImplemented
}

func (s *SQLiteStore) DeleteOrphanEmbeddings(ctx context.Context) (int64, error) {
	return 0, errNotImplemented
}

func (s *SQLiteStore) HasStaleModelEmbeddings(ctx context.Context, sourceType, sourceID, model string) (bool, error) {
	return false, errNotImplemented
}

func (s *SQLiteStore) EmbeddingsFor(ctx context.Context, sourceType, sourceID string) (map[string]string, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SourceRefs(ctx context.Context) ([]SourceRef, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) EmbeddingStatus(ctx context.Context, model string) ([]TypeStatus, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) RecordEmbedFailure(ctx context.Context, sourceType, sourceID, errMsg string) error {
	return errNotImplemented
}

func (s *SQLiteStore) ClearEmbedFailure(ctx context.Context, sourceType, sourceID string) error {
	return errNotImplemented
}

func (s *SQLiteStore) FailedSourceRefs(ctx context.Context) ([]SourceRef, error) {
	return nil, errNotImplemented
}
