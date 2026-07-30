package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

func TestLintCommandReportsBadDocument(t *testing.T) {
	dsn := "sqlite:" + filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	ctx := context.Background()
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	// Seeded through the store, so write-path validation never ran: a
	// markdown list inside section prose plus an unknown field.
	doc := &models.Document{
		ID: "legacy-bad", Title: "Legacy Bad", Type: "spec", Status: models.StatusTodo,
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "s1", "title": "S", "content": "items:\n- one\n- two"},
			map[string]any{"type": "code", "id": "c1", "language": "go", "content": "x"},
		}},
	}
	if err := st.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("create doc: %v", err)
	}
	st.Close()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"lint", "--dsn", dsn})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute lint: %v", err)
	}
	got := out.String()
	for _, want := range []string{"legacy-bad", "a list", "unknown field"} {
		if !strings.Contains(got, want) {
			t.Errorf("lint output missing %q:\n%s", want, got)
		}
	}
}
