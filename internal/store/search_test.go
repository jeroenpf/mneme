package store_test

import (
	"context"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
)

// TestJournalFTSColumn proves migration 012 added a working search_vector to
// journal_entries: an entry is findable by a word from its summary.
func TestJournalFTSColumn(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.CreateJournalEntry(ctx, &models.JournalEntry{
		Project: ptr("apollo"),
		Summary: "shipped the zigbee coordinator swap",
	}); err != nil {
		t.Fatal(err)
	}

	var n int
	err := s.Pool().QueryRow(ctx,
		`SELECT count(*) FROM journal_entries
		 WHERE search_vector @@ websearch_to_tsquery('english', $1)`,
		"zigbee").Scan(&n)
	if err != nil {
		t.Fatalf("query journal search_vector: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 journal FTS match, got %d", n)
	}
}
