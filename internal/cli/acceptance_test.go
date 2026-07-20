package cli_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/jeroenpf/mneme/internal/backup"
	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

func acceptPtr(s string) *string { return &s }

// seedAcceptance populates a store with one of every knowledge type, so a
// backup exercises the whole surface. The "coordinator" text gives the offline
// lexical-search step a term to match.
func seedAcceptance(t *testing.T, st store.Store) {
	t.Helper()
	ctx := context.Background()
	must := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed %s: %v", what, err)
		}
	}
	must("project", st.CreateProject(ctx, &models.Project{Name: "Apollo", Slug: "apollo"}))
	must("document", st.CreateDocument(ctx, &models.Document{
		ID: "coordinator-plan", Title: "Coordinator Plan", Project: acceptPtr("apollo"),
		Type: models.TypePlan, Status: models.StatusInProgress, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "s1", "title": "Intro", "content": "the coordinator wires everything"},
		}},
	}))
	must("decision", st.CreateDecision(ctx, &models.Decision{
		Title: "Use pgx", Project: acceptPtr("apollo"), Decision: "pgx/v5", Status: models.DecisionAccepted,
	}))
	must("snippet", st.CreateSnippet(ctx, &models.Snippet{
		Title: "pool", Project: acceptPtr("apollo"), Language: "go", Content: "pgxpool.New(ctx, dsn)",
	}))
	must("journal", st.CreateJournalEntry(ctx, &models.JournalEntry{
		Summary: "kickoff", Project: acceptPtr("apollo"), SessionRef: "sp-1",
	}))
	must("solution", st.CreateSolution(ctx, &models.Solution{
		ErrorDescription: "TLS handshake", Solution: "trust the mkcert CA",
	}))
	must("memory", st.SetMemory(ctx, &models.Memory{
		Scope: models.ScopeProject, Project: acceptPtr("apollo"), Key: "db", Value: "sqlite",
	}))
	must("env", st.SetEnv(ctx, &models.EnvEntry{Project: "apollo", Key: "API_PORT", Value: "8443"}))
}

// TestLocalLifecycleAcceptance is the complete local-product acceptance pass
// (roadmap P8-t7): fresh install → offline (lexical-only) search → idempotent
// upgrade → backup → restore into a clean install → lossless round-trip
// verification, all on the self-contained SQLite backend a beta tester installs.
// It composes the real migration, store, FTS search, and backup code paths so a
// regression anywhere in the lifecycle fails one CI test.
func TestLocalLifecycleAcceptance(t *testing.T) {
	ctx := context.Background()

	// 1. Fresh install: migrate an empty database to head and open the store.
	dsn := "sqlite:" + filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("fresh-install migrate: %v", err)
	}
	src, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(src.Close)
	seedAcceptance(t, src)

	// 2. Offline mode: lexical FTS search must work with no embedding provider.
	if hits, err := src.SearchDocuments(ctx, "coordinator", store.Filter{}); err != nil || len(hits) == 0 {
		t.Fatalf("offline lexical search returned nothing (len=%d err=%v) — FTS-only mode broken", len(hits), err)
	}

	// 3. Upgrade: re-running migrations on a populated database is a no-op and
	// leaves data intact.
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("upgrade (re-migrate) not idempotent: %v", err)
	}
	if docs, err := src.ListDocuments(ctx, store.Filter{}); err != nil || len(docs) == 0 {
		t.Fatalf("documents lost across upgrade: len=%d err=%v", len(docs), err)
	}

	// 4. Backup: export all knowledge and serialize it to backup-file bytes.
	arch, err := backup.Export(ctx, src)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	var buf bytes.Buffer
	if err := arch.Write(&buf); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	// 5. Restore into a clean install from the serialized backup file.
	dstDSN := "sqlite:" + filepath.Join(t.TempDir(), "restored.db")
	if err := migrations.Up(dstDSN); err != nil {
		t.Fatalf("restore-target migrate: %v", err)
	}
	dst, err := store.New(ctx, dstDSN)
	if err != nil {
		t.Fatalf("open restored store: %v", err)
	}
	t.Cleanup(dst.Close)
	readBack, err := backup.Read(&buf)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if _, err := backup.Import(ctx, dst, readBack); err != nil {
		t.Fatalf("import: %v", err)
	}

	// 6. Round-trip verification: the restored install holds the same knowledge.
	reExport, err := backup.Export(ctx, dst)
	if err != nil {
		t.Fatalf("re-export: %v", err)
	}
	if err := backup.Verify(arch, reExport); err != nil {
		t.Errorf("restore is not lossless: %v", err)
	}
	// And the restored install is fully usable offline.
	if hits, err := dst.SearchDocuments(ctx, "coordinator", store.Filter{}); err != nil || len(hits) == 0 {
		t.Fatalf("restored install offline search broken: len=%d err=%v", len(hits), err)
	}
}
