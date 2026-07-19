package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jeroenpfeil/mneme/internal/migrations"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

var testDSN string

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("mneme"),
		tcpostgres.WithUsername("mneme"),
		tcpostgres.WithPassword("mneme"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = testcontainers.TerminateContainer(container) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection string: %v\n", err)
		os.Exit(1)
	}
	testDSN = dsn

	if err := migrations.Up(testDSN); err != nil {
		fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// newStore returns a fresh-state store. It TRUNCATEs documents and
// projects so each test starts from a clean slate.
func newStore(t *testing.T) *store.PostgresStore {
	t.Helper()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx,
		`TRUNCATE documents, projects, embeddings, embed_failures RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return store.NewWithPool(pool)
}

func ptr[T any](v T) *T { return &v }

func ptrScope(s models.MemoryScope) *models.MemoryScope { return &s }

// seedProjects inserts the given slugs into the projects table.
// documents.project has an FK to projects.slug, so any doc that sets
// Project must have its slug seeded first.
func seedProjects(t *testing.T, s *store.PostgresStore, slugs ...string) {
	t.Helper()
	ctx := context.Background()
	for _, slug := range slugs {
		_, err := s.Pool().Exec(ctx,
			`INSERT INTO projects (name, slug) VALUES ($1, $1)`, slug)
		if err != nil {
			t.Fatalf("seed project %q: %v", slug, err)
		}
	}
}

func sampleDoc(id, title string) *models.Document {
	return &models.Document{
		ID:     id,
		Title:  title,
		Type:   models.TypePlan,
		Status: models.StatusTodo,
		Tags:   []string{"go", "postgres"},
		Meta:   map[string]any{"phases": []any{map[string]any{"title": "Foundation", "status": "wip"}}},
		Body: map[string]any{
			"sections": []any{
				map[string]any{
					"type":  "section",
					"id":    "overview",
					"title": "Overview",
				},
			},
		},
	}
}

func TestCreateAndGetDocument(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	seedProjects(t, s, "apollo")
	doc := sampleDoc("doc-001", "Vehicle Listing API")
	doc.Project = ptr("apollo")
	doc.Ticket = ptr("C1-142")

	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if doc.CreatedAt.IsZero() {
		t.Fatal("CreatedAt not populated by RETURNING")
	}
	if doc.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt not populated by RETURNING")
	}

	got, err := s.GetDocument(ctx, "doc-001")
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Title != "Vehicle Listing API" {
		t.Errorf("title: got %q, want %q", got.Title, "Vehicle Listing API")
	}
	if got.Project == nil || *got.Project != "apollo" {
		t.Errorf("project: got %v, want apollo", got.Project)
	}
	if got.Ticket == nil || *got.Ticket != "C1-142" {
		t.Errorf("ticket: got %v, want C1-142", got.Ticket)
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
}

func TestCreateDocumentRejectsUnknownProject(t *testing.T) {
	s := newStore(t)
	doc := sampleDoc("orphan", "Orphan")
	doc.Project = ptr("does-not-exist")
	err := s.CreateDocument(context.Background(), doc)
	if !errors.Is(err, store.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject, got: %v", err)
	}
}

func TestCreateDocumentRejectsDuplicateID(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.CreateDocument(ctx, sampleDoc("dup", "First")); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := s.CreateDocument(ctx, sampleDoc("dup", "Second"))
	if !errors.Is(err, store.ErrDuplicateID) {
		t.Fatalf("expected ErrDuplicateID, got: %v", err)
	}
}

func TestCreateProject(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	p := &models.Project{Name: "TradeGod", Slug: "tradegod", Description: ptr("Trading bot")}
	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ID == "" {
		t.Error("ID not populated by RETURNING")
	}
	if p.CreatedAt.IsZero() {
		t.Error("CreatedAt not populated by RETURNING")
	}

	got, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "tradegod" || got[0].Name != "TradeGod" {
		t.Fatalf("expected [tradegod/TradeGod], got %+v", got)
	}
	if got[0].Description == nil || *got[0].Description != "Trading bot" {
		t.Errorf("description: got %v, want 'Trading bot'", got[0].Description)
	}

	// The whole point: a document may now reference the new project.
	doc := sampleDoc("d1", "First")
	doc.Project = ptr("tradegod")
	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument referencing new project: %v", err)
	}
}

func TestCreateProjectNullDescription(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	p := &models.Project{Name: "hermes", Slug: "hermes"}
	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	got, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(got) != 1 || got[0].Description != nil {
		t.Fatalf("expected one project with nil description, got %+v", got)
	}
}

func TestCreateProjectRejectsDuplicateSlug(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.CreateProject(ctx, &models.Project{Name: "A", Slug: "dup"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := s.CreateProject(ctx, &models.Project{Name: "B", Slug: "dup"})
	if !errors.Is(err, store.ErrDuplicateProject) {
		t.Fatalf("expected ErrDuplicateProject, got: %v", err)
	}
}

func TestListProjectsCounts(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	seedProjects(t, s, "apollo", "hermes")

	mkdoc := func(id, project, status string) *models.Document {
		d := sampleDoc(id, "T "+id)
		d.Project = ptr(project)
		d.Status = status
		return d
	}
	for _, d := range []*models.Document{
		mkdoc("a1", "apollo", models.StatusTodo),
		mkdoc("a2", "apollo", models.StatusInProgress),
		mkdoc("a3", "apollo", models.StatusComplete),
		mkdoc("a4", "apollo", models.StatusArchived),
		mkdoc("h1", "hermes", models.StatusBlocked),
	} {
		if err := s.CreateDocument(ctx, d); err != nil {
			t.Fatalf("CreateDocument %s: %v", d.ID, err)
		}
	}

	got, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d projects, want 2", len(got))
	}
	byslug := map[string]*models.ProjectStats{}
	for _, p := range got {
		byslug[p.Slug] = p
	}
	apollo := byslug["apollo"]
	if apollo.Counts.Total != 4 ||
		apollo.Counts.Todo != 1 ||
		apollo.Counts.InProgress != 1 ||
		apollo.Counts.Complete != 1 ||
		apollo.Counts.Archived != 1 {
		t.Errorf("apollo counts wrong: %+v", apollo.Counts)
	}
	hermes := byslug["hermes"]
	if hermes.Counts.Total != 1 || hermes.Counts.Blocked != 1 {
		t.Errorf("hermes counts wrong: %+v", hermes.Counts)
	}
}

func TestGetDocumentNotFound(t *testing.T) {
	s := newStore(t)
	if _, err := s.GetDocument(context.Background(), "nope"); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateDocument(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	doc := sampleDoc("doc-update", "Original")
	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	originalUpdatedAt := doc.UpdatedAt

	// Trigger granularity is microsecond — sleep enough to guarantee a
	// distinguishable timestamp on platforms with coarse clocks.
	time.Sleep(10 * time.Millisecond)

	doc.Title = "Renamed"
	doc.Status = models.StatusInProgress
	doc.Tags = []string{"go"}
	doc.Body = map[string]any{"sections": []any{map[string]any{"id": "new"}}}
	if err := s.UpdateDocument(ctx, doc); err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}
	if !doc.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("updated_at trigger did not bump: before=%v after=%v",
			originalUpdatedAt, doc.UpdatedAt)
	}

	got, err := s.GetDocument(ctx, "doc-update")
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Title != "Renamed" || got.Status != models.StatusInProgress {
		t.Errorf("update not persisted: %+v", got)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "go" {
		t.Errorf("tags not updated: %v", got.Tags)
	}
}

func TestUpdateDocumentNotFound(t *testing.T) {
	s := newStore(t)
	doc := sampleDoc("ghost", "Ghost")
	if err := s.UpdateDocument(context.Background(), doc); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestArchiveDocument(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	doc := sampleDoc("doc-archive", "Will be archived")
	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if err := s.ArchiveDocument(ctx, "doc-archive"); err != nil {
		t.Fatalf("ArchiveDocument: %v", err)
	}
	got, err := s.GetDocument(ctx, "doc-archive")
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.Status != models.StatusArchived {
		t.Errorf("status: got %q, want %q", got.Status, models.StatusArchived)
	}

	if err := s.ArchiveDocument(ctx, "nope"); err != store.ErrNotFound {
		t.Errorf("missing doc: expected ErrNotFound, got %v", err)
	}
}

func TestListDocumentsFilters(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	mkdoc := func(id, project, docType string, tags []string) *models.Document {
		d := sampleDoc(id, "Title "+id)
		d.Type = docType
		d.Project = ptr(project)
		d.Tags = tags
		return d
	}
	seedProjects(t, s, "apollo", "hermes")
	docs := []*models.Document{
		mkdoc("a", "apollo", models.TypePlan, []string{"go", "api"}),
		mkdoc("b", "apollo", models.TypeSpec, []string{"go"}),
		mkdoc("c", "hermes", models.TypePlan, []string{"vue"}),
	}
	for _, d := range docs {
		if err := s.CreateDocument(ctx, d); err != nil {
			t.Fatalf("CreateDocument %s: %v", d.ID, err)
		}
	}

	// Filter by project.
	got, err := s.ListDocuments(ctx, store.Filter{Project: ptr("apollo")})
	if err != nil {
		t.Fatalf("ListDocuments by project: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("by project: got %d, want 2", len(got))
	}

	// Filter by type.
	got, err = s.ListDocuments(ctx, store.Filter{Type: ptr(models.TypePlan)})
	if err != nil {
		t.Fatalf("ListDocuments by type: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("by type: got %d, want 2", len(got))
	}

	// Filter by tags (must contain ALL given tags).
	got, err = s.ListDocuments(ctx, store.Filter{Tags: []string{"go", "api"}})
	if err != nil {
		t.Fatalf("ListDocuments by tags: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Errorf("by tags: got %d items, want [a]", len(got))
	}

	// Combined filter.
	got, err = s.ListDocuments(ctx, store.Filter{
		Project: ptr("apollo"),
		Type:    ptr(models.TypePlan),
	})
	if err != nil {
		t.Fatalf("combined filter: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Errorf("combined: got %d items, want [a]", len(got))
	}

	// Limit.
	got, err = s.ListDocuments(ctx, store.Filter{Limit: 2})
	if err != nil {
		t.Fatalf("limit: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("limit: got %d, want 2", len(got))
	}
}

func TestSearchDocumentsRanksAndFilters(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	seedProjects(t, s, "apollo", "hermes")
	docs := []*models.Document{
		{
			// Title-only match — should rank highest (weight A).
			ID: "title-hit", Title: "Vehicle Listing API",
			Type: models.TypePlan, Status: models.StatusTodo,
			Tags: []string{"api"}, Project: ptr("apollo"),
		},
		{
			// Body-only match — should rank below the title hit (weight C).
			ID: "body-hit", Title: "Inventory Spec",
			Type: models.TypeSpec, Status: models.StatusTodo,
			Tags: []string{"api"}, Project: ptr("apollo"),
			Body: map[string]any{
				"sections": []any{map[string]any{
					"id": "intro", "content": "describes the vehicle endpoint",
				}},
			},
		},
		{
			ID: "unrelated", Title: "Pricing Engine",
			Type: models.TypePlan, Status: models.StatusTodo,
			Tags: []string{"billing"}, Project: ptr("hermes"),
		},
	}
	for _, d := range docs {
		if err := s.CreateDocument(ctx, d); err != nil {
			t.Fatalf("CreateDocument %s: %v", d.ID, err)
		}
	}

	got, err := s.SearchDocuments(ctx, "vehicle", store.Filter{})
	if err != nil {
		t.Fatalf("SearchDocuments: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("search 'vehicle': got %d results, want 2", len(got))
	}
	// Title weight (A) beats body weight (C), so the title-only match
	// should outrank the body-only match.
	if got[0].ID != "title-hit" {
		t.Errorf("ranking: got %q first, want 'title-hit' (title beats body)", got[0].ID)
	}

	// FTS + filter composition.
	got, err = s.SearchDocuments(ctx, "vehicle", store.Filter{
		Type: ptr(models.TypeSpec),
	})
	if err != nil {
		t.Fatalf("SearchDocuments with filter: %v", err)
	}
	if len(got) != 1 || got[0].ID != "body-hit" {
		t.Errorf("filtered search: got %d items, want [body-hit]", len(got))
	}

	// No matches.
	got, err = s.SearchDocuments(ctx, "thisreallyshouldnotmatchanything", store.Filter{})
	if err != nil {
		t.Fatalf("empty search: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}

	// websearch syntax: OR pulls in 'unrelated' alongside 'title-hit'.
	got, err = s.SearchDocuments(ctx, "vehicle OR pricing", store.Filter{})
	if err != nil {
		t.Fatalf("OR search: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("OR search: got %d matches, want 3 (title-hit, body-hit, unrelated)", len(got))
	}

	// websearch syntax: -term excludes.
	got, err = s.SearchDocuments(ctx, "vehicle -inventory", store.Filter{})
	if err != nil {
		t.Fatalf("exclusion search: %v", err)
	}
	if len(got) != 1 || got[0].ID != "title-hit" {
		t.Errorf("exclusion search: got %+v, want [title-hit]", docIDs(got))
	}
}

func docIDs(docs []*models.Document) []string {
	out := make([]string, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.ID)
	}
	return out
}

func TestSetMemoryGlobalUpsert(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	m := &models.Memory{Scope: models.ScopeGlobal, Key: "editor", Value: "vscode"}
	if err := s.SetMemory(ctx, m); err != nil {
		t.Fatalf("SetMemory: %v", err)
	}
	if m.ID == "" || m.UpdatedAt.IsZero() {
		t.Errorf("id/updated_at not populated: %+v", m)
	}

	// Re-set same (scope,project,area,key) — must UPDATE, not duplicate.
	m2 := &models.Memory{Scope: models.ScopeGlobal, Key: "editor", Value: "neovim"}
	if err := s.SetMemory(ctx, m2); err != nil {
		t.Fatalf("SetMemory upsert: %v", err)
	}
	got, err := s.ListMemory(ctx, store.MemoryFilter{Scope: ptrScope(models.ScopeGlobal)})
	if err != nil {
		t.Fatalf("ListMemory: %v", err)
	}
	if len(got) != 1 || got[0].Value != "neovim" {
		t.Fatalf("expected single upserted row 'neovim', got %+v", got)
	}
}

func TestSetMemoryProjectScopeAndFilter(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")

	if err := s.SetMemory(ctx, &models.Memory{Scope: models.ScopeGlobal, Key: "k", Value: "g"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMemory(ctx, &models.Memory{Scope: models.ScopeProject, Project: ptr("apollo"), Key: "k", Value: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMemory(ctx, &models.Memory{Scope: models.ScopeArea, Project: ptr("apollo"), Area: ptr("billing"), Key: "k", Value: "a"}); err != nil {
		t.Fatal(err)
	}

	proj, err := s.ListMemory(ctx, store.MemoryFilter{Scope: ptrScope(models.ScopeProject), Project: ptr("apollo")})
	if err != nil {
		t.Fatal(err)
	}
	if len(proj) != 1 || proj[0].Value != "p" {
		t.Errorf("project filter: got %+v", proj)
	}
	all, err := s.ListMemory(ctx, store.MemoryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("unfiltered: got %d rows, want 3", len(all))
	}
}

func TestSetMemoryUnknownProject(t *testing.T) {
	s := newStore(t)
	err := s.SetMemory(context.Background(),
		&models.Memory{Scope: models.ScopeProject, Project: ptr("ghost"), Key: "k", Value: "v"})
	if !errors.Is(err, store.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject, got %v", err)
	}
}

func TestDeleteMemory(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.SetMemory(ctx, &models.Memory{Scope: models.ScopeGlobal, Key: "k", Value: "v"}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteMemory(ctx, models.ScopeGlobal, nil, nil, "k"); err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}
	if err := s.DeleteMemory(ctx, models.ScopeGlobal, nil, nil, "k"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second delete: expected ErrNotFound, got %v", err)
	}
}

func TestSetEnvUpsertAndList(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")

	e := &models.EnvEntry{Project: "apollo", Key: "API_PORT", Value: "8443", Description: ptr("https port")}
	if err := s.SetEnv(ctx, e); err != nil {
		t.Fatalf("SetEnv: %v", err)
	}
	if e.ID == "" || e.UpdatedAt.IsZero() {
		t.Errorf("id/updated_at not populated: %+v", e)
	}

	// Re-set same (project,key) — must UPDATE value + description, not duplicate.
	e2 := &models.EnvEntry{Project: "apollo", Key: "API_PORT", Value: "9000", Description: ptr("moved")}
	if err := s.SetEnv(ctx, e2); err != nil {
		t.Fatalf("SetEnv upsert: %v", err)
	}
	got, err := s.ListEnv(ctx, "apollo")
	if err != nil {
		t.Fatalf("ListEnv: %v", err)
	}
	if len(got) != 1 || got[0].Value != "9000" || got[0].Description == nil || *got[0].Description != "moved" {
		t.Fatalf("expected single upserted row 9000/moved, got %+v", got)
	}
}

func TestSetEnvNullDescriptionAndOrder(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.SetEnv(ctx, &models.EnvEntry{Project: "apollo", Key: "ZED", Value: "z"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetEnv(ctx, &models.EnvEntry{Project: "apollo", Key: "ABLE", Value: "a"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListEnv(ctx, "apollo")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Key != "ABLE" || got[1].Key != "ZED" {
		t.Fatalf("expected key-ordered [ABLE ZED], got %+v", got)
	}
	if got[0].Description != nil {
		t.Errorf("expected NULL description to round-trip as nil, got %v", *got[0].Description)
	}
}

func TestSetEnvUnknownProject(t *testing.T) {
	s := newStore(t)
	err := s.SetEnv(context.Background(), &models.EnvEntry{Project: "ghost", Key: "k", Value: "v"})
	if !errors.Is(err, store.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject, got %v", err)
	}
}

func TestDeleteEnv(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.SetEnv(ctx, &models.EnvEntry{Project: "apollo", Key: "k", Value: "v"}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteEnv(ctx, "apollo", "k"); err != nil {
		t.Fatalf("DeleteEnv: %v", err)
	}
	if err := s.DeleteEnv(ctx, "apollo", "k"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second delete: expected ErrNotFound, got %v", err)
	}
}

func TestCreateDecisionAndList(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")

	d := &models.Decision{
		Title:     "Use pgx over database/sql",
		Project:   ptr("apollo"),
		Decision:  "Adopt jackc/pgx v5 directly.",
		Rationale: "Native Postgres types and better performance.",
		Status:    models.DecisionAccepted,
	}
	if err := s.CreateDecision(ctx, d); err != nil {
		t.Fatalf("CreateDecision: %v", err)
	}
	if d.ID == "" || d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		t.Errorf("id/timestamps not populated: %+v", d)
	}

	got, err := s.ListDecisions(ctx, store.DecisionFilter{Project: ptr("apollo")})
	if err != nil {
		t.Fatalf("ListDecisions: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Use pgx over database/sql" {
		t.Fatalf("expected 1 decision, got %+v", got)
	}
}

func TestCreateDecisionGlobal(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	d := &models.Decision{Title: "Raw SQL only", Decision: "No ORMs.", Status: models.DecisionAccepted}
	if err := s.CreateDecision(ctx, d); err != nil {
		t.Fatalf("CreateDecision global: %v", err)
	}
	got, err := s.ListDecisions(ctx, store.DecisionFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Project != nil {
		t.Fatalf("expected 1 global decision (nil project), got %+v", got)
	}
}

func TestCreateDecisionUnknownProject(t *testing.T) {
	s := newStore(t)
	err := s.CreateDecision(context.Background(),
		&models.Decision{Title: "t", Project: ptr("ghost"), Decision: "d", Status: models.DecisionAccepted})
	if !errors.Is(err, store.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject, got %v", err)
	}
}

func TestUpdateDecision(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	d := &models.Decision{Title: "t", Decision: "d", Status: models.DecisionProposed}
	if err := s.CreateDecision(ctx, d); err != nil {
		t.Fatal(err)
	}

	d.Status = models.DecisionAccepted
	d.Rationale = "ratified in review"
	if err := s.UpdateDecision(ctx, d); err != nil {
		t.Fatalf("UpdateDecision: %v", err)
	}

	got, err := s.GetDecision(ctx, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.DecisionAccepted || got.Rationale != "ratified in review" {
		t.Fatalf("update not persisted: %+v", got)
	}
}

func TestUpdateDecisionNotFound(t *testing.T) {
	s := newStore(t)
	err := s.UpdateDecision(context.Background(),
		&models.Decision{ID: "00000000-0000-0000-0000-000000000000", Title: "t", Decision: "d", Status: models.DecisionAccepted})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListDecisionsFilterAndOrder(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	// older proposed, then newer accepted — expect newest-first ordering.
	if err := s.CreateDecision(ctx, &models.Decision{Title: "first", Project: ptr("apollo"), Decision: "d", Status: models.DecisionProposed}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDecision(ctx, &models.Decision{Title: "second", Project: ptr("apollo"), Decision: "d", Status: models.DecisionAccepted}); err != nil {
		t.Fatal(err)
	}

	all, err := s.ListDecisions(ctx, store.DecisionFilter{Project: ptr("apollo")})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Title != "second" {
		t.Fatalf("expected newest-first [second, first], got %+v", all)
	}

	accepted := models.DecisionAccepted
	only, err := s.ListDecisions(ctx, store.DecisionFilter{Status: &accepted})
	if err != nil {
		t.Fatal(err)
	}
	if len(only) != 1 || only[0].Title != "second" {
		t.Fatalf("status filter: got %+v", only)
	}
}

func TestSearchDecisions(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.CreateDecision(ctx, &models.Decision{
		Title: "Choose Sanctum for API auth", Decision: "Use Laravel Sanctum.",
		Rationale: "Token auth without the OAuth ceremony of Passport.", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDecision(ctx, &models.Decision{
		Title: "Cursor pagination", Decision: "Keyset pagination on id.",
		Rationale: "Stable pages under concurrent inserts.", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}

	hits, err := s.SearchDecisions(ctx, "sanctum auth", store.DecisionFilter{})
	if err != nil {
		t.Fatalf("SearchDecisions: %v", err)
	}
	if len(hits) == 0 || hits[0].Title != "Choose Sanctum for API auth" {
		t.Fatalf("expected Sanctum decision ranked first, got %+v", hits)
	}
}

func TestCreateSnippetAndList(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")

	sn := &models.Snippet{
		Title:       "Cursor pagination",
		Project:     ptr("apollo"),
		Language:    "typescript",
		Content:     "const after = cursor ? { id: { gt: cursor } } : {}",
		Tags:        []string{"pagination", "apollo"},
		Description: "Keyset pagination helper.",
	}
	if err := s.CreateSnippet(ctx, sn); err != nil {
		t.Fatalf("CreateSnippet: %v", err)
	}
	if sn.ID == "" || sn.CreatedAt.IsZero() || sn.UpdatedAt.IsZero() {
		t.Errorf("id/timestamps not populated: %+v", sn)
	}

	got, err := s.ListSnippets(ctx, store.SnippetFilter{Project: ptr("apollo")})
	if err != nil {
		t.Fatalf("ListSnippets: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Cursor pagination" {
		t.Fatalf("expected 1 snippet, got %+v", got)
	}
	if len(got[0].Tags) != 2 {
		t.Errorf("tags not round-tripped: %+v", got[0].Tags)
	}
}

func TestCreateSnippetGlobal(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sn := &models.Snippet{Title: "errgroup fan-out", Language: "go", Content: "g, ctx := errgroup.WithContext(ctx)"}
	if err := s.CreateSnippet(ctx, sn); err != nil {
		t.Fatalf("CreateSnippet global: %v", err)
	}
	got, err := s.ListSnippets(ctx, store.SnippetFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Project != nil {
		t.Fatalf("expected 1 global snippet (nil project), got %+v", got)
	}
	if len(got[0].Tags) != 0 {
		t.Errorf("expected empty tags, got %+v", got[0].Tags)
	}
}

func TestCreateSnippetUnknownProject(t *testing.T) {
	s := newStore(t)
	err := s.CreateSnippet(context.Background(),
		&models.Snippet{Title: "t", Project: ptr("ghost"), Content: "c"})
	if !errors.Is(err, store.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject, got %v", err)
	}
}

func TestUpdateSnippet(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sn := &models.Snippet{Title: "t", Language: "go", Content: "old"}
	if err := s.CreateSnippet(ctx, sn); err != nil {
		t.Fatal(err)
	}

	sn.Content = "new content"
	sn.Tags = []string{"updated"}
	if err := s.UpdateSnippet(ctx, sn); err != nil {
		t.Fatalf("UpdateSnippet: %v", err)
	}

	got, err := s.GetSnippet(ctx, sn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "new content" || len(got.Tags) != 1 || got.Tags[0] != "updated" {
		t.Fatalf("update not persisted: %+v", got)
	}
}

func TestUpdateSnippetNotFound(t *testing.T) {
	s := newStore(t)
	err := s.UpdateSnippet(context.Background(),
		&models.Snippet{ID: "00000000-0000-0000-0000-000000000000", Title: "t", Content: "c"})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListSnippetsFilterAndOrder(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	// older 'go' first, then newer 'sql' — expect newest-first ordering.
	if err := s.CreateSnippet(ctx, &models.Snippet{Title: "first", Project: ptr("apollo"), Language: "go", Content: "c", Tags: []string{"http"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSnippet(ctx, &models.Snippet{Title: "second", Project: ptr("apollo"), Language: "sql", Content: "c", Tags: []string{"query"}}); err != nil {
		t.Fatal(err)
	}

	all, err := s.ListSnippets(ctx, store.SnippetFilter{Project: ptr("apollo")})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Title != "second" {
		t.Fatalf("expected newest-first [second, first], got %+v", all)
	}

	lang := "go"
	byLang, err := s.ListSnippets(ctx, store.SnippetFilter{Language: &lang})
	if err != nil {
		t.Fatal(err)
	}
	if len(byLang) != 1 || byLang[0].Title != "first" {
		t.Fatalf("language filter: got %+v", byLang)
	}

	byTag, err := s.ListSnippets(ctx, store.SnippetFilter{Tags: []string{"query"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(byTag) != 1 || byTag[0].Title != "second" {
		t.Fatalf("tag filter: got %+v", byTag)
	}
}

func TestSearchSnippets(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.CreateSnippet(ctx, &models.Snippet{
		Title: "Cursor pagination", Language: "typescript", Content: "keyset on id",
		Description: "Stable pages under concurrent inserts.",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSnippet(ctx, &models.Snippet{
		Title: "Errgroup fan-out", Language: "go", Content: "g, ctx := errgroup.WithContext(ctx)",
		Description: "Bounded concurrency for parallel calls.",
	}); err != nil {
		t.Fatal(err)
	}

	hits, err := s.SearchSnippets(ctx, "cursor pagination", store.SnippetFilter{})
	if err != nil {
		t.Fatalf("SearchSnippets: %v", err)
	}
	if len(hits) == 0 || hits[0].Title != "Cursor pagination" {
		t.Fatalf("expected pagination snippet ranked first, got %+v", hits)
	}
}

func TestCreateJournalEntryAndList(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")

	e := &models.JournalEntry{
		Project:      ptr("apollo"),
		SessionRef:   "sp-3-1",
		Summary:      "Wired the pagination endpoint",
		Accomplished: []string{"cursor pagination", "integration tests"},
		Deferred:     []string{"rate limiting"},
	}
	if err := s.CreateJournalEntry(ctx, e); err != nil {
		t.Fatalf("CreateJournalEntry: %v", err)
	}
	if e.ID == "" || e.CreatedAt.IsZero() || e.UpdatedAt.IsZero() {
		t.Errorf("id/timestamps not populated: %+v", e)
	}

	got, err := s.ListJournalEntries(ctx, store.JournalFilter{Project: ptr("apollo")})
	if err != nil {
		t.Fatalf("ListJournalEntries: %v", err)
	}
	if len(got) != 1 || got[0].Summary != "Wired the pagination endpoint" {
		t.Fatalf("expected 1 entry, got %+v", got)
	}
	if len(got[0].Accomplished) != 2 || len(got[0].Deferred) != 1 {
		t.Errorf("arrays not round-tripped: %+v", got[0])
	}
}

func TestCreateJournalEntryGlobal(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	e := &models.JournalEntry{Summary: "Refactored the migration runner"}
	if err := s.CreateJournalEntry(ctx, e); err != nil {
		t.Fatalf("CreateJournalEntry global: %v", err)
	}
	got, err := s.ListJournalEntries(ctx, store.JournalFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Project != nil {
		t.Fatalf("expected 1 global entry (nil project), got %+v", got)
	}
	if len(got[0].Accomplished) != 0 || len(got[0].Deferred) != 0 {
		t.Errorf("expected empty arrays, got %+v", got[0])
	}
}

func TestCreateJournalEntryUnknownProject(t *testing.T) {
	s := newStore(t)
	err := s.CreateJournalEntry(context.Background(),
		&models.JournalEntry{Project: ptr("ghost"), Summary: "s"})
	if !errors.Is(err, store.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject, got %v", err)
	}
}

func TestUpdateJournalEntry(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	e := &models.JournalEntry{Summary: "old summary", Accomplished: []string{"a"}}
	if err := s.CreateJournalEntry(ctx, e); err != nil {
		t.Fatal(err)
	}

	e.Summary = "new summary"
	e.Deferred = []string{"later"}
	if err := s.UpdateJournalEntry(ctx, e); err != nil {
		t.Fatalf("UpdateJournalEntry: %v", err)
	}

	got, err := s.GetJournalEntry(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "new summary" || len(got.Deferred) != 1 || got.Deferred[0] != "later" {
		t.Fatalf("update not persisted: %+v", got)
	}
}

func TestUpdateJournalEntryNotFound(t *testing.T) {
	s := newStore(t)
	err := s.UpdateJournalEntry(context.Background(),
		&models.JournalEntry{ID: "00000000-0000-0000-0000-000000000000", Summary: "s"})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListJournalEntriesFilterAndOrder(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.CreateJournalEntry(ctx, &models.JournalEntry{Project: ptr("apollo"), Summary: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateJournalEntry(ctx, &models.JournalEntry{Project: ptr("apollo"), Summary: "second"}); err != nil {
		t.Fatal(err)
	}

	all, err := s.ListJournalEntries(ctx, store.JournalFilter{Project: ptr("apollo")})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Summary != "second" {
		t.Fatalf("expected newest-first [second, first], got %+v", all)
	}
}

func TestListJournalEntriesSince(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	e := &models.JournalEntry{Summary: "only entry"}
	if err := s.CreateJournalEntry(ctx, e); err != nil {
		t.Fatal(err)
	}

	future := e.CreatedAt.Add(time.Hour)
	none, err := s.ListJournalEntries(ctx, store.JournalFilter{Since: &future})
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("since=future should exclude the entry, got %+v", none)
	}

	past := e.CreatedAt.Add(-time.Hour)
	one, err := s.ListJournalEntries(ctx, store.JournalFilter{Since: &past})
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 {
		t.Fatalf("since=past should include the entry, got %+v", one)
	}
}

func TestCreateSolutionAndList(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")

	sol := &models.Solution{
		Project:          ptr("apollo"),
		ErrorDescription: "pgx scan fails: cannot scan NULL into *string",
		Solution:         "Model nullable FK columns as *string, not string",
		Tags:             []string{"go", "pgx"},
		SourceURL:        "https://example.test/pgx-null",
	}
	if err := s.CreateSolution(ctx, sol); err != nil {
		t.Fatalf("CreateSolution: %v", err)
	}
	if sol.ID == "" || sol.CreatedAt.IsZero() || sol.UpdatedAt.IsZero() {
		t.Errorf("id/timestamps not populated: %+v", sol)
	}

	got, err := s.ListSolutions(ctx, store.SolutionFilter{Project: ptr("apollo")})
	if err != nil {
		t.Fatalf("ListSolutions: %v", err)
	}
	if len(got) != 1 || got[0].Solution != "Model nullable FK columns as *string, not string" {
		t.Fatalf("expected 1 entry, got %+v", got)
	}
	if len(got[0].Tags) != 2 || got[0].SourceURL != "https://example.test/pgx-null" {
		t.Errorf("fields not round-tripped: %+v", got[0])
	}
}

func TestCreateSolutionGlobal(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sol := &models.Solution{
		ErrorDescription: "mneme.dev resolves slowly on macOS",
		Solution:         "Use a non-.local host; .local routes through mDNS",
	}
	if err := s.CreateSolution(ctx, sol); err != nil {
		t.Fatalf("CreateSolution global: %v", err)
	}
	got, err := s.ListSolutions(ctx, store.SolutionFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Project != nil {
		t.Fatalf("expected 1 global entry (nil project), got %+v", got)
	}
	if len(got[0].Tags) != 0 || got[0].SourceURL != "" {
		t.Errorf("expected empty tags + source_url, got %+v", got[0])
	}
}

func TestCreateSolutionUnknownProject(t *testing.T) {
	s := newStore(t)
	err := s.CreateSolution(context.Background(),
		&models.Solution{Project: ptr("ghost"), ErrorDescription: "e", Solution: "s"})
	if !errors.Is(err, store.ErrInvalidProject) {
		t.Fatalf("expected ErrInvalidProject, got %v", err)
	}
}

func TestUpdateSolution(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sol := &models.Solution{ErrorDescription: "old error", Solution: "old fix", Tags: []string{"a"}}
	if err := s.CreateSolution(ctx, sol); err != nil {
		t.Fatal(err)
	}

	sol.Solution = "new fix"
	sol.Tags = []string{"b", "c"}
	if err := s.UpdateSolution(ctx, sol); err != nil {
		t.Fatalf("UpdateSolution: %v", err)
	}

	got, err := s.GetSolution(ctx, sol.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Solution != "new fix" || len(got.Tags) != 2 {
		t.Fatalf("update not persisted: %+v", got)
	}
}

func TestUpdateSolutionNotFound(t *testing.T) {
	s := newStore(t)
	err := s.UpdateSolution(context.Background(),
		&models.Solution{ID: "00000000-0000-0000-0000-000000000000", ErrorDescription: "e", Solution: "s"})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListSolutionsFilterAndOrder(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.CreateSolution(ctx, &models.Solution{Project: ptr("apollo"), ErrorDescription: "first", Solution: "f", Tags: []string{"docker"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSolution(ctx, &models.Solution{Project: ptr("apollo"), ErrorDescription: "second", Solution: "s", Tags: []string{"docker", "macos"}}); err != nil {
		t.Fatal(err)
	}

	all, err := s.ListSolutions(ctx, store.SolutionFilter{Project: ptr("apollo")})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ErrorDescription != "second" {
		t.Fatalf("expected newest-first [second, first], got %+v", all)
	}

	tagged, err := s.ListSolutions(ctx, store.SolutionFilter{Tags: []string{"macos"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tagged) != 1 || tagged[0].ErrorDescription != "second" {
		t.Fatalf("tag filter: expected only [second], got %+v", tagged)
	}
}

func TestSearchSolutionsRankedAndLimit(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		if err := s.CreateSolution(ctx, &models.Solution{
			ErrorDescription: "container startup timeout on boot",
			Solution:         "raise the healthcheck start_period",
		}); err != nil {
			t.Fatal(err)
		}
	}
	// unrelated entry that must NOT match the query
	if err := s.CreateSolution(ctx, &models.Solution{
		ErrorDescription: "certificate expired", Solution: "regenerate with mkcert",
	}); err != nil {
		t.Fatal(err)
	}

	hits, err := s.SearchSolutions(ctx, "startup timeout", store.SolutionFilter{})
	if err != nil {
		t.Fatalf("SearchSolutions: %v", err)
	}
	if len(hits) != 4 {
		t.Fatalf("expected 4 timeout matches, got %d: %+v", len(hits), hits)
	}

	capped, err := s.SearchSolutions(ctx, "startup timeout", store.SolutionFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(capped) != 2 {
		t.Fatalf("expected limit=2 to cap at 2, got %d", len(capped))
	}
}

func TestGetProject(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")

	got, err := s.GetProject(ctx, "apollo")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Slug != "apollo" {
		t.Fatalf("expected slug apollo, got %+v", got)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	s := newStore(t)
	_, err := s.GetProject(context.Background(), "ghost")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
