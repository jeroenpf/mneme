package store

import (
	"context"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// This file holds the SQLiteStore Store methods not yet implemented. CRUD +
// embeddings storage land in P3; the search methods below land in P5 (FTS5 +
// Go vectors). When the port is complete this file is empty. The compile-time
// assertion that *SQLiteStore satisfies Store keeps the interface honest.

var _ Store = (*SQLiteStore)(nil)

// --- Search (P5: FTS5 + Go vectors) ----------------------------------------

func (s *SQLiteStore) SearchDocuments(ctx context.Context, q string, f Filter) ([]*models.Document, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SearchDecisions(ctx context.Context, q string, f DecisionFilter) ([]*models.Decision, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SearchSnippets(ctx context.Context, q string, f SnippetFilter) ([]*models.Snippet, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) SearchSolutions(ctx context.Context, q string, f SolutionFilter) ([]*models.Solution, error) {
	return nil, errNotImplemented
}

func (s *SQLiteStore) Search(ctx context.Context, q string, f SearchFilter) ([]*models.SearchHit, error) {
	return nil, errNotImplemented
}
