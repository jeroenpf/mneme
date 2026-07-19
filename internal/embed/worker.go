package embed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jeroenpfeil/mneme/internal/models"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// SourceRef is a queue token naming a source to (re)embed.
type SourceRef struct {
	Type string
	ID   string
}

// Enqueuer accepts embedding jobs. Write tools depend on this interface so
// they're agnostic to whether embedding is enabled.
type Enqueuer interface {
	Enqueue(SourceRef)
}

// NopEnqueuer is used when embedding is disabled (no key).
type NopEnqueuer struct{}

func (NopEnqueuer) Enqueue(SourceRef) {}

// Worker embeds sources off a deduplicated, non-dropping in-memory queue.
// Jobs are best-effort: failures are logged and recovered by the next
// ReconcileAll.
type Worker struct {
	store   store.Store
	client  Client
	limiter *rateLimiter  // spaces actual provider requests under a low RPM tier
	pending *pendingSet   // dedup, non-dropping work queue
	signal  chan struct{} // buffered(1) wake-up for Run when new work arrives
}

// NewWorker builds the worker. rpm>0 installs a proactive rate limiter
// (60s/rpm) that spaces only real embedding requests, for accounts on the low
// no-payment-method tier; rpm=0 relies solely on the client's 429 backoff.
// buf is an initial capacity hint for the pending queue.
func NewWorker(st store.Store, c Client, buf, rpm int) *Worker {
	return &Worker{
		store:   st,
		client:  c,
		limiter: newRateLimiter(rpm),
		pending: newPendingSet(buf),
		signal:  make(chan struct{}, 1),
	}
}

// Enqueue is non-blocking and never drops: it coalesces repeat refs for the
// same source (dedup) and wakes Run. A slow Voyage never stalls an MCP tool,
// and bursts are held rather than lost.
func (w *Worker) Enqueue(ref SourceRef) {
	if w.pending.Add(ref) {
		select {
		case w.signal <- struct{}{}:
		default: // a wake-up is already queued; Run will drain all pending
		}
	}
}

// Run drains the pending queue until ctx is cancelled. Rate limiting lives in
// Process (it waits the limiter only around an actual provider request), so
// warm sources drain at full speed and the loop itself never throttles.
func (w *Worker) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		ref, ok := w.pending.Next()
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-w.signal:
			}
			continue
		}
		if _, err := w.Process(ctx, ref); err != nil {
			slog.Error("embed failed", "type", ref.Type, "id", ref.ID, "err", err)
		}
	}
}

// reconcilePassTimeout bounds a single reconciliation pass (orphan sweep +
// SourceRefs + enqueue) so a slow DB can't wedge the loop. The actual
// embedding runs asynchronously in Run and is intentionally not bounded here.
const reconcilePassTimeout = 2 * time.Minute

// Reconcile runs a bounded reconciliation immediately (startup backfill), then
// repeats every `every` until ctx is cancelled, so missed enqueue events
// (dropped signals, crashes, restarts) self-heal. every<=0 runs only the
// single startup pass. Each pass is bounded by reconcilePassTimeout.
func (w *Worker) Reconcile(ctx context.Context, every time.Duration) {
	for {
		pass, cancel := context.WithTimeout(ctx, reconcilePassTimeout)
		if err := w.ReconcileAll(pass); err != nil {
			slog.Error("reconcile pass failed", "err", err)
		}
		cancel()
		if every <= 0 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}

// ReconcileAll enqueues every source; the per-job chunk_text diff makes a
// warm DB cheap (no Voyage calls when nothing changed). It first sweeps
// orphaned vectors — deleted sources are never enqueued, so this is the only
// path that collects the chunks they left behind.
func (w *Worker) ReconcileAll(ctx context.Context) error {
	if n, err := w.store.DeleteOrphanEmbeddings(ctx); err != nil {
		return fmt.Errorf("sweep orphans: %w", err)
	} else if n > 0 {
		slog.Info("reconcile swept orphaned embeddings", "rows", n)
	}
	refs, err := w.store.SourceRefs(ctx)
	if err != nil {
		return err
	}
	for _, r := range refs {
		w.Enqueue(SourceRef{Type: r.Type, ID: r.ID})
	}
	return nil
}

// Process embeds a single source, enforcing the replacement contract: a
// source's stored vectors always reflect its current chunk set under the
// current model.
//
//   - Delete: a source with no live row has all its vectors purged.
//   - Rewrite: added/changed chunks are re-embedded, removed chunks pruned,
//     unchanged chunks kept.
//   - Model change: if any stored vector is on a different model, the whole
//     source is re-embedded so vector spaces never mix.
//
// It returns embedded=true iff it called the provider (the chunk diff found
// new, changed, or stale-model work), so Run throttles only after real API
// requests. Exported for direct testing.
func (w *Worker) Process(ctx context.Context, ref SourceRef) (embedded bool, err error) {
	src, title, project, err := w.load(ctx, ref)
	if err != nil {
		return false, err
	}
	if src == nil {
		// The source was deleted between enqueue and now: purge any
		// embeddings it left behind (passing keep=nil deletes them all)
		// so orphans never linger in search or coverage.
		return false, w.store.DeleteEmbeddingsExcept(ctx, ref.Type, ref.ID, nil)
	}
	chunks := Chunks(src)
	existing, err := w.store.EmbeddingsFor(ctx, ref.Type, ref.ID)
	if err != nil {
		return false, err
	}
	// A model switch leaves stale-model vectors that must all be replaced,
	// so re-embed the full chunk set even where the text is unchanged.
	staleModel, err := w.store.HasStaleModelEmbeddings(ctx, ref.Type, ref.ID, w.client.Model())
	if err != nil {
		return false, err
	}

	// The chunk comparison runs before any provider call, so unchanged
	// sources never reach the rate limiter.
	var toEmbed []Chunk
	for _, c := range chunks {
		if staleModel || existing[c.ID] != c.Text {
			toEmbed = append(toEmbed, c)
		}
	}
	if len(toEmbed) > 0 {
		texts := make([]string, len(toEmbed))
		for i, c := range toEmbed {
			texts[i] = c.Text
		}
		// Acquire the rate limiter only now that the diff has found real
		// work — warm sources never reach it (P3-t1) and only actual
		// provider requests are rate-limited (P3-t2).
		if err := w.limiter.Wait(ctx); err != nil {
			return false, err
		}
		vecs, err := w.client.Embed(ctx, texts, "document")
		if err != nil {
			return false, err
		}
		rows := make([]models.Embedding, len(toEmbed))
		for i, c := range toEmbed {
			rows[i] = models.Embedding{
				SourceType: ref.Type, SourceID: ref.ID, ChunkID: c.ID, ChunkText: c.Text,
				Embedding: vecs[i], Project: project, SourceTitle: title, Model: w.client.Model(),
			}
		}
		if err := w.store.UpsertEmbeddings(ctx, rows); err != nil {
			return false, err
		}
		embedded = true
	}

	keep := make([]string, len(chunks))
	for i, c := range chunks {
		keep[i] = c.ID
	}
	return embedded, w.store.DeleteEmbeddingsExcept(ctx, ref.Type, ref.ID, keep)
}

// load fetches the source model + its title/project for the embedding row.
// A deleted source (ErrNotFound) yields (nil, "", nil, nil) — nothing to
// embed, not an error. `title` is the primary label SearchHit uses per type.
func (w *Worker) load(ctx context.Context, ref SourceRef) (src any, title string, project *string, err error) {
	switch ref.Type {
	case "documents":
		d, e := w.store.GetDocument(ctx, ref.ID)
		if e != nil {
			return skip(e)
		}
		return d, d.Title, d.Project, nil
	case "decisions":
		d, e := w.store.GetDecision(ctx, ref.ID)
		if e != nil {
			return skip(e)
		}
		return d, d.Title, d.Project, nil
	case "snippets":
		sn, e := w.store.GetSnippet(ctx, ref.ID)
		if e != nil {
			return skip(e)
		}
		return sn, sn.Title, sn.Project, nil
	case "solutions":
		so, e := w.store.GetSolution(ctx, ref.ID)
		if e != nil {
			return skip(e)
		}
		return so, so.ErrorDescription, so.Project, nil
	case "journal":
		j, e := w.store.GetJournalEntry(ctx, ref.ID)
		if e != nil {
			return skip(e)
		}
		return j, j.Summary, j.Project, nil
	}
	return nil, "", nil, nil
}

// skip turns ErrNotFound into a no-op (deleted source) and wraps anything else.
func skip(err error) (any, string, *string, error) {
	if errors.Is(err, store.ErrNotFound) {
		return nil, "", nil, nil
	}
	return nil, "", nil, fmt.Errorf("load source: %w", err)
}
