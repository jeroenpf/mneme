package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// TestJournalFTSColumn proves migration 012 added a working search_vector to
// journal_entries: an entry is findable by a word from its summary.
func TestJournalFTSColumn(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.CreateJournalEntry(ctx, &models.JournalEntry{
		Project: ptr("apollo"),
		Summary: "shipped the zigbee coordinator swap",
	}); err != nil {
		t.Fatal(err)
	}

	var n int
	err := s.Pool().QueryRow(ctx,
		`SELECT count(*) FROM journal_entries
		 WHERE search_vector @@ websearch_to_tsquery('english', $1)`,
		"zigbee").Scan(&n)
	if err != nil {
		t.Fatalf("query journal search_vector: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 journal FTS match, got %d", n)
	}
}

func seedSearchCorpus(t *testing.T, s *store.PostgresStore) {
	t.Helper()
	ctx := context.Background()
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "zigbee-plan", Title: "Zigbee migration plan", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusInProgress,
		Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	// The decision text carries the bare token "zigbee" so a "zigbee" query
	// matches it. Postgres tokenizes "zigbee2mqtt" as a single numword
	// lexeme that does NOT match "zigbee" (discovered during execution — see
	// the plan's note), hence the extra bare word alongside the realistic
	// product name.
	if err := s.CreateDecision(ctx, &models.Decision{
		Title: "Use zigbee2mqtt", Project: ptr("apollo"),
		Decision: "adopt zigbee2mqtt for the zigbee mesh", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSnippet(ctx, &models.Snippet{
		Title: "zigbee pairing helper", Project: ptr("apollo"),
		Language: "go", Content: "// zigbee pair", Tags: []string{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSolution(ctx, &models.Solution{
		Project: ptr("apollo"), ErrorDescription: "zigbee coordinator offline",
		Solution: "reflash the stick", Tags: []string{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateJournalEntry(ctx, &models.JournalEntry{
		Project: ptr("apollo"), Summary: "zigbee coordinator swap done",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSearchSpansAllTypes(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo")
	seedSearchCorpus(t, s)

	hits, err := s.Search(context.Background(), "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	got := map[string]bool{}
	for _, h := range hits {
		got[h.Type] = true
		if h.Title == "" || h.Score <= 0 {
			t.Errorf("bad hit: %+v", h)
		}
	}
	for _, ty := range store.SearchTypes {
		if !got[ty] {
			t.Errorf("expected a %s hit, got types %v", ty, got)
		}
	}
}

func TestSearchTypeAndProjectFilter(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo", "hyperion")
	seedSearchCorpus(t, s)
	// a decision under a different project that must be filtered out by project
	if err := s.CreateDecision(context.Background(), &models.Decision{
		Title: "zigbee elsewhere", Project: ptr("hyperion"),
		Decision: "zigbee", Status: models.DecisionAccepted,
	}); err != nil {
		t.Fatal(err)
	}

	// types filter: only decisions
	hits, err := s.Search(context.Background(), "zigbee",
		store.SearchFilter{Types: []string{"decisions"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if h.Type != "decisions" {
			t.Errorf("types filter leaked: %+v", h)
		}
	}
	// project filter: only apollo decisions
	hits, err = s.Search(context.Background(), "zigbee",
		store.SearchFilter{Types: []string{"decisions"}, Project: ptr("apollo")})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if h.Project == nil || *h.Project != "apollo" {
			t.Errorf("project filter leaked: %+v", h)
		}
	}
}

func TestSearchUnknownTypeErrors(t *testing.T) {
	s := newStore(t)
	_, err := s.Search(context.Background(), "x", store.SearchFilter{Types: []string{"bogus"}})
	if !errors.Is(err, store.ErrInvalidSearchType) {
		t.Fatalf("expected ErrInvalidSearchType, got %v", err)
	}
}

func TestSearchLimit(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo")
	seedSearchCorpus(t, s)
	hits, err := s.Search(context.Background(), "zigbee", store.SearchFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("limit=2 should cap at 2, got %d", len(hits))
	}
}

func TestSearchHybridFusesVectorAndFTS(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	// A doc that does NOT contain the FTS term, reachable only by vector.
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "sem-only", Title: "Wireless mesh notes", Project: ptr("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{}, Body: map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	// Give it an embedding near the query vector.
	qv := fakeVec(0.9)
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{{
		SourceType: "documents", SourceID: "sem-only", ChunkID: "full",
		ChunkText: "wireless mesh coordinator", Embedding: qv,
		Project: ptr("apollo"), SourceTitle: "Wireless mesh notes", Model: "fake",
	}}); err != nil {
		t.Fatal(err)
	}

	// FTS-only (nil vector): 'zigbee' does not match "Wireless mesh notes".
	fts, err := s.Search(ctx, "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range fts {
		if h.ID == "sem-only" {
			t.Fatal("FTS-only should not surface the semantic-only doc")
		}
	}

	// Hybrid: with the query vector, the semantic-only doc appears.
	hyb, err := s.Search(ctx, "zigbee", store.SearchFilter{Vector: qv})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, h := range hyb {
		if h.ID == "sem-only" {
			found = true
			if h.Title == "" || h.Excerpt == "" || h.Score <= 0 {
				t.Errorf("vector-only hit missing projected fields: %+v", h)
			}
		}
	}
	if !found {
		t.Fatalf("hybrid search should surface the semantic-only doc, got %+v", hyb)
	}
}

func TestSearchNilVectorMatchesFTSPath(t *testing.T) {
	s := newStore(t)
	seedProjects(t, s, "apollo")
	seedSearchCorpus(t, s) // from 2.8a
	a, err := s.Search(context.Background(), "zigbee", store.SearchFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(a) == 0 {
		t.Fatal("expected FTS hits for zigbee")
	}
}
