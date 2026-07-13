package embed_test

import (
	"context"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/models"
)

// fakeClient returns a deterministic 1024-dim vector per input.
type fakeClient struct{ calls int }

func (f *fakeClient) Model() string { return "fake" }
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
