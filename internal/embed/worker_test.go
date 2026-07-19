package embed_test

import (
	"context"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/models"
)

// fakeClient returns a deterministic 1024-dim vector per input. model
// defaults to "fake" so existing tests need not set it.
type fakeClient struct {
	calls int
	model string
}

func (f *fakeClient) Model() string {
	if f.model == "" {
		return "fake"
	}
	return f.model
}
func (f *fakeClient) Embed(_ context.Context, texts []string, _ string) ([][]float32, error) {
	f.calls += len(texts)
	out := make([][]float32, len(texts))
	for i := range texts {
		v := make([]float32, 1024)
		v[0] = float32(i + 1)
		out[i] = v
	}
	return out, nil
}

func TestWorkerEmbedsAndPrunes(t *testing.T) {
	s := newEmbedStore(t) // container-backed store, mirrors store.newStore
	ctx := context.Background()
	seedProject(t, s, "apollo")
	doc := &models.Document{
		ID: "d1", Title: "Zigbee", Project: ptrs("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "O", "content": "coordinator"},
		}},
	}
	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}

	fc := &fakeClient{}
	w := embed.NewWorker(s, fc, 8, 0) // rpm=0: no throttle in tests
	if err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "d1"}); err != nil {
		t.Fatalf("Process: %v", err)
	}
	got, _ := s.EmbeddingsFor(ctx, "documents", "d1")
	if got["overview"] == "" {
		t.Fatalf("expected an embedding for section overview, got %+v", got)
	}

	// Re-process unchanged: no new embed calls (chunk_text diff skips it).
	before := fc.calls
	if err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "d1"}); err != nil {
		t.Fatal(err)
	}
	if fc.calls != before {
		t.Fatalf("unchanged re-process should not call Embed: before=%d after=%d", before, fc.calls)
	}

	// Remove the section → re-process prunes the stale chunk.
	doc.Body = map[string]any{"sections": []any{}}
	if err := s.UpdateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "d1"}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.EmbeddingsFor(ctx, "documents", "d1")
	if _, ok := got["overview"]; ok {
		t.Fatalf("stale section chunk not pruned: %+v", got)
	}
}

// A source that vanishes between enqueue and Process must have its embeddings
// purged, not left orphaned. load() returns src==nil on ErrNotFound; Process
// must delete the source's chunks before returning.
func TestWorkerPurgesEmbeddingsForDeletedSource(t *testing.T) {
	s := newEmbedStore(t)
	ctx := context.Background()
	seedProject(t, s, "apollo")
	doc := &models.Document{
		ID: "gone", Title: "Doomed", Project: ptrs("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "O", "content": "coordinator"},
		}},
	}
	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}

	w := embed.NewWorker(s, &fakeClient{}, 8, 0)
	if err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "gone"}); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got, _ := s.EmbeddingsFor(ctx, "documents", "gone"); len(got) == 0 {
		t.Fatalf("precondition: expected an embedding before deletion, got %+v", got)
	}

	// Delete the underlying source, then re-Process its queued ref.
	if _, err := s.Pool().Exec(ctx, `DELETE FROM documents WHERE id=$1`, "gone"); err != nil {
		t.Fatal(err)
	}
	if err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "gone"}); err != nil {
		t.Fatalf("Process after delete: %v", err)
	}
	got, _ := s.EmbeddingsFor(ctx, "documents", "gone")
	if len(got) != 0 {
		t.Fatalf("deleted source left orphaned embeddings: %+v", got)
	}
}

// Reconciliation only enqueues live sources, so a deleted source's vectors are
// never re-processed. ReconcileAll must sweep them itself so the index
// self-heals after missed delete events.
func TestReconcileAllSweepsOrphans(t *testing.T) {
	s := newEmbedStore(t)
	ctx := context.Background()
	if err := s.UpsertEmbeddings(ctx, []models.Embedding{
		{SourceType: "documents", SourceID: "ghost", ChunkID: "c0", ChunkText: "x",
			Embedding: make([]float32, 1024), SourceTitle: "Ghost", Model: "m"},
	}); err != nil {
		t.Fatal(err)
	}

	w := embed.NewWorker(s, &fakeClient{}, 8, 0)
	if err := w.ReconcileAll(ctx); err != nil {
		t.Fatalf("ReconcileAll: %v", err)
	}
	if got, _ := s.EmbeddingsFor(ctx, "documents", "ghost"); len(got) != 0 {
		t.Fatalf("ReconcileAll did not sweep orphan vector: %+v", got)
	}
}

// A source's stored vectors must all reflect the current embedding model.
// Switching models must re-embed the source even when its chunk text is
// unchanged, replacing the stale-model vectors.
func TestProcessReembedsOnModelChange(t *testing.T) {
	s := newEmbedStore(t)
	ctx := context.Background()
	seedProject(t, s, "apollo")
	doc := &models.Document{
		ID: "m1", Title: "Model bump", Project: ptrs("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "O", "content": "coordinator"},
		}},
	}
	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	ref := embed.SourceRef{Type: "documents", ID: "m1"}

	if err := embed.NewWorker(s, &fakeClient{model: "v1"}, 8, 0).Process(ctx, ref); err != nil {
		t.Fatalf("Process v1: %v", err)
	}

	// Same text, new model: must re-embed and replace the stored vector.
	fc2 := &fakeClient{model: "v2"}
	if err := embed.NewWorker(s, fc2, 8, 0).Process(ctx, ref); err != nil {
		t.Fatalf("Process v2: %v", err)
	}
	if fc2.calls == 0 {
		t.Fatal("model change should force re-embed of unchanged-text chunks")
	}
	var model string
	if err := s.Pool().QueryRow(ctx,
		`SELECT model FROM embeddings WHERE source_type='documents' AND source_id=$1 AND chunk_id='overview'`,
		"m1").Scan(&model); err != nil {
		t.Fatal(err)
	}
	if model != "v2" {
		t.Fatalf("stored vector not replaced under new model: got %q, want v2", model)
	}
}
