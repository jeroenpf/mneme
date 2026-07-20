package store_test

import (
	"context"
	"testing"

	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
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

	cov, err := s.EmbeddingStatus(ctx, "m")
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string]store.TypeStatus{}
	for _, c := range cov {
		byType[c.Type] = c
	}
	if byType["decisions"].Total < 1 || byType["decisions"].Embedded != 0 || byType["decisions"].Missing < 1 {
		t.Fatalf("status wrong for decisions: %+v", byType["decisions"])
	}
}

// Status must count only embeddings whose source_id still resolves to a live
// source row: an orphan must not inflate Embedded, but must surface as Orphaned.
func TestEmbeddingStatusExcludesOrphans(t *testing.T) {
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

	cov, err := s.EmbeddingStatus(ctx, "m")
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string]store.TypeStatus{}
	for _, c := range cov {
		byType[c.Type] = c
	}
	if got := byType["documents"]; got.Total != 1 || got.Embedded != 1 ||
		got.Reconciled != 1 || got.Orphaned != 1 {
		t.Fatalf("status should count only the live source and flag the orphan, got %+v", got)
	}
}

// DeleteOrphanEmbeddings removes vectors whose source_id no longer resolves
// to a live row of that type, across every embeddable type, while leaving
// live sources' vectors intact.
func TestDeleteOrphanEmbeddings(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	if err := s.CreateDocument(ctx, sampleDoc("d1", "Live")); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "d1", ChunkID: "overview", ChunkText: "a",
			Embedding: fakeVec(0.1), SourceTitle: "Live", Model: "m"},
		// Orphan document (no such id) and orphan decision (no such uuid).
		{SourceType: "documents", SourceID: "ghost", ChunkID: "overview", ChunkText: "b",
			Embedding: fakeVec(0.2), SourceTitle: "Ghost", Model: "m"},
		{SourceType: "decisions", SourceID: "11111111-1111-1111-1111-111111111111", ChunkID: "c0",
			ChunkText: "c", Embedding: fakeVec(0.3), SourceTitle: "Dead", Model: "m"},
	}); err != nil {
		t.Fatal(err)
	}

	n, err := s.DeleteOrphanEmbeddings(ctx)
	if err != nil {
		t.Fatalf("DeleteOrphanEmbeddings: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 orphan rows swept, got %d", n)
	}
	if got, _ := s.EmbeddingsFor(ctx, "documents", "d1"); len(got) != 1 {
		t.Fatalf("live source vector must survive the sweep: %+v", got)
	}
	if got, _ := s.EmbeddingsFor(ctx, "documents", "ghost"); len(got) != 0 {
		t.Fatalf("orphan document vector not swept: %+v", got)
	}
	if got, _ := s.EmbeddingsFor(ctx, "decisions", "11111111-1111-1111-1111-111111111111"); len(got) != 0 {
		t.Fatalf("orphan decision vector not swept: %+v", got)
	}
}

// EmbeddingStatus splits each type's live sources into reconciled (embedded on
// the current model), stale (embedded but on an outdated model), and missing
// (no vector), and counts orphaned vectors (source gone) separately. Failed is
// nil until P3-t5 tracks terminal failures.
func TestEmbeddingStatusBuckets(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	const cur = "voyage-cur"
	for _, id := range []string{"recon", "stale1", "miss"} {
		if err := s.CreateDocument(ctx, sampleDoc(id, id)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "recon", ChunkID: "full", ChunkText: "a",
			Embedding: fakeVec(0.1), SourceTitle: "recon", Model: cur},
		{SourceType: "documents", SourceID: "stale1", ChunkID: "full", ChunkText: "b",
			Embedding: fakeVec(0.2), SourceTitle: "stale1", Model: "voyage-old"},
		// Orphan: no such document.
		{SourceType: "documents", SourceID: "ghost", ChunkID: "full", ChunkText: "c",
			Embedding: fakeVec(0.3), SourceTitle: "ghost", Model: cur},
	}); err != nil {
		t.Fatal(err)
	}

	st, err := s.EmbeddingStatus(ctx, cur)
	if err != nil {
		t.Fatalf("EmbeddingStatus: %v", err)
	}
	byType := map[string]store.TypeStatus{}
	for _, x := range st {
		byType[x.Type] = x
	}
	d := byType["documents"]
	if d.Total != 3 || d.Embedded != 2 || d.Reconciled != 1 ||
		d.Missing != 1 || d.Stale != 1 || d.Orphaned != 1 {
		t.Fatalf("documents status wrong: %+v", d)
	}
	if d.Failed != 0 {
		t.Fatalf("no failures recorded, failed should be 0, got %d", d.Failed)
	}
}

// Terminal embed failures are recorded per source, feed the live "failed"
// bucket, and are listed for manual retry. A failure for a deleted source is
// still retryable but must not inflate the live bucket.
func TestEmbedFailureTracking(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.CreateDocument(ctx, sampleDoc("d1", "Live")); err != nil {
		t.Fatal(err)
	}

	if err := s.RecordEmbedFailure(ctx, "documents", "d1", "boom"); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordEmbedFailure(ctx, "documents", "d1", "boom again"); err != nil {
		t.Fatal(err)
	}
	// A failure for a source with no live row (already deleted).
	if err := s.RecordEmbedFailure(ctx, "documents", "ghost", "gone"); err != nil {
		t.Fatal(err)
	}

	refs, err := s.FailedSourceRefs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("both failed refs should be retryable, got %+v", refs)
	}

	byType := func() map[string]store.TypeStatus {
		sts, err := s.EmbeddingStatus(ctx, "m")
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]store.TypeStatus{}
		for _, x := range sts {
			m[x.Type] = x
		}
		return m
	}
	if got := byType()["documents"].Failed; got != 1 {
		t.Fatalf("failed bucket should count only the live failed source, got %d", got)
	}

	if err := s.ClearEmbedFailure(ctx, "documents", "d1"); err != nil {
		t.Fatal(err)
	}
	if got := byType()["documents"].Failed; got != 0 {
		t.Fatalf("cleared failure should drop the bucket to 0, got %d", got)
	}
}

func TestHasStaleModelEmbeddings(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "d1", ChunkID: "overview", ChunkText: "a",
			Embedding: fakeVec(0.1), SourceTitle: "D", Model: "voyage-3"},
	}); err != nil {
		t.Fatal(err)
	}

	// Same model → not stale; a newer model → stale.
	if stale, err := s.HasStaleModelEmbeddings(ctx, "documents", "d1", "voyage-3"); err != nil || stale {
		t.Fatalf("same model should not be stale: stale=%v err=%v", stale, err)
	}
	if stale, err := s.HasStaleModelEmbeddings(ctx, "documents", "d1", "voyage-4"); err != nil || !stale {
		t.Fatalf("changed model should be stale: stale=%v err=%v", stale, err)
	}
	// A source with no vectors at all is not stale.
	if stale, err := s.HasStaleModelEmbeddings(ctx, "documents", "absent", "voyage-4"); err != nil || stale {
		t.Fatalf("absent source should not be stale: stale=%v err=%v", stale, err)
	}
}
