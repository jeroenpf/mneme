package embed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// failClient always errors, to exercise terminal-failure recording.
type failClient struct{}

func (failClient) Model() string { return "fake" }
func (failClient) Embed(context.Context, []string, string) ([][]float32, error) {
	return nil, errors.New("embed boom")
}

func hasRef(refs []store.SourceRef, id string) bool {
	for _, r := range refs {
		if r.ID == id {
			return true
		}
	}
	return false
}

func mkSectionDoc(t *testing.T, s interface {
	CreateDocument(context.Context, *models.Document) error
}, id string) {
	t.Helper()
	if err := s.CreateDocument(context.Background(), &models.Document{
		ID: id, Title: id, Project: ptrs("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "O", "content": "coordinator"},
		}},
	}); err != nil {
		t.Fatal(err)
	}
}

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
	if _, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "d1"}); err != nil {
		t.Fatalf("Process: %v", err)
	}
	got, _ := s.EmbeddingsFor(ctx, "documents", "d1")
	if got["overview"] == "" {
		t.Fatalf("expected an embedding for section overview, got %+v", got)
	}

	// Re-process unchanged: no new embed calls (chunk_text diff skips it).
	before := fc.calls
	if _, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "d1"}); err != nil {
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
	if _, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "d1"}); err != nil {
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
	if _, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "gone"}); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got, _ := s.EmbeddingsFor(ctx, "documents", "gone"); len(got) == 0 {
		t.Fatalf("precondition: expected an embedding before deletion, got %+v", got)
	}

	// Delete the underlying source, then re-Process its queued ref.
	if _, err := s.Pool().Exec(ctx, `DELETE FROM documents WHERE id=$1`, "gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "gone"}); err != nil {
		t.Fatalf("Process after delete: %v", err)
	}
	got, _ := s.EmbeddingsFor(ctx, "documents", "gone")
	if len(got) != 0 {
		t.Fatalf("deleted source left orphaned embeddings: %+v", got)
	}
}

// A source whose embed errors must be recorded as a terminal failure (for
// visibility + manual retry); a later successful embed clears it.
func TestWorkerRecordsFailureOnEmbedError(t *testing.T) {
	s := newEmbedStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seedProject(t, s, "apollo")
	mkSectionDoc(t, s, "f1")

	w := embed.NewWorker(s, failClient{}, 8, 0)
	go w.Run(ctx)
	w.Enqueue(embed.SourceRef{Type: "documents", ID: "f1"})

	deadline := time.Now().Add(3 * time.Second)
	for {
		refs, _ := s.FailedSourceRefs(ctx)
		if hasRef(refs, "f1") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("embed error was not recorded as a terminal failure")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWorkerClearsFailureOnSuccess(t *testing.T) {
	s := newEmbedStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seedProject(t, s, "apollo")
	mkSectionDoc(t, s, "f2")
	if err := s.RecordEmbedFailure(ctx, "documents", "f2", "old boom"); err != nil {
		t.Fatal(err)
	}

	w := embed.NewWorker(s, &fakeClient{}, 8, 0) // succeeds
	go w.Run(ctx)
	w.Enqueue(embed.SourceRef{Type: "documents", ID: "f2"})

	deadline := time.Now().Add(3 * time.Second)
	for {
		refs, _ := s.FailedSourceRefs(ctx)
		if !hasRef(refs, "f2") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("successful embed did not clear the recorded failure")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Periodic reconciliation must self-heal sources whose enqueue event was
// missed (dropped signal, crash): a source created without being enqueued is
// still discovered and embedded by a later reconcile pass.
func TestReconcileSelfHealsMissedSources(t *testing.T) {
	s := newEmbedStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seedProject(t, s, "apollo")

	w := embed.NewWorker(s, &fakeClient{}, 8, 0)
	go w.Run(ctx)
	go w.Reconcile(ctx, 50*time.Millisecond)

	// Create a source WITHOUT enqueuing it — simulates a missed event.
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "heal1", Title: "Missed", Project: ptrs("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "O", "content": "coordinator"},
		}},
	}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		if got, _ := s.EmbeddingsFor(ctx, "documents", "heal1"); got["overview"] != "" {
			return // a periodic pass discovered and embedded it
		}
		if time.Now().After(deadline) {
			t.Fatal("periodic reconcile did not self-heal the missed source")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Run drains enqueued work through the deduplicated pending queue: an
// enqueued source is picked up and embedded without any channel-buffer drop.
func TestWorkerRunDrainsEnqueued(t *testing.T) {
	s := newEmbedStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seedProject(t, s, "apollo")
	if err := s.CreateDocument(ctx, &models.Document{
		ID: "q1", Title: "Queued", Project: ptrs("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "O", "content": "coordinator"},
		}},
	}); err != nil {
		t.Fatal(err)
	}

	w := embed.NewWorker(s, &fakeClient{}, 8, 0)
	go w.Run(ctx)
	// Enqueue the same source twice — dedup must not break delivery.
	w.Enqueue(embed.SourceRef{Type: "documents", ID: "q1"})
	w.Enqueue(embed.SourceRef{Type: "documents", ID: "q1"})

	deadline := time.Now().Add(3 * time.Second)
	for {
		if got, _ := s.EmbeddingsFor(ctx, "documents", "q1"); got["overview"] != "" {
			return // delivered and embedded
		}
		if time.Now().After(deadline) {
			t.Fatal("Run did not drain the enqueued source within 3s")
		}
		time.Sleep(20 * time.Millisecond)
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

// Process reports whether it actually called the provider, so Run only spends
// rate-limit time on real requests: a warm (unchanged) source makes no API
// call and must report no embed, gating the throttle.
func TestProcessReportsEmbedActivity(t *testing.T) {
	s := newEmbedStore(t)
	ctx := context.Background()
	seedProject(t, s, "apollo")
	doc := &models.Document{
		ID: "act1", Title: "Activity", Project: ptrs("apollo"),
		Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
		Body: map[string]any{"sections": []any{
			map[string]any{"type": "section", "id": "overview", "title": "O", "content": "coordinator"},
		}},
	}
	if err := s.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	w := embed.NewWorker(s, &fakeClient{}, 8, 0)
	ref := embed.SourceRef{Type: "documents", ID: "act1"}

	embedded, err := w.Process(ctx, ref)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !embedded {
		t.Fatal("fresh source should report an embed")
	}

	embedded, err = w.Process(ctx, ref)
	if err != nil {
		t.Fatalf("Process (warm): %v", err)
	}
	if embedded {
		t.Fatal("unchanged source should report no embed (no rate-limit spend)")
	}
}

// The limiter must space real provider requests (t2: rate-limit actual API
// requests) while never delaying a warm source that makes no request.
func TestWorkerRateLimitsOnlyRealRequests(t *testing.T) {
	s := newEmbedStore(t)
	ctx := context.Background()
	seedProject(t, s, "apollo")
	mkdoc := func(id, content string) {
		if err := s.CreateDocument(ctx, &models.Document{
			ID: id, Title: id, Project: ptrs("apollo"),
			Type: models.TypePlan, Status: models.StatusTodo, Tags: []string{}, Meta: map[string]any{},
			Body: map[string]any{"sections": []any{
				map[string]any{"type": "section", "id": "overview", "title": "O", "content": content},
			}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	mkdoc("rl1", "coordinator one")
	mkdoc("rl2", "coordinator two")

	// rpm=600 => 100ms between real requests.
	w := embed.NewWorker(s, &fakeClient{}, 8, 600)

	// First real request passes immediately; the second is spaced ~100ms.
	if _, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "rl1"}); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "rl2"}); err != nil {
		t.Fatal(err)
	}
	if spaced := time.Since(start); spaced < 70*time.Millisecond {
		t.Fatalf("consecutive real requests not rate-limited: gap %v", spaced)
	}

	// A warm (unchanged) source skips the limiter and returns promptly.
	start = time.Now()
	embedded, err := w.Process(ctx, embed.SourceRef{Type: "documents", ID: "rl1"})
	if err != nil {
		t.Fatal(err)
	}
	if embedded {
		t.Fatal("warm source should not embed")
	}
	if delay := time.Since(start); delay > 60*time.Millisecond {
		t.Fatalf("warm source incurred rate-limit delay (%v); limiter must gate only real requests", delay)
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

	if _, err := embed.NewWorker(s,&fakeClient{model: "v1"}, 8, 0).Process(ctx, ref); err != nil {
		t.Fatalf("Process v1: %v", err)
	}

	// Same text, new model: must re-embed and replace the stored vector.
	fc2 := &fakeClient{model: "v2"}
	if _, err := embed.NewWorker(s,fc2, 8, 0).Process(ctx, ref); err != nil {
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
