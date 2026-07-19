package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/migrations"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

func TestMigrateIDsCommandReportsThenApplies(t *testing.T) {
	dsn := "sqlite:" + filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	ctx := context.Background()
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	doc := &models.Document{
		ID: "legacy", Title: "Legacy", Type: "spec", Status: models.StatusTodo,
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "title": "No id"},
		}},
	}
	if err := st.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("create doc: %v", err)
	}
	st.Close()

	run := func(args ...string) string {
		t.Helper()
		root := newRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("execute %v: %v", args, err)
		}
		return out.String()
	}

	// Report-only (default): names the document, does not persist.
	out := run("migrate", "ids", "--dsn", dsn)
	if !strings.Contains(out, "legacy") || !strings.Contains(out, "would mint") {
		t.Errorf("report output = %q", out)
	}
	if !strings.Contains(out, "--apply") {
		t.Errorf("report should suggest --apply: %q", out)
	}

	// Apply persists the backfill.
	out = run("migrate", "ids", "--dsn", dsn, "--apply")
	if !strings.Contains(out, "minted") {
		t.Errorf("apply output = %q", out)
	}

	st2, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st2.Close()
	reload, err := st2.GetDocument(ctx, "legacy")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	sec := reload.Body["sections"].([]any)[0].(map[string]any)
	if id, _ := sec["id"].(string); id == "" {
		t.Errorf("id not persisted: %+v", sec)
	}
}
