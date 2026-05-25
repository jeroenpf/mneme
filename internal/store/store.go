package store

import (
	"context"
	"errors"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// ErrNotFound is returned when a document lookup misses.
var ErrNotFound = errors.New("document not found")

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

	// Ping verifies the underlying connection is alive — used by the
	// /health endpoint.
	Ping(ctx context.Context) error

	// Close releases pooled resources.
	Close()
}
