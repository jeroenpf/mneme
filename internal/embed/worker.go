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

// Worker embeds sources off an in-memory channel. Jobs are best-effort:
// failures are logged and recovered by the next ReconcileAll.
type Worker struct {
	store       store.Store
	client      Client
	ch          chan SourceRef
	minInterval time.Duration // >0 throttles Run to stay under a low RPM tier
}

// NewWorker builds the worker. rpm>0 sets a proactive inter-job delay
// (60s/rpm) in Run for accounts on the low no-payment-method tier; rpm=0
// relies solely on the client's 429 backoff.
func NewWorker(st store.Store, c Client, buf, rpm int) *Worker {
	var interval time.Duration
	if rpm > 0 {
		interval = time.Minute / time.Duration(rpm)
	}
	return &Worker{store: st, client: c, ch: make(chan SourceRef, buf), minInterval: interval}
}

// Enqueue is non-blocking: a full buffer drops the job (the next reconcile
// re-enqueues it) so a slow Voyage never stalls an MCP tool.
func (w *Worker) Enqueue(ref SourceRef) {
	select {
	case w.ch <- ref:
	default:
		slog.Warn("embed queue full, dropping (reconcile will recover)", "type", ref.Type, "id", ref.ID)
	}
}

// Run drains the queue until ctx is cancelled, optionally throttled to
// minInterval between jobs (the client's 429 backoff is the real safety net;
// this just avoids bursting on a low tier).
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ref := <-w.ch:
			if err := w.Process(ctx, ref); err != nil {
				slog.Error("embed failed", "type", ref.Type, "id", ref.ID, "err", err)
			}
			if w.minInterval > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(w.minInterval):
				}
			}
		}
	}
}

// ReconcileAll enqueues every source; the per-job chunk_text diff makes a
// warm DB cheap (no Voyage calls when nothing changed).
func (w *Worker) ReconcileAll(ctx context.Context) error {
	refs, err := w.store.SourceRefs(ctx)
	if err != nil {
		return err
	}
	for _, r := range refs {
		w.Enqueue(SourceRef{Type: r.Type, ID: r.ID})
	}
	return nil
}

// Process embeds a single source: chunk → diff → embed changed → upsert →
// prune stale chunks. Exported for direct testing.
func (w *Worker) Process(ctx context.Context, ref SourceRef) error {
	src, title, project, err := w.load(ctx, ref)
	if err != nil || src == nil {
		return err
	}
	chunks := Chunks(src)
	existing, err := w.store.EmbeddingsFor(ctx, ref.Type, ref.ID)
	if err != nil {
		return err
	}

	var toEmbed []Chunk
	for _, c := range chunks {
		if existing[c.ID] != c.Text {
			toEmbed = append(toEmbed, c)
		}
	}
	if len(toEmbed) > 0 {
		texts := make([]string, len(toEmbed))
		for i, c := range toEmbed {
			texts[i] = c.Text
		}
		vecs, err := w.client.Embed(ctx, texts, "document")
		if err != nil {
			return err
		}
		rows := make([]models.Embedding, len(toEmbed))
		for i, c := range toEmbed {
			rows[i] = models.Embedding{
				SourceType: ref.Type, SourceID: ref.ID, ChunkID: c.ID, ChunkText: c.Text,
				Embedding: vecs[i], Project: project, SourceTitle: title, Model: w.client.Model(),
			}
		}
		if err := w.store.UpsertEmbeddings(ctx, rows); err != nil {
			return err
		}
	}

	keep := make([]string, len(chunks))
	for i, c := range chunks {
		keep[i] = c.ID
	}
	return w.store.DeleteEmbeddingsExcept(ctx, ref.Type, ref.ID, keep)
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
