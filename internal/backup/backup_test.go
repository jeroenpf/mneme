package backup_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/backup"
	"github.com/jeroenpfeil/mneme/internal/migrations"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

func freshStore(t *testing.T) store.Store {
	t.Helper()
	dsn := "sqlite:" + filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

func strptr(s string) *string { return &s }

func seed(t *testing.T, st store.Store) {
	t.Helper()
	ctx := context.Background()
	if err := st.CreateProject(ctx, &models.Project{Name: "Apollo", Slug: "apollo"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDocument(ctx, &models.Document{
		ID: "plan-1", Title: "The Plan", Project: strptr("apollo"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		Body: map[string]any{"sections": []any{map[string]any{"type": "section", "id": "s1", "title": "Intro"}}},
		Meta: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDecision(ctx, &models.Decision{Title: "Use pgx", Project: strptr("apollo"), Decision: "pgx/v5", Status: models.DecisionAccepted}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSnippet(ctx, &models.Snippet{Title: "pool", Project: strptr("apollo"), Language: "go", Content: "pgxpool.New(ctx, dsn)"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateJournalEntry(ctx, &models.JournalEntry{Summary: "kickoff", Project: strptr("apollo"), SessionRef: "sp-1"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSolution(ctx, &models.Solution{ErrorDescription: "TLS handshake", Solution: "trust the mkcert CA"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetMemory(ctx, &models.Memory{Scope: models.ScopeProject, Project: strptr("apollo"), Key: "db", Value: "postgres"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetEnv(ctx, &models.EnvEntry{Project: "apollo", Key: "API_PORT", Value: "8443"}); err != nil {
		t.Fatal(err)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	ctx := context.Background()
	src := freshStore(t)
	seed(t, src)

	// Export the source, restore into a fresh store, re-export, and verify the
	// two archives hold the same knowledge by content.
	arch1, err := backup.Export(ctx, src)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(arch1.Documents) != 1 || len(arch1.Projects) != 1 || len(arch1.Env) != 1 {
		t.Fatalf("export incomplete: %+v", arch1)
	}

	dst := freshStore(t)
	res, err := backup.Import(ctx, dst, arch1)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.Created["documents"] != 1 || res.Created["decisions"] != 1 || res.Created["memory"] != 1 {
		t.Errorf("import counts: %+v", res.Created)
	}

	arch2, err := backup.Export(ctx, dst)
	if err != nil {
		t.Fatalf("re-export: %v", err)
	}
	if err := backup.Verify(arch1, arch2); err != nil {
		t.Errorf("round-trip verify failed: %v", err)
	}
}

func TestArchiveJSONRoundTrip(t *testing.T) {
	ctx := context.Background()
	src := freshStore(t)
	seed(t, src)

	arch, err := backup.Export(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := arch.Write(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err := backup.Read(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := backup.Verify(arch, back); err != nil {
		t.Errorf("json round-trip changed content: %v", err)
	}
}

func TestImportSkipsDuplicates(t *testing.T) {
	ctx := context.Background()
	src := freshStore(t)
	seed(t, src)
	arch, _ := backup.Export(ctx, src)

	// Importing back into the source: the project and document already exist and
	// are skipped rather than erroring.
	res, err := backup.Import(ctx, src, arch)
	if err != nil {
		t.Fatalf("re-import into populated store: %v", err)
	}
	if res.Skipped["projects"] != 1 || res.Skipped["documents"] != 1 {
		t.Errorf("expected project+document skipped, got %+v", res.Skipped)
	}
}

func TestReadRejectsUnknownVersion(t *testing.T) {
	if _, err := backup.Read(bytes.NewReader([]byte(`{"version":999}`))); err == nil {
		t.Error("expected an unsupported-version error")
	}
}
