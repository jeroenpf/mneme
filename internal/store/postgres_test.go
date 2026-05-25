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
		`TRUNCATE documents, projects, embeddings RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return store.NewWithPool(pool)
}

func ptr[T any](v T) *T { return &v }

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
