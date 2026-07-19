package embed_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// seedEmbeddedCorpus creates n documents and embeds each once, leaving a fully
// warm index (every source already up to date on the current model).
func seedEmbeddedCorpus(tb testing.TB, s *store.PostgresStore, w *embed.Worker, n int) {
	tb.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("doc-%d", i)
		if err := s.CreateDocument(ctx, &models.Document{
			ID: id, Title: id, Project: ptrs("apollo"),
			Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
			Body: map[string]any{"sections": []any{
				map[string]any{"type": "section", "id": "overview", "title": "O",
					"content": fmt.Sprintf("coordinator body %d", i)},
			}},
		}); err != nil {
			tb.Fatal(err)
		}
		if _, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: id}); err != nil {
			tb.Fatal(err)
		}
	}
}

func warmPass(tb testing.TB, s *store.PostgresStore, w *embed.Worker) {
	ctx := context.Background()
	refs, err := s.SourceRefs(ctx)
	if err != nil {
		tb.Fatal(err)
	}
	for _, r := range refs {
		if _, err := w.Process(ctx, embed.SourceRef{Type: r.Type, ID: r.ID}); err != nil {
			tb.Fatal(err)
		}
	}
}

// A warm reconciliation must make zero provider calls — this is what keeps a
// warm pass in the seconds range rather than hours of needless re-embedding.
func TestWarmReconcileMakesNoProviderCalls(t *testing.T) {
	s := newEmbedStore(t)
	seedProject(t, s, "apollo")
	fc := &fakeClient{}
	w := embed.NewWorker(s, fc, 64, 0)
	seedEmbeddedCorpus(t, s, w, 20)

	before := fc.calls
	warmPass(t, s, w)
	if fc.calls != before {
		t.Fatalf("warm reconcile must make zero provider calls, before=%d after=%d", before, fc.calls)
	}
}

// BenchmarkWarmReconcile measures a full warm pass (SourceRefs + per-source
// diff, zero provider calls) over a personal-scale corpus. Run with:
//
//	go test ./internal/embed/ -run=^$ -bench=BenchmarkWarmReconcile -benchtime=10x
//
// Target: a warm reconcile of a personal-scale corpus stays in the seconds
// range, not the hours a naive re-embed-everything pass would take. Measured
// ~0.36s for 200 sources (Apple M2 Pro) — ~1.8ms/source, DB-bound with zero
// Voyage calls, thanks to the chunk diff and P3-t1 (no throttle on warm
// sources). Regression guard: the warm pass must make no provider calls.
func BenchmarkWarmReconcile(b *testing.B) {
	s := newEmbedStore(b)
	seedProject(b, s, "apollo")
	fc := &fakeClient{}
	w := embed.NewWorker(s, fc, 512, 0)
	seedEmbeddedCorpus(b, s, w, 200)
	embedCalls := fc.calls

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warmPass(b, s, w)
	}
	b.StopTimer()

	if fc.calls != embedCalls {
		b.Fatalf("warm benchmark made provider calls: seed=%d after=%d", embedCalls, fc.calls)
	}
}
