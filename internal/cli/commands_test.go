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

// The command surface is locked early: init, doctor, export, and import all
// exist as subcommands with a runnable body, even where that body is still a
// stub. This pins the surface so later phases fill in behaviour without moving
// commands around.
func TestCommandSurfaceRegistered(t *testing.T) {
	for _, name := range []string{"server", "init", "doctor", "export", "import"} {
		root := newRootCmd()
		sub, _, err := root.Find([]string{name})
		if err != nil {
			t.Errorf("find %q subcommand: %v", name, err)
			continue
		}
		if sub.Name() != name {
			t.Errorf("expected %q subcommand, got %q", name, sub.Name())
		}
		if sub.RunE == nil {
			t.Errorf("%q subcommand has no RunE", name)
		}
	}
}

// export and import round-trip all local knowledge through a portable JSON
// backup: exporting from one store and importing into a fresh one recreates the
// content (roadmap P6-t7).
func TestExportImportCLIRoundTrip(t *testing.T) {
	ctx := context.Background()
	srcDSN := "sqlite:" + filepath.Join(t.TempDir(), "src.db")
	dstDSN := "sqlite:" + filepath.Join(t.TempDir(), "dst.db")
	for _, d := range []string{srcDSN, dstDSN} {
		if err := migrations.Up(d); err != nil {
			t.Fatalf("migrate %s: %v", d, err)
		}
	}

	src, err := store.New(ctx, srcDSN)
	if err != nil {
		t.Fatal(err)
	}
	if err := src.CreateProject(ctx, &models.Project{Name: "Apollo", Slug: "apollo"}); err != nil {
		t.Fatal(err)
	}
	if err := src.CreateDocument(ctx, &models.Document{
		ID: "plan-1", Title: "The Plan", Project: ptrStr("apollo"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		Body: map[string]any{"sections": []any{}}, Meta: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	src.Close()

	run := func(args ...string) string {
		t.Helper()
		root := newRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		if err := root.ExecuteContext(ctx); err != nil {
			t.Fatalf("execute %v: %v\n%s", args, err, out.String())
		}
		return out.String()
	}

	file := filepath.Join(t.TempDir(), "backup.json")
	if out := run("export", file, "--dsn", srcDSN); !strings.Contains(out, "verified") {
		t.Errorf("export should self-verify: %q", out)
	}
	if out := run("import", file, "--dsn", dstDSN); !strings.Contains(out, "import complete") {
		t.Errorf("import output: %q", out)
	}

	// The restored store has the document.
	dst, err := store.New(ctx, dstDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	got, err := dst.GetDocument(ctx, "plan-1")
	if err != nil {
		t.Fatalf("restored document missing: %v", err)
	}
	if got.Title != "The Plan" {
		t.Errorf("restored title = %q", got.Title)
	}
}

func ptrStr(s string) *string { return &s }
