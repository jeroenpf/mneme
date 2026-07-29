package relations

import (
	"context"
	"fmt"

	"github.com/jeroenpf/mneme/internal/store"
)

// Backfill populates the relations table from every stored document when the
// table is empty — the upgrade path for datasets written before relations
// existed. Any later start is a no-op: the write-path sync owns the table
// from then on. Returns the number of documents scanned.
func Backfill(ctx context.Context, st store.Store) (int, error) {
	n, err := st.CountRelations(ctx)
	if err != nil {
		return 0, fmt.Errorf("backfill relations: %w", err)
	}
	if n > 0 {
		return 0, nil
	}
	docs, err := st.ListDocuments(ctx, store.Filter{})
	if err != nil {
		return 0, fmt.Errorf("backfill relations: list documents: %w", err)
	}
	for _, d := range docs {
		if err := st.ReplaceAutoMentions(ctx, d.PublicID, MentionRows(d)); err != nil {
			return 0, fmt.Errorf("backfill relations: %s: %w", d.ID, err)
		}
	}
	return len(docs), nil
}
