package mcp_test

import (
	"context"
	"testing"

	"github.com/jeroenpf/mneme/internal/ids"
	mcpsrv "github.com/jeroenpf/mneme/internal/mcp"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

func TestMigrateDocIDsBackfillsMissing(t *testing.T) {
	resetDB(t)
	st := store.NewWithPool(testPool)
	ctx := context.Background()

	// A document inserted straight through the store (bypassing push's id
	// normalization) with one section lacking an id and one keeping a semantic id.
	doc := &models.Document{
		ID: "legacy", Title: "Legacy", Type: "spec", Status: models.StatusTodo,
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "title": "No id"},
			map[string]any{"type": "section", "id": "kept", "title": "Kept"},
		}},
	}
	if err := st.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}

	// Dry run reports the missing id but persists nothing.
	sum, err := mcpsrv.MigrateDocIDs(ctx, st, false)
	if err != nil {
		t.Fatalf("MigrateDocIDs(dry): %v", err)
	}
	if sum.Backfilled["legacy"] != 1 {
		t.Errorf("dry run backfilled = %v, want legacy:1", sum.Backfilled)
	}
	reload, _ := st.GetDocument(ctx, "legacy")
	if _, ok := reload.Body["sections"].([]any)[0].(map[string]any)["id"]; ok {
		t.Error("dry run must not persist ids")
	}

	// Apply mints and persists, preserving the existing semantic id.
	if _, err := mcpsrv.MigrateDocIDs(ctx, st, true); err != nil {
		t.Fatalf("MigrateDocIDs(apply): %v", err)
	}
	reload, _ = st.GetDocument(ctx, "legacy")
	first := reload.Body["sections"].([]any)[0].(map[string]any)
	if id, _ := first["id"].(string); !ids.ValidFor(ids.KindBlock, id) {
		t.Errorf("missing id not backfilled: %q", id)
	}
	if kept := reload.Body["sections"].([]any)[1].(map[string]any); kept["id"] != "kept" {
		t.Errorf("existing semantic id should be preserved: %v", kept["id"])
	}
}

func TestMigrateDocIDsReportsDuplicates(t *testing.T) {
	resetDB(t)
	st := store.NewWithPool(testPool)
	ctx := context.Background()

	doc := &models.Document{
		ID: "dups", Title: "Dups", Type: "spec", Status: models.StatusTodo,
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "text", "id": "same", "content": "a"},
			map[string]any{"type": "text", "id": "same", "content": "b"},
		}},
	}
	if err := st.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}

	sum, err := mcpsrv.MigrateDocIDs(ctx, st, true)
	if err != nil {
		t.Fatalf("MigrateDocIDs: %v", err)
	}
	if sum.Problems["dups"] == "" {
		t.Errorf("duplicate ids should be reported as a problem: %+v", sum)
	}
	if _, backfilled := sum.Backfilled["dups"]; backfilled {
		t.Errorf("a document with duplicate ids should not be backfilled")
	}
}
