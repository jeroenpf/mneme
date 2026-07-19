package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jeroenpfeil/mneme/internal/migrations"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// The conformance suite runs the Store interface contract against every backend
// so both Postgres and SQLite are held to identical behaviour (plan p3-t1). It
// asserts feature parity — not identical ranking, which the search parity tests
// (P5) cover separately. Postgres reuses the shared container (TRUNCATE-isolated
// per subtest); SQLite gets a fresh temp file per subtest.

// storeBackend names a Store implementation and how to build a clean one.
type storeBackend struct {
	name string
	make func(t *testing.T) store.Store
}

// conformanceBackends is the set every conformance test runs against.
func conformanceBackends() []storeBackend {
	return []storeBackend{
		{"postgres", newPostgresConformanceStore},
		{"sqlite", newSQLiteConformanceStore},
	}
}

// newPostgresConformanceStore reuses the package-shared container, truncating
// every table so the subtest starts clean.
func newPostgresConformanceStore(t *testing.T) store.Store {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx,
		`TRUNCATE documents, projects, decisions, snippets, journal_entries,
		         solutions, memories, env_entries, embeddings, embed_failures
		 RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return store.NewWithPool(pool)
}

// newSQLiteConformanceStore migrates a fresh temp .db and opens it.
func newSQLiteConformanceStore(t *testing.T) store.Store {
	t.Helper()
	dsn := "sqlite:" + filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

// forEachBackend runs fn as a subtest against every backend with a clean store.
func forEachBackend(t *testing.T, fn func(t *testing.T, st store.Store)) {
	t.Helper()
	for _, b := range conformanceBackends() {
		t.Run(b.name, func(t *testing.T) { fn(t, b.make(t)) })
	}
}

// seedProjectsIfc creates projects through the Store interface so seeding works
// on both backends (the Postgres tests' seedProjects reaches into the pool).
func seedProjectsIfc(t *testing.T, st store.Store, slugs ...string) {
	t.Helper()
	ctx := context.Background()
	for _, slug := range slugs {
		if err := st.CreateProject(ctx, &models.Project{Name: slug, Slug: slug}); err != nil {
			t.Fatalf("seed project %q: %v", slug, err)
		}
	}
}

func TestConformanceDocumentCRUD(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		doc := sampleDoc("doc-001", "Vehicle Listing API")
		doc.Project = ptr("apollo")
		doc.Ticket = ptr("C1-142")
		if err := st.CreateDocument(ctx, doc); err != nil {
			t.Fatalf("CreateDocument: %v", err)
		}
		if doc.CreatedAt.IsZero() || doc.UpdatedAt.IsZero() {
			t.Fatal("Create did not populate CreatedAt/UpdatedAt")
		}
		if !strings.HasPrefix(doc.PublicID, "doc_") {
			t.Errorf("public_id: got %q, want doc_ prefix", doc.PublicID)
		}

		got, err := st.GetDocument(ctx, "doc-001")
		if err != nil {
			t.Fatalf("GetDocument: %v", err)
		}
		if got.Title != "Vehicle Listing API" {
			t.Errorf("title: got %q", got.Title)
		}
		if got.Project == nil || *got.Project != "apollo" {
			t.Errorf("project: got %v", got.Project)
		}
		if got.Ticket == nil || *got.Ticket != "C1-142" {
			t.Errorf("ticket: got %v", got.Ticket)
		}
		if len(got.Tags) != 2 || got.Tags[0] != "go" {
			t.Errorf("tags: got %v", got.Tags)
		}
		if got.Meta["phases"] == nil {
			t.Errorf("meta.phases not roundtripped: %+v", got.Meta)
		}
		if got.Body["sections"] == nil {
			t.Errorf("body.sections not roundtripped: %+v", got.Body)
		}

		// Update mutates fields and advances updated_at. A short pause makes
		// the strict After() assertion deterministic on SQLite, whose strftime
		// timestamps are millisecond-precision (Postgres is microsecond).
		time.Sleep(2 * time.Millisecond)
		got.Title = "Vehicle Listing API v2"
		got.Status = models.StatusInProgress
		got.Tags = []string{"go"}
		if err := st.UpdateDocument(ctx, got, nil); err != nil {
			t.Fatalf("UpdateDocument: %v", err)
		}
		reloaded, err := st.GetDocument(ctx, "doc-001")
		if err != nil {
			t.Fatalf("GetDocument after update: %v", err)
		}
		if reloaded.Title != "Vehicle Listing API v2" {
			t.Errorf("updated title: got %q", reloaded.Title)
		}
		if reloaded.Status != models.StatusInProgress {
			t.Errorf("updated status: got %q", reloaded.Status)
		}
		if len(reloaded.Tags) != 1 {
			t.Errorf("updated tags: got %v", reloaded.Tags)
		}
		if !reloaded.UpdatedAt.After(reloaded.CreatedAt) {
			t.Errorf("updated_at %v should be after created_at %v", reloaded.UpdatedAt, reloaded.CreatedAt)
		}

		// Archive.
		if err := st.ArchiveDocument(ctx, "doc-001"); err != nil {
			t.Fatalf("ArchiveDocument: %v", err)
		}
		archived, _ := st.GetDocument(ctx, "doc-001")
		if archived.Status != models.StatusArchived {
			t.Errorf("archived status: got %q", archived.Status)
		}
	})
}

func TestConformanceDocumentErrors(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()

		// Missing → ErrNotFound.
		if _, err := st.GetDocument(ctx, "nope"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetDocument(missing): got %v, want ErrNotFound", err)
		}
		if err := st.UpdateDocument(ctx, sampleDoc("ghost", "Ghost"), nil); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("UpdateDocument(missing): got %v, want ErrNotFound", err)
		}
		if err := st.ArchiveDocument(ctx, "ghost"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("ArchiveDocument(missing): got %v, want ErrNotFound", err)
		}

		// Unknown project → ErrInvalidProject.
		bad := sampleDoc("doc-x", "X")
		bad.Project = ptr("ghost-project")
		if err := st.CreateDocument(ctx, bad); !errors.Is(err, store.ErrInvalidProject) {
			t.Errorf("CreateDocument(unknown project): got %v, want ErrInvalidProject", err)
		}

		// Duplicate id → ErrDuplicateID.
		if err := st.CreateDocument(ctx, sampleDoc("dup", "First")); err != nil {
			t.Fatalf("CreateDocument: %v", err)
		}
		if err := st.CreateDocument(ctx, sampleDoc("dup", "Second")); !errors.Is(err, store.ErrDuplicateID) {
			t.Errorf("CreateDocument(dup id): got %v, want ErrDuplicateID", err)
		}
	})
}

func TestConformanceDocumentRevisions(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		doc := sampleDoc("rev-001", "Rev doc")
		doc.Project = ptr("apollo")
		if err := st.CreateDocument(ctx, doc); err != nil {
			t.Fatalf("CreateDocument: %v", err)
		}
		if doc.Revision != 1 {
			t.Fatalf("new document revision = %d, want 1", doc.Revision)
		}

		// Unconditional update bumps the revision.
		doc.Title = "Rev doc v2"
		if err := st.UpdateDocument(ctx, doc, nil); err != nil {
			t.Fatalf("UpdateDocument(nil): %v", err)
		}
		if doc.Revision != 2 {
			t.Errorf("revision after update = %d, want 2", doc.Revision)
		}
		got, _ := st.GetDocument(ctx, "rev-001")
		if got.Revision != 2 {
			t.Errorf("reloaded revision = %d, want 2", got.Revision)
		}

		// Conditional update with the correct expected revision succeeds.
		got.Title = "Rev doc v3"
		if err := st.UpdateDocument(ctx, got, ptr(2)); err != nil {
			t.Fatalf("UpdateDocument(expected=2): %v", err)
		}
		if got.Revision != 3 {
			t.Errorf("revision after conditional update = %d, want 3", got.Revision)
		}

		// A stale expected revision is rejected with a typed conflict carrying
		// the current revision, and leaves the row untouched.
		stale, _ := st.GetDocument(ctx, "rev-001")
		stale.Title = "should not persist"
		err := st.UpdateDocument(ctx, stale, ptr(2))
		if !errors.Is(err, store.ErrRevisionConflict) {
			t.Fatalf("stale update: got %v, want ErrRevisionConflict", err)
		}
		var conflict *store.RevisionConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("conflict error is not *RevisionConflictError: %T", err)
		}
		if conflict.Current != 3 || conflict.DocumentID != "rev-001" {
			t.Errorf("conflict details = %+v, want current=3 id=rev-001", conflict)
		}
		reloaded, _ := st.GetDocument(ctx, "rev-001")
		if reloaded.Title != "Rev doc v3" || reloaded.Revision != 3 {
			t.Errorf("stale update must not persist: title=%q rev=%d", reloaded.Title, reloaded.Revision)
		}
	})
}

func TestConformanceRevisionConflictReportsChangedIDs(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		doc := sampleDoc("cc-1", "CC doc")
		doc.Project = ptr("apollo")
		if err := st.CreateDocument(ctx, doc); err != nil { // rev 1
			t.Fatalf("CreateDocument: %v", err)
		}
		// A concurrent write advances to rev 2 and records which block it changed.
		doc.Title = "CC v2"
		if err := st.UpdateDocument(ctx, doc, nil); err != nil { // rev 2
			t.Fatalf("UpdateDocument: %v", err)
		}
		if err := st.AppendDocumentRevision(ctx, &models.DocumentRevision{
			DocumentID: "cc-1", Revision: 2, Op: "update_section", Actor: "mcp",
			TargetIDs: []string{"blk-7"}, Title: doc.Title, Status: doc.Status,
			Meta: doc.Meta, Body: doc.Body,
		}); err != nil {
			t.Fatalf("AppendDocumentRevision: %v", err)
		}

		// A stale writer based on rev 1 conflicts and learns the current revision
		// AND which ids changed since their base.
		stale, _ := st.GetDocument(ctx, "cc-1")
		stale.Title = "stale"
		err := st.UpdateDocument(ctx, stale, ptr(1))
		var conflict *store.RevisionConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("expected RevisionConflictError, got %v", err)
		}
		if conflict.Current != 2 {
			t.Errorf("conflict current = %d, want 2", conflict.Current)
		}
		if len(conflict.ChangedIDs) != 1 || conflict.ChangedIDs[0] != "blk-7" {
			t.Errorf("conflict changed ids = %v, want [blk-7]", conflict.ChangedIDs)
		}
	})
}

func TestConformanceDocumentRevisionSnapshots(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		doc := sampleDoc("snap-001", "Snap doc")
		doc.Project = ptr("apollo")
		if err := st.CreateDocument(ctx, doc); err != nil {
			t.Fatalf("CreateDocument: %v", err)
		}

		// Record two snapshots for the document.
		r1 := &models.DocumentRevision{
			DocumentID: "snap-001", Revision: 1, Op: "push_document", Actor: "mcp",
			TargetIDs: []string{"snap-001"}, Title: doc.Title, Status: doc.Status,
			Meta: doc.Meta, Body: doc.Body,
		}
		if err := st.AppendDocumentRevision(ctx, r1); err != nil {
			t.Fatalf("AppendDocumentRevision r1: %v", err)
		}
		if r1.ID == 0 || r1.CreatedAt.IsZero() {
			t.Errorf("append did not populate ID/CreatedAt: %+v", r1)
		}
		r2 := &models.DocumentRevision{
			DocumentID: "snap-001", Revision: 2, Op: "tick_task", Actor: "mcp",
			TargetIDs: []string{"task-9"}, Title: "Snap doc v2", Status: "in-progress",
			Meta: map[string]any{}, Body: map[string]any{"sections": []any{}},
		}
		if err := st.AppendDocumentRevision(ctx, r2); err != nil {
			t.Fatalf("AppendDocumentRevision r2: %v", err)
		}

		// A duplicate (document_id, revision) is rejected.
		if err := st.AppendDocumentRevision(ctx, r1); err == nil {
			t.Errorf("duplicate revision should be rejected")
		}

		// List returns newest-first.
		list, err := st.ListDocumentRevisions(ctx, "snap-001", 0)
		if err != nil {
			t.Fatalf("ListDocumentRevisions: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("revision count = %d, want 2", len(list))
		}
		if list[0].Revision != 2 || list[1].Revision != 1 {
			t.Errorf("not newest-first: %d, %d", list[0].Revision, list[1].Revision)
		}
		if list[0].Op != "tick_task" || len(list[0].TargetIDs) != 1 || list[0].TargetIDs[0] != "task-9" {
			t.Errorf("snapshot metadata lost: %+v", list[0])
		}

		// Get one revision round-trips the body snapshot.
		got, err := st.GetDocumentRevision(ctx, "snap-001", 1)
		if err != nil {
			t.Fatalf("GetDocumentRevision: %v", err)
		}
		if got.Title != "Snap doc" || got.Body["sections"] == nil {
			t.Errorf("revision 1 snapshot not round-tripped: %+v", got)
		}

		// Missing revision → ErrNotFound.
		if _, err := st.GetDocumentRevision(ctx, "snap-001", 99); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("missing revision: got %v, want ErrNotFound", err)
		}
	})
}

func TestConformanceListDocuments(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo", "zephyr")

		a := sampleDoc("d-a", "Alpha")
		a.Project = ptr("apollo")
		a.Status = models.StatusTodo
		b := sampleDoc("d-b", "Beta")
		b.Project = ptr("zephyr")
		b.Status = models.StatusComplete
		b.Type = models.TypeSpec
		for _, d := range []*models.Document{a, b} {
			if err := st.CreateDocument(ctx, d); err != nil {
				t.Fatalf("create %s: %v", d.ID, err)
			}
		}

		all, err := st.ListDocuments(ctx, store.Filter{})
		if err != nil {
			t.Fatalf("ListDocuments: %v", err)
		}
		if len(all) != 2 {
			t.Fatalf("ListDocuments: got %d, want 2", len(all))
		}

		apollo := ptr("apollo")
		got, err := st.ListDocuments(ctx, store.Filter{Project: apollo})
		if err != nil {
			t.Fatalf("ListDocuments(project): %v", err)
		}
		if len(got) != 1 || got[0].ID != "d-a" {
			t.Errorf("project filter: got %v", got)
		}

		spec := models.TypeSpec
		got, err = st.ListDocuments(ctx, store.Filter{Type: &spec})
		if err != nil {
			t.Fatalf("ListDocuments(type): %v", err)
		}
		if len(got) != 1 || got[0].ID != "d-b" {
			t.Errorf("type filter: got %v", got)
		}
	})
}

func TestConformanceDecisions(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		d := &models.Decision{
			Title: "Use RRF for fusion", Project: ptr("apollo"),
			Decision:  "Fuse FTS and vector via reciprocal rank",
			Rationale: "Rank-only fusion is dialect-free", Status: models.DecisionAccepted,
		}
		if err := st.CreateDecision(ctx, d); err != nil {
			t.Fatalf("CreateDecision: %v", err)
		}
		if d.ID == "" || !strings.HasPrefix(d.PublicID, "dec_") || d.CreatedAt.IsZero() {
			t.Errorf("Create did not populate id/public_id/created_at: %+v", d)
		}

		got, err := st.GetDecision(ctx, d.ID)
		if err != nil {
			t.Fatalf("GetDecision: %v", err)
		}
		if got.Title != "Use RRF for fusion" || got.Rationale != "Rank-only fusion is dialect-free" {
			t.Errorf("roundtrip: %+v", got)
		}

		// Global decision (nil project).
		g := &models.Decision{Title: "Global", Decision: "x", Status: models.DecisionProposed}
		if err := st.CreateDecision(ctx, g); err != nil {
			t.Fatalf("CreateDecision(global): %v", err)
		}
		if got, _ := st.GetDecision(ctx, g.ID); got.Project != nil {
			t.Errorf("global project should be nil, got %v", got.Project)
		}

		// Update.
		got.Status = models.DecisionDeprecated
		got.Consequences = "superseded"
		if err := st.UpdateDecision(ctx, got); err != nil {
			t.Fatalf("UpdateDecision: %v", err)
		}
		if re, _ := st.GetDecision(ctx, d.ID); re.Status != models.DecisionDeprecated || re.Consequences != "superseded" {
			t.Errorf("update not persisted: %+v", re)
		}

		// List: filter by status.
		acc := models.DecisionDeprecated
		list, err := st.ListDecisions(ctx, store.DecisionFilter{Status: &acc})
		if err != nil {
			t.Fatalf("ListDecisions: %v", err)
		}
		if len(list) != 1 || list[0].ID != d.ID {
			t.Errorf("status filter: got %d results", len(list))
		}

		// Errors.
		if err := st.CreateDecision(ctx, &models.Decision{Title: "x", Decision: "y", Project: ptr("ghost"), Status: models.DecisionAccepted}); !errors.Is(err, store.ErrInvalidProject) {
			t.Errorf("unknown project: got %v", err)
		}
		// A syntactically valid but absent id: Postgres decisions.id is a UUID,
		// so a malformed value would raise a parse error rather than ErrNotFound.
		const missingID = "00000000-0000-0000-0000-000000000000"
		if err := st.UpdateDecision(ctx, &models.Decision{ID: missingID, Title: "x", Decision: "y", Status: models.DecisionAccepted}); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("update missing: got %v", err)
		}
		if _, err := st.GetDecision(ctx, missingID); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("get missing: got %v", err)
		}
	})
}

func TestConformanceSnippets(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		sn := &models.Snippet{
			Title: "pgx pool", Project: ptr("apollo"), Language: "go",
			Content: "pgxpool.New(ctx, dsn)", Tags: []string{"db", "go"},
			Description: "open a pool",
		}
		if err := st.CreateSnippet(ctx, sn); err != nil {
			t.Fatalf("CreateSnippet: %v", err)
		}
		if sn.ID == "" || !strings.HasPrefix(sn.PublicID, "snip_") {
			t.Errorf("Create did not populate id/public_id: %+v", sn)
		}
		got, err := st.GetSnippet(ctx, sn.ID)
		if err != nil {
			t.Fatalf("GetSnippet: %v", err)
		}
		if got.Content != "pgxpool.New(ctx, dsn)" || len(got.Tags) != 2 {
			t.Errorf("roundtrip: %+v", got)
		}

		got.Language = "golang"
		got.Tags = []string{"db"}
		if err := st.UpdateSnippet(ctx, got); err != nil {
			t.Fatalf("UpdateSnippet: %v", err)
		}
		if re, _ := st.GetSnippet(ctx, sn.ID); re.Language != "golang" || len(re.Tags) != 1 {
			t.Errorf("update not persisted: %+v", re)
		}

		// Tags filter (ALL tags must match).
		list, err := st.ListSnippets(ctx, store.SnippetFilter{Tags: []string{"db"}})
		if err != nil {
			t.Fatalf("ListSnippets: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("tags filter: got %d", len(list))
		}
		none, _ := st.ListSnippets(ctx, store.SnippetFilter{Tags: []string{"missing"}})
		if len(none) != 0 {
			t.Errorf("tags filter (absent): got %d", len(none))
		}

		if err := st.CreateSnippet(ctx, &models.Snippet{Title: "x", Content: "y", Project: ptr("ghost")}); !errors.Is(err, store.ErrInvalidProject) {
			t.Errorf("unknown project: got %v", err)
		}
	})
}

func TestConformanceJournal(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		e := &models.JournalEntry{
			Project: ptr("apollo"), SessionRef: "s4-t3", Summary: "CRUD parity",
			Accomplished: []string{"documents", "projects"}, Deferred: []string{"search"},
		}
		if err := st.CreateJournalEntry(ctx, e); err != nil {
			t.Fatalf("CreateJournalEntry: %v", err)
		}
		if e.ID == "" || !strings.HasPrefix(e.PublicID, "jrnl_") {
			t.Errorf("Create did not populate id/public_id: %+v", e)
		}
		got, err := st.GetJournalEntry(ctx, e.ID)
		if err != nil {
			t.Fatalf("GetJournalEntry: %v", err)
		}
		if got.Summary != "CRUD parity" || len(got.Accomplished) != 2 || len(got.Deferred) != 1 {
			t.Errorf("roundtrip: %+v", got)
		}

		got.Summary = "CRUD parity done"
		got.Deferred = []string{}
		if err := st.UpdateJournalEntry(ctx, got); err != nil {
			t.Fatalf("UpdateJournalEntry: %v", err)
		}
		if re, _ := st.GetJournalEntry(ctx, e.ID); re.Summary != "CRUD parity done" || len(re.Deferred) != 0 {
			t.Errorf("update not persisted: %+v", re)
		}

		list, err := st.ListJournalEntries(ctx, store.JournalFilter{Project: ptr("apollo")})
		if err != nil {
			t.Fatalf("ListJournalEntries: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("project filter: got %d", len(list))
		}
	})
}

func TestConformanceSolutions(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		sol := &models.Solution{
			Project: ptr("apollo"), ErrorDescription: "mDNS stalls resolution",
			Solution: "use mneme.dev not .local", Tags: []string{"dns"},
			SourceURL: "https://example.test",
		}
		if err := st.CreateSolution(ctx, sol); err != nil {
			t.Fatalf("CreateSolution: %v", err)
		}
		if sol.ID == "" || !strings.HasPrefix(sol.PublicID, "sol_") {
			t.Errorf("Create did not populate id/public_id: %+v", sol)
		}
		got, err := st.GetSolution(ctx, sol.ID)
		if err != nil {
			t.Fatalf("GetSolution: %v", err)
		}
		if got.Solution != "use mneme.dev not .local" || got.SourceURL != "https://example.test" {
			t.Errorf("roundtrip: %+v", got)
		}

		got.Solution = "add /etc/hosts entry"
		if err := st.UpdateSolution(ctx, got); err != nil {
			t.Fatalf("UpdateSolution: %v", err)
		}
		if re, _ := st.GetSolution(ctx, sol.ID); re.Solution != "add /etc/hosts entry" {
			t.Errorf("update not persisted: %+v", re)
		}

		list, err := st.ListSolutions(ctx, store.SolutionFilter{Tags: []string{"dns"}})
		if err != nil {
			t.Fatalf("ListSolutions: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("tags filter: got %d", len(list))
		}
	})
}

func TestConformanceMemory(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		// Insert then upsert (same identity) updates value, not a new row.
		m := &models.Memory{Scope: models.ScopeGlobal, Key: "editor", Value: "vim"}
		if err := st.SetMemory(ctx, m); err != nil {
			t.Fatalf("SetMemory: %v", err)
		}
		if m.ID == "" || m.UpdatedAt.IsZero() {
			t.Errorf("SetMemory did not populate id/updated_at: %+v", m)
		}
		firstID := m.ID
		m2 := &models.Memory{Scope: models.ScopeGlobal, Key: "editor", Value: "emacs"}
		if err := st.SetMemory(ctx, m2); err != nil {
			t.Fatalf("SetMemory(upsert): %v", err)
		}
		if m2.ID != firstID {
			t.Errorf("upsert should reuse row id: got %q want %q", m2.ID, firstID)
		}

		// Project-scoped entry.
		pm := &models.Memory{Scope: models.ScopeProject, Project: ptr("apollo"), Key: "stack", Value: "go"}
		if err := st.SetMemory(ctx, pm); err != nil {
			t.Fatalf("SetMemory(project): %v", err)
		}

		all, err := st.ListMemory(ctx, store.MemoryFilter{})
		if err != nil {
			t.Fatalf("ListMemory: %v", err)
		}
		if len(all) != 2 {
			t.Fatalf("ListMemory: got %d want 2", len(all))
		}
		globalScope := models.ScopeGlobal
		globals, _ := st.ListMemory(ctx, store.MemoryFilter{Scope: &globalScope})
		if len(globals) != 1 || globals[0].Value != "emacs" {
			t.Errorf("scope filter: %+v", globals)
		}

		// Delete the global entry.
		if err := st.DeleteMemory(ctx, models.ScopeGlobal, nil, nil, "editor"); err != nil {
			t.Fatalf("DeleteMemory: %v", err)
		}
		if err := st.DeleteMemory(ctx, models.ScopeGlobal, nil, nil, "editor"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("DeleteMemory(again): got %v want ErrNotFound", err)
		}

		// Unknown project → ErrInvalidProject.
		if err := st.SetMemory(ctx, &models.Memory{Scope: models.ScopeProject, Project: ptr("ghost"), Key: "k", Value: "v"}); !errors.Is(err, store.ErrInvalidProject) {
			t.Errorf("unknown project: got %v", err)
		}
	})
}

func TestConformanceEnv(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		e := &models.EnvEntry{Project: "apollo", Key: "PORT", Value: "8443", Description: ptr("tls port")}
		if err := st.SetEnv(ctx, e); err != nil {
			t.Fatalf("SetEnv: %v", err)
		}
		if e.ID == "" || e.UpdatedAt.IsZero() {
			t.Errorf("SetEnv did not populate id/updated_at: %+v", e)
		}
		firstID := e.ID

		// Upsert replaces value AND description, reusing the row.
		e2 := &models.EnvEntry{Project: "apollo", Key: "PORT", Value: "9443"}
		if err := st.SetEnv(ctx, e2); err != nil {
			t.Fatalf("SetEnv(upsert): %v", err)
		}
		if e2.ID != firstID {
			t.Errorf("upsert should reuse row id")
		}
		list, err := st.ListEnv(ctx, "apollo")
		if err != nil {
			t.Fatalf("ListEnv: %v", err)
		}
		if len(list) != 1 || list[0].Value != "9443" || list[0].Description != nil {
			t.Errorf("upsert result: %+v", list)
		}

		if err := st.DeleteEnv(ctx, "apollo", "PORT"); err != nil {
			t.Fatalf("DeleteEnv: %v", err)
		}
		if err := st.DeleteEnv(ctx, "apollo", "PORT"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("DeleteEnv(again): got %v want ErrNotFound", err)
		}

		if err := st.SetEnv(ctx, &models.EnvEntry{Project: "ghost", Key: "K", Value: "V"}); !errors.Is(err, store.ErrInvalidProject) {
			t.Errorf("unknown project: got %v", err)
		}
	})
}

// TestConformanceGetByPublicID exercises the by-public-id lookups every backend
// must provide so resolve_reference can turn a pasted mneme:// id into an
// entity. Each type resolves its own generated public id back to the same row,
// and a well-formed but absent id yields ErrNotFound.
func TestConformanceGetByPublicID(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		doc := sampleDoc("pid-doc", "By-id Doc")
		doc.Project = ptr("apollo")
		if err := st.CreateDocument(ctx, doc); err != nil {
			t.Fatalf("CreateDocument: %v", err)
		}
		if got, err := st.GetDocumentByPublicID(ctx, doc.PublicID); err != nil || got.ID != "pid-doc" {
			t.Errorf("GetDocumentByPublicID(%q): got %+v err %v", doc.PublicID, got, err)
		}

		dec := &models.Decision{Title: "D", Decision: "x", Status: models.DecisionAccepted, Project: ptr("apollo")}
		if err := st.CreateDecision(ctx, dec); err != nil {
			t.Fatalf("CreateDecision: %v", err)
		}
		if got, err := st.GetDecisionByPublicID(ctx, dec.PublicID); err != nil || got.ID != dec.ID {
			t.Errorf("GetDecisionByPublicID(%q): got %+v err %v", dec.PublicID, got, err)
		}

		sn := &models.Snippet{Title: "S", Content: "code", Language: "go", Project: ptr("apollo")}
		if err := st.CreateSnippet(ctx, sn); err != nil {
			t.Fatalf("CreateSnippet: %v", err)
		}
		if got, err := st.GetSnippetByPublicID(ctx, sn.PublicID); err != nil || got.ID != sn.ID {
			t.Errorf("GetSnippetByPublicID(%q): got %+v err %v", sn.PublicID, got, err)
		}

		je := &models.JournalEntry{Project: ptr("apollo"), Summary: "did stuff"}
		if err := st.CreateJournalEntry(ctx, je); err != nil {
			t.Fatalf("CreateJournalEntry: %v", err)
		}
		if got, err := st.GetJournalEntryByPublicID(ctx, je.PublicID); err != nil || got.ID != je.ID {
			t.Errorf("GetJournalEntryByPublicID(%q): got %+v err %v", je.PublicID, got, err)
		}

		sol := &models.Solution{ErrorDescription: "boom", Solution: "fix", Project: ptr("apollo")}
		if err := st.CreateSolution(ctx, sol); err != nil {
			t.Fatalf("CreateSolution: %v", err)
		}
		if got, err := st.GetSolutionByPublicID(ctx, sol.PublicID); err != nil || got.ID != sol.ID {
			t.Errorf("GetSolutionByPublicID(%q): got %+v err %v", sol.PublicID, got, err)
		}

		prj, err := st.GetProject(ctx, "apollo")
		if err != nil {
			t.Fatalf("GetProject: %v", err)
		}
		if got, err := st.GetProjectByPublicID(ctx, prj.PublicID); err != nil || got.Slug != "apollo" {
			t.Errorf("GetProjectByPublicID(%q): got %+v err %v", prj.PublicID, got, err)
		}

		// Well-formed but absent public ids resolve to ErrNotFound.
		missing := map[string]func(string) error{
			"doc_000000000000":  func(id string) error { _, e := st.GetDocumentByPublicID(ctx, id); return e },
			"dec_000000000000":  func(id string) error { _, e := st.GetDecisionByPublicID(ctx, id); return e },
			"snip_000000000000": func(id string) error { _, e := st.GetSnippetByPublicID(ctx, id); return e },
			"jrnl_000000000000": func(id string) error { _, e := st.GetJournalEntryByPublicID(ctx, id); return e },
			"sol_000000000000":  func(id string) error { _, e := st.GetSolutionByPublicID(ctx, id); return e },
			"prj_000000000000":  func(id string) error { _, e := st.GetProjectByPublicID(ctx, id); return e },
		}
		for id, lookup := range missing {
			if err := lookup(id); !errors.Is(err, store.ErrNotFound) {
				t.Errorf("lookup(%q) absent: got %v, want ErrNotFound", id, err)
			}
		}
	})
}

// vecOf builds an n-dimensional vector filled with val, for embedding tests.
func vecOf(n int, val float32) []float32 {
	v := make([]float32, n)
	for i := range v {
		v[i] = val
	}
	return v
}

// unitVec builds an n-dimensional basis vector: 1 at idx, 0 elsewhere. Two
// distinct basis vectors are orthogonal (cosine 0); a vector with itself is 1.
func unitVec(n, idx int) []float32 {
	v := make([]float32, n)
	v[idx] = 1
	return v
}

// hasHit reports whether a SearchHit of the given type and id is present.
func hasHit(hits []*models.SearchHit, typ, id string) bool {
	return findHit(hits, typ, id) != nil
}

// findHit returns the SearchHit of the given type and id, or nil.
func findHit(hits []*models.SearchHit, typ, id string) *models.SearchHit {
	for _, h := range hits {
		if h.Type == typ && h.ID == id {
			return h
		}
	}
	return nil
}

func TestConformanceSearchHitsCarryPublicID(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		doc := sampleDoc("pub-1", "Zebra indexing guide")
		doc.Project = ptr("apollo")
		if err := st.CreateDocument(ctx, doc); err != nil {
			t.Fatalf("create doc: %v", err)
		}
		dec := &models.Decision{Title: "Adopt Wombat cache", Decision: "use the wombat cache", Status: models.DecisionAccepted, Project: ptr("apollo")}
		if err := st.CreateDecision(ctx, dec); err != nil {
			t.Fatalf("create decision: %v", err)
		}
		if err := st.SetMemory(ctx, &models.Memory{Scope: models.ScopeProject, Project: ptr("apollo"), Key: "quokka", Value: "the quokka setting"}); err != nil {
			t.Fatalf("set memory: %v", err)
		}

		// A document hit carries the document's doc_ public id.
		hits, err := st.Search(ctx, "Zebra", store.SearchFilter{})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		h := findHit(hits, "documents", "pub-1")
		if h == nil {
			t.Fatalf(`Search("Zebra") missing pub-1: %+v`, hits)
		}
		if h.PublicID != doc.PublicID || h.PublicID == "" {
			t.Errorf("document hit public_id = %q, want %q", h.PublicID, doc.PublicID)
		}

		// A decision hit carries the decision's dec_ public id.
		hits, _ = st.Search(ctx, "Wombat", store.SearchFilter{})
		h = findHit(hits, "decisions", dec.ID)
		if h == nil {
			t.Fatalf(`Search("Wombat") missing decision: %+v`, hits)
		}
		if h.PublicID != dec.PublicID || h.PublicID == "" {
			t.Errorf("decision hit public_id = %q, want %q", h.PublicID, dec.PublicID)
		}

		// Memory has no public id; its hit's PublicID stays empty (deep-linked by key).
		hits, _ = st.Search(ctx, "quokka", store.SearchFilter{})
		for _, mh := range hits {
			if mh.Type == "memory" && mh.PublicID != "" {
				t.Errorf("memory hit should have empty public_id, got %q", mh.PublicID)
			}
		}
	})
}

func TestConformanceUnifiedSearch(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo", "zephyr")

		d1 := sampleDoc("srch-1", "PostgreSQL indexing guide")
		d1.Project = ptr("apollo")
		d2 := sampleDoc("srch-2", "Sourdough bread recipe")
		d2.Project = ptr("zephyr")
		for _, d := range []*models.Document{d1, d2} {
			if err := st.CreateDocument(ctx, d); err != nil {
				t.Fatalf("create %s: %v", d.ID, err)
			}
		}
		dec := &models.Decision{Title: "Adopt pgvector", Decision: "use pgvector for embeddings", Status: models.DecisionAccepted, Project: ptr("apollo")}
		if err := st.CreateDecision(ctx, dec); err != nil {
			t.Fatalf("create decision: %v", err)
		}

		// A distinctive term surfaces the matching document, not the other.
		hits, err := st.Search(ctx, "PostgreSQL", store.SearchFilter{})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if !hasHit(hits, "documents", "srch-1") {
			t.Errorf(`Search("PostgreSQL") missing srch-1: %+v`, hits)
		}
		if hasHit(hits, "documents", "srch-2") {
			t.Errorf(`Search("PostgreSQL") should not surface srch-2`)
		}

		// Cross-type: a decision surfaces for its own term.
		hits, _ = st.Search(ctx, "pgvector", store.SearchFilter{})
		if !hasHit(hits, "decisions", dec.ID) {
			t.Errorf(`Search("pgvector") missing decision hit: %+v`, hits)
		}

		// Project filter honored (scope includes globals; srch-2 is zephyr-only).
		hits, _ = st.Search(ctx, "bread", store.SearchFilter{Project: ptr("apollo")})
		if hasHit(hits, "documents", "srch-2") {
			t.Errorf(`Search("bread", apollo) should exclude zephyr's srch-2`)
		}
		hits, _ = st.Search(ctx, "bread", store.SearchFilter{Project: ptr("zephyr")})
		if !hasHit(hits, "documents", "srch-2") {
			t.Errorf(`Search("bread", zephyr) missing srch-2: %+v`, hits)
		}

		// Type filter restricts to the requested types.
		hits, _ = st.Search(ctx, "PostgreSQL", store.SearchFilter{Types: []string{"decisions"}})
		if hasHit(hits, "documents", "srch-1") {
			t.Errorf("type filter should exclude documents")
		}

		// Degenerate queries never error and return no FTS hits.
		for _, q := range []string{"", "  ", "-", `"`, "((()))", "-only"} {
			if _, err := st.Search(ctx, q, store.SearchFilter{}); err != nil {
				t.Errorf("Search(%q) errored: %v", q, err)
			}
		}
	})
}

func TestConformanceTypeScopedSearch(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")

		d := sampleDoc("ts-1", "Kubernetes networking deep dive")
		d.Project = ptr("apollo")
		if err := st.CreateDocument(ctx, d); err != nil {
			t.Fatalf("create doc: %v", err)
		}
		docs, err := st.SearchDocuments(ctx, "Kubernetes", store.Filter{})
		if err != nil {
			t.Fatalf("SearchDocuments: %v", err)
		}
		if len(docs) != 1 || docs[0].ID != "ts-1" {
			t.Errorf("SearchDocuments(Kubernetes): %+v", docs)
		}
		if none, _ := st.SearchDocuments(ctx, "nonexistentxyz", store.Filter{}); len(none) != 0 {
			t.Errorf("SearchDocuments(absent): got %d", len(none))
		}
		// Filter honored: wrong type filter excludes the doc.
		spec := models.TypeSpec
		if filtered, _ := st.SearchDocuments(ctx, "Kubernetes", store.Filter{Type: &spec}); len(filtered) != 0 {
			t.Errorf("SearchDocuments type filter: got %d", len(filtered))
		}

		dec := &models.Decision{Title: "Use gRPC", Decision: "adopt grpc transport", Status: models.DecisionAccepted}
		if err := st.CreateDecision(ctx, dec); err != nil {
			t.Fatalf("create decision: %v", err)
		}
		if decs, _ := st.SearchDecisions(ctx, "gRPC", store.DecisionFilter{}); len(decs) != 1 {
			t.Errorf("SearchDecisions(gRPC): got %d", len(decs))
		}

		sn := &models.Snippet{Title: "debounce helper", Content: "setTimeout wrapper", Language: "js"}
		if err := st.CreateSnippet(ctx, sn); err != nil {
			t.Fatalf("create snippet: %v", err)
		}
		if sns, _ := st.SearchSnippets(ctx, "debounce", store.SnippetFilter{}); len(sns) != 1 {
			t.Errorf("SearchSnippets(debounce): got %d", len(sns))
		}

		sol := &models.Solution{ErrorDescription: "CORS request blocked", Solution: "add Access-Control header"}
		if err := st.CreateSolution(ctx, sol); err != nil {
			t.Fatalf("create solution: %v", err)
		}
		if sols, _ := st.SearchSolutions(ctx, "CORS", store.SolutionFilter{}); len(sols) != 1 {
			t.Errorf("SearchSolutions(CORS): got %d", len(sols))
		}

		// Degenerate queries never error.
		for _, q := range []string{"", "-", `"`} {
			if _, err := st.SearchDocuments(ctx, q, store.Filter{}); err != nil {
				t.Errorf("SearchDocuments(%q) errored: %v", q, err)
			}
		}
	})
}

func TestConformanceVectorFloor(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")
		st.SetSearchMaxDist(0.5) // drop vector hits with cosine distance >= 0.5

		d := sampleDoc("vec-1", "Vector doc")
		d.Project = ptr("apollo")
		if err := st.CreateDocument(ctx, d); err != nil {
			t.Fatalf("create doc: %v", err)
		}
		base := unitVec(1024, 0)
		if err := st.UpsertEmbeddings(ctx, []models.Embedding{{
			SourceType: "documents", SourceID: "vec-1", ChunkID: "c1",
			ChunkText: "the vector doc chunk", Embedding: base, Project: ptr("apollo"),
			SourceTitle: "Vector doc", Model: "voyage-4-large",
		}}); err != nil {
			t.Fatalf("UpsertEmbeddings: %v", err)
		}

		// An FTS term that matches nothing isolates the vector channel.
		const noMatch = "zzunmatchable"

		// Query vector aligned with the stored one → similarity ~1, passes floor.
		near, err := st.Search(ctx, noMatch, store.SearchFilter{Vector: base})
		if err != nil {
			t.Fatalf("Search(near): %v", err)
		}
		if !hasHit(near, "documents", "vec-1") {
			t.Fatalf("aligned vector should surface vec-1: %+v", near)
		}
		for _, h := range near {
			if h.ID == "vec-1" && (h.Similarity == nil || *h.Similarity < 0.9) {
				t.Errorf("vec-1 similarity: got %v, want ~1", h.Similarity)
			}
		}

		// Orthogonal query vector → similarity ~0, distance ~1 ≥ floor → dropped.
		orth := unitVec(1024, 1)
		far, _ := st.Search(ctx, noMatch, store.SearchFilter{Vector: orth})
		if hasHit(far, "documents", "vec-1") {
			t.Errorf("orthogonal vector should be dropped by the floor: %+v", far)
		}
	})
}

func TestConformanceEmbeddings(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()
		seedProjectsIfc(t, st, "apollo")
		doc := sampleDoc("doc-emb", "Embeddable")
		doc.Project = ptr("apollo")
		if err := st.CreateDocument(ctx, doc); err != nil {
			t.Fatalf("CreateDocument: %v", err)
		}

		mk := func(chunk, text string, fill float32) models.Embedding {
			return models.Embedding{
				SourceType: "documents", SourceID: "doc-emb", ChunkID: chunk,
				ChunkText: text, Embedding: vecOf(1024, fill), Project: ptr("apollo"),
				SourceTitle: "Embeddable", Model: "voyage-4-large",
			}
		}

		if err := st.UpsertEmbeddings(ctx, []models.Embedding{mk("c1", "first chunk", 0.1), mk("c2", "second chunk", 0.2)}); err != nil {
			t.Fatalf("UpsertEmbeddings: %v", err)
		}
		got, err := st.EmbeddingsFor(ctx, "documents", "doc-emb")
		if err != nil {
			t.Fatalf("EmbeddingsFor: %v", err)
		}
		if len(got) != 2 || got["c1"] != "first chunk" || got["c2"] != "second chunk" {
			t.Fatalf("EmbeddingsFor: %v", got)
		}

		// Upsert conflict updates the existing chunk (still two rows).
		if err := st.UpsertEmbeddings(ctx, []models.Embedding{mk("c1", "updated first", 0.15)}); err != nil {
			t.Fatalf("UpsertEmbeddings(update): %v", err)
		}
		got, _ = st.EmbeddingsFor(ctx, "documents", "doc-emb")
		if len(got) != 2 || got["c1"] != "updated first" {
			t.Fatalf("after conflict upsert: %v", got)
		}

		// DeleteEmbeddingsExcept keeps only c1.
		if err := st.DeleteEmbeddingsExcept(ctx, "documents", "doc-emb", []string{"c1"}); err != nil {
			t.Fatalf("DeleteEmbeddingsExcept: %v", err)
		}
		got, _ = st.EmbeddingsFor(ctx, "documents", "doc-emb")
		if len(got) != 1 || got["c1"] == "" {
			t.Fatalf("after prune: %v", got)
		}

		// SourceRefs enumerates live sources (includes our document).
		refs, err := st.SourceRefs(ctx)
		if err != nil {
			t.Fatalf("SourceRefs: %v", err)
		}
		found := false
		for _, r := range refs {
			if r.Type == "documents" && r.ID == "doc-emb" {
				found = true
			}
		}
		if !found {
			t.Errorf("SourceRefs missing documents/doc-emb: %v", refs)
		}

		// HasStaleModelEmbeddings: matches current model → false; other → true.
		if stale, _ := st.HasStaleModelEmbeddings(ctx, "documents", "doc-emb", "voyage-4-large"); stale {
			t.Errorf("HasStaleModel(current): got true")
		}
		if stale, _ := st.HasStaleModelEmbeddings(ctx, "documents", "doc-emb", "other-model"); !stale {
			t.Errorf("HasStaleModel(other): got false")
		}

		// EmbeddingStatus documents bucket: 1 live source, embedded, reconciled.
		status, err := st.EmbeddingStatus(ctx, "voyage-4-large")
		if err != nil {
			t.Fatalf("EmbeddingStatus: %v", err)
		}
		var docs *store.TypeStatus
		for i := range status {
			if status[i].Type == "documents" {
				docs = &status[i]
			}
		}
		if docs == nil || docs.Total != 1 || docs.Embedded != 1 || docs.Reconciled != 1 || docs.Missing != 0 {
			t.Errorf("documents status: %+v", docs)
		}

		// DeleteOrphanEmbeddings removes vectors whose source no longer exists.
		if err := st.UpsertEmbeddings(ctx, []models.Embedding{{
			SourceType: "documents", SourceID: "ghost-doc", ChunkID: "g1",
			ChunkText: "orphan", Embedding: vecOf(1024, 0.9), SourceTitle: "ghost", Model: "voyage-4-large",
		}}); err != nil {
			t.Fatalf("UpsertEmbeddings(orphan): %v", err)
		}
		n, err := st.DeleteOrphanEmbeddings(ctx)
		if err != nil {
			t.Fatalf("DeleteOrphanEmbeddings: %v", err)
		}
		if n != 1 {
			t.Errorf("DeleteOrphanEmbeddings: removed %d, want 1", n)
		}

		// Failure tracking: record, list, clear.
		if err := st.RecordEmbedFailure(ctx, "documents", "doc-emb", "boom"); err != nil {
			t.Fatalf("RecordEmbedFailure: %v", err)
		}
		if err := st.RecordEmbedFailure(ctx, "documents", "doc-emb", "boom again"); err != nil {
			t.Fatalf("RecordEmbedFailure(2): %v", err)
		}
		failed, err := st.FailedSourceRefs(ctx)
		if err != nil {
			t.Fatalf("FailedSourceRefs: %v", err)
		}
		if len(failed) != 1 || failed[0].ID != "doc-emb" {
			t.Errorf("FailedSourceRefs: %v", failed)
		}
		if err := st.ClearEmbedFailure(ctx, "documents", "doc-emb"); err != nil {
			t.Fatalf("ClearEmbedFailure: %v", err)
		}
		if failed, _ := st.FailedSourceRefs(ctx); len(failed) != 0 {
			t.Errorf("after clear: %v", failed)
		}
	})
}

func TestConformanceProjects(t *testing.T) {
	forEachBackend(t, func(t *testing.T, st store.Store) {
		ctx := context.Background()

		p := &models.Project{Name: "Apollo", Slug: "apollo", Description: ptr("the moon program")}
		if err := st.CreateProject(ctx, p); err != nil {
			t.Fatalf("CreateProject: %v", err)
		}
		if p.ID == "" || !strings.HasPrefix(p.PublicID, "prj_") || p.CreatedAt.IsZero() {
			t.Errorf("CreateProject did not populate id/public_id/created_at: %+v", p)
		}

		got, err := st.GetProject(ctx, "apollo")
		if err != nil {
			t.Fatalf("GetProject: %v", err)
		}
		if got.Name != "Apollo" || got.Description == nil || *got.Description != "the moon program" {
			t.Errorf("GetProject roundtrip: %+v", got)
		}

		// Duplicate slug → ErrDuplicateProject.
		if err := st.CreateProject(ctx, &models.Project{Name: "Apollo2", Slug: "apollo"}); !errors.Is(err, store.ErrDuplicateProject) {
			t.Errorf("CreateProject(dup slug): got %v, want ErrDuplicateProject", err)
		}

		// Missing → ErrNotFound.
		if _, err := st.GetProject(ctx, "ghost"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetProject(missing): got %v, want ErrNotFound", err)
		}

		// ListProjects reports per-status counts.
		d := sampleDoc("pd-1", "Doc")
		d.Project = ptr("apollo")
		d.Status = models.StatusInProgress
		if err := st.CreateDocument(ctx, d); err != nil {
			t.Fatalf("create doc: %v", err)
		}
		stats, err := st.ListProjects(ctx)
		if err != nil {
			t.Fatalf("ListProjects: %v", err)
		}
		if len(stats) != 1 {
			t.Fatalf("ListProjects: got %d, want 1", len(stats))
		}
		if stats[0].Counts.Total != 1 || stats[0].Counts.InProgress != 1 {
			t.Errorf("counts: %+v", stats[0].Counts)
		}
	})
}
