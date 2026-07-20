package mcp

import (
	"context"
	"fmt"

	"github.com/jeroenpf/mneme/internal/store"
)

// MigrateSummary reports an id audit/backfill over every stored document.
// Backfilled maps a document id to how many block/task ids were minted for
// nodes that lacked one; Problems maps a document id to why it could not be
// normalized (e.g. a duplicate id, which cannot be auto-resolved and needs a
// human). Scanned is the total number of documents examined.
type MigrateSummary struct {
	Scanned    int               `json:"scanned"`
	Backfilled map[string]int    `json:"backfilled"`
	Problems   map[string]string `json:"problems"`
}

// Changed reports how many documents would be (or were) modified.
func (s *MigrateSummary) Changed() int { return len(s.Backfilled) }

// MigrateDocIDs walks every stored document tree and normalizes its block/task
// ids: nodes missing an id have one minted, and duplicate ids are reported (not
// silently changed). With apply=false it only reports what would change — data
// is untouched — so an operator can inspect before committing. With apply=true
// the backfilled documents are persisted. Duplicates are always reported, never
// auto-resolved, since picking which node keeps the id is a human decision.
func MigrateDocIDs(ctx context.Context, st store.Store, apply bool) (*MigrateSummary, error) {
	docs, err := st.ListDocuments(ctx, store.Filter{})
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	sum := &MigrateSummary{Backfilled: map[string]int{}, Problems: map[string]string{}}
	for _, doc := range docs {
		sum.Scanned++
		created, err := normalizeBodyIDs(doc.Body)
		if err != nil {
			sum.Problems[doc.ID] = err.Error()
			continue
		}
		if len(created) == 0 {
			continue
		}
		sum.Backfilled[doc.ID] = len(created)
		if apply {
			if err := st.UpdateDocument(ctx, doc, nil); err != nil {
				return nil, fmt.Errorf("persist %s: %w", doc.ID, err)
			}
		}
	}
	return sum, nil
}
