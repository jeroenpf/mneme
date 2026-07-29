package relations

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// Backfill scans every stored document once when the relations table is
// empty — the upgrade path for datasets written before relations existed —
// and is a no-op on any later start.
func TestBackfill(t *testing.T) {
	ctx := context.Background()
	dsn := "sqlite:" + filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	a := &models.Document{ID: "plan-a", Title: "A", Type: "plan", Status: "todo",
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": "see [[plan-b]]"},
		}}}
	if err := st.CreateDocument(ctx, a); err != nil {
		t.Fatalf("create a: %v", err)
	}
	b := &models.Document{ID: "plan-b", Title: "B", Type: "plan", Status: "todo",
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "p", "content": "back at " + a.PublicID},
		}}}
	if err := st.CreateDocument(ctx, b); err != nil {
		t.Fatalf("create b: %v", err)
	}

	scanned, err := Backfill(ctx, st)
	if err != nil {
		t.Fatalf("Backfill: %v", err)
	}
	if scanned != 2 {
		t.Fatalf("scanned = %d, want 2", scanned)
	}
	n, err := st.CountRelations(ctx)
	if err != nil {
		t.Fatalf("CountRelations: %v", err)
	}
	if n != 2 {
		t.Fatalf("relations after backfill = %d, want 2", n)
	}

	// Non-empty table → no-op.
	scanned, err = Backfill(ctx, st)
	if err != nil {
		t.Fatalf("Backfill(second): %v", err)
	}
	if scanned != 0 {
		t.Fatalf("second Backfill scanned = %d, want 0", scanned)
	}
}
