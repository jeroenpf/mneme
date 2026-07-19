package store_test

import (
	"context"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

func fakeVec(seed float32) []float32 {
	v := make([]float32, 1024) // matches migration 013 → vector(1024)
	for i := range v {
		v[i] = seed
	}
	return v
}

func TestUpsertAndEmbeddingsFor(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")

	rows := []models.Embedding{
		{SourceType: "documents", SourceID: "d1", ChunkID: "overview", ChunkText: "a",
			Embedding: fakeVec(0.1), Project: ptr("apollo"), SourceTitle: "Doc", Model: "voyage-4-large"},
		{SourceType: "documents", SourceID: "d1", ChunkID: "risks", ChunkText: "b",
			Embedding: fakeVec(0.2), Project: ptr("apollo"), SourceTitle: "Doc", Model: "voyage-4-large"},
	}
	if err := s.UpsertEmbeddings(ctx, rows); err != nil {
		t.Fatalf("UpsertEmbeddings: %v", err)
	}

	got, err := s.EmbeddingsFor(ctx, "documents", "d1")
	if err != nil {
		t.Fatalf("EmbeddingsFor: %v", err)
	}
	if got["overview"] != "a" || got["risks"] != "b" {
		t.Fatalf("chunk_text map wrong: %+v", got)
	}

	// upsert same key updates chunk_text (no duplicate row)
	rows[0].ChunkText = "a2"
	if err := s.UpsertEmbeddings(ctx, rows[:1]); err != nil {
		t.Fatal(err)
	}
	got, _ = s.EmbeddingsFor(ctx, "documents", "d1")
	if got["overview"] != "a2" || len(got) != 2 {
		t.Fatalf("expected upsert to update, got %+v", got)
	}
}

func TestDeleteEmbeddingsExcept(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "d1", ChunkID: "keep", ChunkText: "k", Embedding: fakeVec(0.1), SourceTitle: "D", Model: "m"},
		{SourceType: "documents", SourceID: "d1", ChunkID: "stale", ChunkText: "s", Embedding: fakeVec(0.2), SourceTitle: "D", Model: "m"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteEmbeddingsExcept(ctx, "documents", "d1", []string{"keep"}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.EmbeddingsFor(ctx, "documents", "d1")
	if _, ok := got["stale"]; ok || len(got) != 1 {
		t.Fatalf("stale chunk not pruned: %+v", got)
	}
}

func TestSourceRefsAndCoverage(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedProjects(t, s, "apollo")
	if err := s.CreateDecision(ctx, &models.Decision{Title: "t", Project: ptr("apollo"), Decision: "d", Status: models.DecisionAccepted}); err != nil {
		t.Fatal(err)
	}
	refs, err := s.SourceRefs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var haveDecision bool
	for _, r := range refs {
		if r.Type == "decisions" {
			haveDecision = true
		}
	}
	if !haveDecision {
		t.Fatalf("SourceRefs missing the decision: %+v", refs)
	}

	cov, err := s.EmbeddingCoverage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string]store.TypeCoverage{}
	for _, c := range cov {
		byType[c.Type] = c
	}
	if byType["decisions"].Total < 1 || byType["decisions"].Embedded != 0 {
		t.Fatalf("coverage wrong for decisions: %+v", byType["decisions"])
	}
}

// Coverage must count only embeddings whose source_id still resolves to a
// live source row; embeddings orphaned by a deleted source must not inflate
// the embedded count.
func TestEmbeddingCoverageExcludesOrphans(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	if err := s.CreateDocument(ctx, sampleDoc("d1", "Live")); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "d1", ChunkID: "overview", ChunkText: "a",
			Embedding: fakeVec(0.1), SourceTitle: "Live", Model: "m"},
		// Orphan: no documents row has id 'ghost'.
		{SourceType: "documents", SourceID: "ghost", ChunkID: "overview", ChunkText: "b",
			Embedding: fakeVec(0.2), SourceTitle: "Ghost", Model: "m"},
	}); err != nil {
		t.Fatal(err)
	}

	cov, err := s.EmbeddingCoverage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string]store.TypeCoverage{}
	for _, c := range cov {
		byType[c.Type] = c
	}
	if got := byType["documents"]; got.Total != 1 || got.Embedded != 1 {
		t.Fatalf("coverage should count only the live source, got %+v", got)
	}
}
